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
	typeParity         = 0xf2
)

// fecPacket is a decoded FEC packet
type fecPacket []byte

func (bts fecPacket) seqid() uint32 { return binary.LittleEndian.Uint32(bts) }
func (bts fecPacket) flag() uint16  { return binary.LittleEndian.Uint16(bts[4:]) }
func (bts fecPacket) data() []byte  { return bts[6:] }

// fecDecoder for decoding incoming packets
type fecDecoder struct {
	rxlimit      int // queue size limit
	dataShards   int
	parityShards int
	shardSize    int
	rx           []fecPacket // ordered receive queue

	// caches
	decodeCache [][]byte
	flagCache   []bool

	// zeros
	zeros []byte

	// RS decoder
	codec reedsolomon.Encoder
}

func newFECDecoder(rxlimit, dataShards, parityShards int) *fecDecoder {
	if dataShards <= 0 || parityShards <= 0 {
		return nil
	}
	if rxlimit < dataShards+parityShards {
		return nil
	}

	dec := new(fecDecoder)
	dec.rxlimit = rxlimit
	dec.dataShards = dataShards
	dec.parityShards = parityShards
	dec.shardSize = dataShards + parityShards
	codec, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil
	}
	dec.codec = codec
	dec.decodeCache = make([][]byte, dec.shardSize)
	dec.flagCache = make([]bool, dec.shardSize)
	dec.zeros = make([]byte, mtuLimit)
	return dec
}

// decode a fec packet
func (dec *fecDecoder) decode(in fecPacket) (recovered [][]byte) {
	// insertion
	n := len(dec.rx) - 1
	insertIdx := 0
	for i := n; i >= 0; i-- {
		if in.seqid() == dec.rx[i].seqid() { // de-duplicate
			return nil
		} else if _itimediff(in.seqid(), dec.rx[i].seqid()) > 0 { // insertion
			insertIdx = i + 1
			break
		}
	}

	// make a copy
	pkt := fecPacket(xmitBuf.Get().([]byte)[:len(in)])
	copy(pkt, in)

	// insert into ordered rx queue
	if insertIdx == n+1 {
		dec.rx = append(dec.rx, pkt)
	} else {
		dec.rx = append(dec.rx, fecPacket{})
		copy(dec.rx[insertIdx+1:], dec.rx[insertIdx:]) // shift right
		dec.rx[insertIdx] = pkt
	}

	// shard range for current packet
	shardBegin := pkt.seqid() - pkt.seqid()%uint32(dec.shardSize)
	shardEnd := shardBegin + uint32(dec.shardSize) - 1

	// max search range in ordered queue for current shard
	searchBegin := insertIdx - int(pkt.seqid()%uint32(dec.shardSize))
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

		// zero caches
		shards := dec.decodeCache
		shardsflag := dec.flagCache
		for k := range dec.decodeCache {
			shards[k] = nil
			shardsflag[k] = false
		}

		// shard assembly
		for i := searchBegin; i <= searchEnd; i++ {
			seqid := dec.rx[i].seqid()
			if _itimediff(seqid, shardEnd) > 0 {
				break
			} else if _itimediff(seqid, shardBegin) >= 0 {
				shards[seqid%uint32(dec.shardSize)] = dec.rx[i].data()
				shardsflag[seqid%uint32(dec.shardSize)] = true
				numshard++
				if dec.rx[i].flag() == typeData {
					numDataShard++
				}
				if numshard == 1 {
					first = i
				}
				if len(dec.rx[i].data()) > maxlen {
					maxlen = len(dec.rx[i].data())
				}
			}
		}

		if numDataShard == dec.dataShards {
			// case 1: no loss on data shards
			dec.rx = dec.freeRange(first, numshard, dec.rx)
		} else if numshard >= dec.dataShards {
			// case 2: loss on data shards, but it's recoverable from parity shards
			for k := range shards {
				if shards[k] != nil {
					dlen := len(shards[k])
					shards[k] = shards[k][:maxlen]
					copy(shards[k][dlen:], dec.zeros)
				} else {
					shards[k] = xmitBuf.Get().([]byte)[:0]
				}
			}
			if err := dec.codec.ReconstructData(shards); err == nil {
				for k := range shards[:dec.dataShards] {
					if !shardsflag[k] {
						// recovered data should be recycled
						recovered = append(recovered, shards[k])
					}
				}
			}
			dec.rx = dec.freeRange(first, numshard, dec.rx)
		}
	}

	// keep rxlimit
	if len(dec.rx) > dec.rxlimit {
		if dec.rx[0].flag() == typeData { // track the unrecoverable data
			atomic.AddUint64(&DefaultSnmp.FECShortShards, 1)
		}
		dec.rx = dec.freeRange(0, 1, dec.rx)
	}
	return
}

