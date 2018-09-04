package kcp

import (
	"encoding/binary"
	"sync/atomic"

	"github.com/klauspost/reedsolomon"
)

const (
	fecHeaderSize      = 6
	fecHeaderSizePlus2 = fecHeaderSize + 2 // plus 2B data size
	typeData           = 0xf1
	typeFEC            = 0xf2
)

type (
	// fecPacket is a decoded FEC packet
	fecPacket struct {
		seqid uint32
		flag  uint16
		data  []byte
	}

	// fecDecoder for decoding incoming packets
	fecDecoder struct {
		rxlimit      int // queue size limit
		dataShards   int
		parityShards int
		shardSize    int
		rx           []fecPacket // ordered receive queue

		// caches
		decodeCache [][]byte
		flagCache   []bool

		// RS decoder
		codec reedsolomon.Encoder
	}
)

func newFECDecoder(rxlimit, dataShards, parityShards int) *fecDecoder {
	if dataShards <= 0 || parityShards <= 0 {
		return nil
	}
	if rxlimit < dataShards+parityShards {
		return nil
	}

	fec := new(fecDecoder)
	fec.rxlimit = rxlimit
	fec.dataShards = dataShards
	fec.parityShards = parityShards
	fec.shardSize = dataShards + parityShards
	enc, err := reedsolomon.New(dataShards, parityShards, reedsolomon.WithMaxGoroutines(1))
	if err != nil {
		return nil
	}
	fec.codec = enc
	fec.decodeCache = make([][]byte, fec.shardSize)
	fec.flagCache = make([]bool, fec.shardSize)
	return fec
}

// decodeBytes a fec packet
func (dec *fecDecoder) decodeBytes(data []byte) fecPacket {
	var pkt fecPacket
	pkt.seqid = binary.LittleEndian.Uint32(data)
	pkt.flag = binary.LittleEndian.Uint16(data[4:])
	// allocate memory & copy
	buf := xmitBuf.Get().([]byte)[:len(data)-6]
	copy(buf, data[6:])
	pkt.data = buf
	return pkt
}

// decode a fec packet
func (dec *fecDecoder) decode(pkt fecPacket) (recovered [][]byte) {
	// insertion
	n := len(dec.rx) - 1
	insertIdx := 0
	for i := n; i >= 0; i-- {
		if pkt.seqid == dec.rx[i].seqid { // de-duplicate
			xmitBuf.Put(pkt.data)
			return nil
		} else if _itimediff(pkt.seqid, dec.rx[i].seqid) > 0 { // insertion
			insertIdx = i + 1
			break
		}
	}

	// insert into ordered rx queue
	if insertIdx == n+1 {
		dec.rx = append(dec.rx, pkt)
	} else {
		dec.rx = append(dec.rx, fecPacket{})
		copy(dec.rx[insertIdx+1:], dec.rx[insertIdx:]) // shift right
		dec.rx[insertIdx] = pkt
	}

	// shard range for current packet
	shardBegin := pkt.seqid - pkt.seqid%uint32(dec.shardSize)
	shardEnd := shardBegin + uint32(dec.shardSize) - 1

	// max search range in ordered queue for current shard
	searchBegin := insertIdx - int(pkt.seqid%uint32(dec.shardSize))
	if searchBegin < 0 {
		searchBegin = 0
	}
	searchEnd := searchBegin + dec.shardSize - 1
	if searchEnd >= len(dec.rx) {
		searchEnd = len(dec.rx) - 1
	}

	// re-construct datashards
	if searchEnd-searchBegin+1 >= dec.dataShards {
		var numshard, numDataShard, first, maxlen int

		// zero cache
		shards := dec.decodeCache
		shardsflag := dec.flagCache
		for k := range dec.decodeCache {
			shards[k] = nil
			shardsflag[k] = false
		}

		// shard assembly
		for i := searchBegin; i <= searchEnd; i++ {
			seqid := dec.rx[i].seqid
			if _itimediff(seqid, shardEnd) > 0 {
				break
			} else if _itimediff(seqid, shardBegin) >= 0 {
				shards[seqid%uint32(dec.shardSize)] = dec.rx[i].data
				shardsflag[seqid%uint32(dec.shardSize)] = true
				numshard++
				if dec.rx[i].flag == typeData {
					numDataShard++
				}
				if numshard == 1 {
					first = i
				}
				if len(dec.rx[i].data) > maxlen {
					maxlen = len(dec.rx[i].data)
				}
			}
		}

		if numDataShard == dec.dataShards {
			// case 1:  no lost data shards
			dec.rx = dec.freeRange(first, numshard, dec.rx)
		} else if numshard >= dec.dataShards {
			// case 2: data shard lost, but  recoverable from parity shard
			for k := range shards {
				if shards[k] != nil {
					dlen := len(shards[k])
					shards[k] = shards[k][:maxlen]
					xorBytes(shards[k][dlen:], shards[k][dlen:], shards[k][dlen:])
				}
			}
			if err := dec.codec.Reconstruct(shards); err == nil {
				for k := range shards[:dec.dataShards] {
					if !shardsflag[k] {
						recovered = append(recovered, shards[k])
					}
				}
			}
			dec.rx = dec.freeRange(first, numshard, dec.rx)
		}
	}

	// keep rxlimit
	if len(dec.rx) > dec.rxlimit {
		if dec.rx[0].flag == typeData { // record unrecoverable data
			atomic.AddUint64(&DefaultSnmp.FECShortShards, 1)
		}
		dec.rx = dec.freeRange(0, 1, dec.rx)
	}
	return
}