// free a range of fecPacket
func (dec *fecDecoder) freeRange(first, n int, q []fecPacket) []fecPacket {
	for i := first; i < first+n; i++ { // recycle buffer
		xmitBuf.Put([]byte(q[i]))
	}

	if first == 0 && n < cap(q)/2 {
		return q[n:]
	}
	copy(q[first:], q[first+n:])
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
		maxSize    int // track maximum data length in datashard

		headerOffset  int // FEC header offset
		payloadOffset int // FEC payload offset

		// caches
		shardCache  [][]byte
		encodeCache [][]byte

		// zeros
		zeros []byte

		// RS encoder
		codec reedsolomon.Encoder
	}
)

func newFECEncoder(dataShards, parityShards, offset int) *fecEncoder {
	if dataShards <= 0 || parityShards <= 0 {
		return nil
	}
	enc := new(fecEncoder)
	enc.dataShards = dataShards
	enc.parityShards = parityShards
	enc.shardSize = dataShards + parityShards
	enc.paws = 0xffffffff / uint32(enc.shardSize) * uint32(enc.shardSize)
	enc.headerOffset = offset
	enc.payloadOffset = enc.headerOffset + fecHeaderSize

	codec, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil
	}
	enc.codec = codec

	// caches
	enc.encodeCache = make([][]byte, enc.shardSize)
	enc.shardCache = make([][]byte, enc.shardSize)
	for k := range enc.shardCache {
		enc.shardCache[k] = make([]byte, mtuLimit)
	}
	enc.zeros = make([]byte, mtuLimit)
	return enc
}

// encodes the packet, outputs parity shards if we have collected quorum datashards
// notice: the contents of 'ps' will be re-written in successive calling
func (enc *fecEncoder) encode(b []byte) (ps [][]byte) {
	// The header format:
	// | FEC SEQID(4B) | FEC TYPE(2B) | SIZE (2B) | PAYLOAD(SIZE-2) |
	// |<-headerOffset                |<-payloadOffset
	enc.markData(b[enc.headerOffset:])
	binary.LittleEndian.PutUint16(b[enc.payloadOffset:], uint16(len(b[enc.payloadOffset:])))

	// copy data from payloadOffset to fec shard cache
	sz := len(b)
	enc.shardCache[enc.shardCount] = enc.shardCache[enc.shardCount][:sz]
	copy(enc.shardCache[enc.shardCount][enc.payloadOffset:], b[enc.payloadOffset:])
	enc.shardCount++

	// track max datashard length
	if sz > enc.maxSize {
		enc.maxSize = sz
	}

	//  Generation of Reed-Solomon Erasure Code
	if enc.shardCount == enc.dataShards {
		// fill '0' into the tail of each datashard
		for i := 0; i < enc.dataShards; i++ {
			shard := enc.shardCache[i]
			slen := len(shard)
			copy(shard[slen:enc.maxSize], enc.zeros)
		}

		// construct equal-sized slice with stripped header
		cache := enc.encodeCache
		for k := range cache {
			cache[k] = enc.shardCache[k][enc.payloadOffset:enc.maxSize]
		}

		// encoding
		if err := enc.codec.Encode(cache); err == nil {
			ps = enc.shardCache[enc.dataShards:]
			for k := range ps {
				enc.markParity(ps[k][enc.headerOffset:])
				ps[k] = ps[k][:enc.maxSize]
			}
		}

		// counters resetting
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

func (enc *fecEncoder) markParity(data []byte) {
	binary.LittleEndian.PutUint32(data, enc.next)
	binary.LittleEndian.PutUint16(data[4:], typeParity)
	// sequence wrap will only happen at parity shard
	enc.next = (enc.next + 1) % enc.paws
}