// free a range of fecPacket, and zero for GC recycling
func (dec *fecDecoder) freeRange(first, n int, q []fecPacket) []fecPacket {
	for i := first; i < first+n; i++ { // free
		xmitBuf.Put(q[i].data)
	}
	copy(q[first:], q[first+n:])
	for i := 0; i < n; i++ { // dereference data
		q[len(q)-1-i].data = nil
	}
	return q[:len(q)-n]
}

type (
	// fecEncoder for encoding outgoing packets
	fecEncoder struct {
		dataShards   int
		parityShards int
		shardSize    int
		paws         uint32 // Protect Against Wrapped Sequence numbers
		next         uint32 // next seqid

		shardCount int // count the number of datashards collected
		maxSize    int // record maximum data length in datashard

		headerOffset  int // FEC header offset
		payloadOffset int // FEC payload offset

		// caches
		shardCache  [][]byte
		encodeCache [][]byte

		// RS encoder
		codec reedsolomon.Encoder
	}
)

func newFECEncoder(dataShards, parityShards, offset int) *fecEncoder {
	if dataShards <= 0 || parityShards <= 0 {
		return nil
	}
	fec := new(fecEncoder)
	fec.dataShards = dataShards
	fec.parityShards = parityShards
	fec.shardSize = dataShards + parityShards
	fec.paws = (0xffffffff/uint32(fec.shardSize) - 1) * uint32(fec.shardSize)
	fec.headerOffset = offset
	fec.payloadOffset = fec.headerOffset + fecHeaderSize

	enc, err := reedsolomon.New(dataShards, parityShards, reedsolomon.WithMaxGoroutines(1))
	if err != nil {
		return nil
	}
	fec.codec = enc

	// caches
	fec.encodeCache = make([][]byte, fec.shardSize)
	fec.shardCache = make([][]byte, fec.shardSize)
	for k := range fec.shardCache {
		fec.shardCache[k] = make([]byte, mtuLimit)
	}
	return fec
}

// encode the packet, output parity shards if we have enough datashards
// the content of returned parityshards will change in next encode
func (enc *fecEncoder) encode(b []byte) (ps [][]byte) {
	enc.markData(b[enc.headerOffset:])
	binary.LittleEndian.PutUint16(b[enc.payloadOffset:], uint16(len(b[enc.payloadOffset:])))

	// copy data to fec datashards
	sz := len(b)
	enc.shardCache[enc.shardCount] = enc.shardCache[enc.shardCount][:sz]
	copy(enc.shardCache[enc.shardCount], b)
	enc.shardCount++

	// record max datashard length
	if sz > enc.maxSize {
		enc.maxSize = sz
	}

	//  calculate Reed-Solomon Erasure Code
	if enc.shardCount == enc.dataShards {
		// bzero each datashard's tail
		for i := 0; i < enc.dataShards; i++ {
			shard := enc.shardCache[i]
			slen := len(shard)
			xorBytes(shard[slen:enc.maxSize], shard[slen:enc.maxSize], shard[slen:enc.maxSize])
		}

		// construct equal-sized slice with stripped header
		cache := enc.encodeCache
		for k := range cache {
			cache[k] = enc.shardCache[k][enc.payloadOffset:enc.maxSize]
		}

		// rs encode
		if err := enc.codec.Encode(cache); err == nil {
			ps = enc.shardCache[enc.dataShards:]
			for k := range ps {
				enc.markFEC(ps[k][enc.headerOffset:])
				ps[k] = ps[k][:enc.maxSize]
			}
		}

		// reset counters to zero
		enc.shardCount = 0
		enc.maxSize = 0
	}

	return
}

func (enc *fecEncoder) markData(data []byte) {
	binary.LittleEndian.PutUint32(data, enc.next)
	binary.LittleEndian.PutUint16(data[4:], typeData)
	enc.next++
}

func (enc *fecEncoder) markFEC(data []byte) {
	binary.LittleEndian.PutUint32(data, enc.next)
	binary.LittleEndian.PutUint16(data[4:], typeFEC)
	enc.next = (enc.next + 1) % enc.paws
}
