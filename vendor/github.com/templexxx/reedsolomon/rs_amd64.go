package reedsolomon

import (
	"errors"
	"sync"

	"github.com/templexxx/cpufeat"
)

// SIMD Instruction Extensions
const (
	none = iota
	avx2
	ssse3
)

var extension = none

func init() {
	getEXT()
}

func getEXT() {
	if cpufeat.X86.HasAVX2 {
		extension = avx2
		return
	} else if cpufeat.X86.HasSSSE3 {
		extension = ssse3
		return
	} else {
		extension = none
		return
	}
}

//go:noescape
func copy32B(dst, src []byte) // Need SSE2(introduced in 2001)

func initTbl(g matrix, rows, cols int, tbl []byte) {
	off := 0
	for i := 0; i < cols; i++ {
		for j := 0; j < rows; j++ {
			c := g[j*cols+i]
			t := lowhighTbl[c][:]
			copy32B(tbl[off:off+32], t)
			off += 32
		}
	}
}

// At most 3060 inverse matrix (when data=14, parity=4, calc by mathtool/cntinverse)
// In practice,  data usually below 12, parity below 5
func okCache(data, parity int) bool {
	if data < 15 && parity < 5 { // you can change it, but the data+parity can't be bigger than 32 (tips: see the codes about make inverse matrix)
		return true
	}
	return false
}

type (
	encSSSE3 encSIMD
	encAVX2  encSIMD
	encSIMD  struct {
		data   int
		parity int
		encode matrix
		gen    matrix
		tbl    []byte
		// inverse matrix cache is design for small vect size ( < 4KB )
		// it will save time for calculating inverse matrix
		// but it's not so important for big vect size
		enableCache  bool
		inverseCache iCache
	}
	iCache struct {
		sync.RWMutex
		data map[uint32][]byte
	}
)

func newRS(d, p int, em matrix) (enc Encoder) {
	g := em[d*d:]
	if extension == none {
		return &encBase{data: d, parity: p, encode: em, gen: g}
	}
	t := make([]byte, d*p*32)
	initTbl(g, p, d, t)
	ok := okCache(d, p)
	if extension == avx2 {
		e := &encAVX2{data: d, parity: p, encode: em, gen: g, tbl: t, enableCache: ok,
			inverseCache: iCache{data: make(map[uint32][]byte)}}
		return e
	}
	e := &encSSSE3{data: d, parity: p, encode: em, gen: g, tbl: t, enableCache: ok,
		inverseCache: iCache{data: make(map[uint32][]byte)}}
	return e
}

// Size of sub-vector
const unit int = 16 * 1024

func getDo(n int) int {
	if n < unit {
		c := n >> 4
		if c == 0 {
			return unit
		}
		return c << 4
	}
	return unit
}

func (e *encAVX2) Encode(vects [][]byte) (err error) {
	d := e.data
	p := e.parity
	size, err := checkEnc(d, p, vects)
	if err != nil {
		return
	}
	dv := vects[:d]
	pv := vects[d:]
	start, end := 0, 0
	do := getDo(size)
	for start < size {
		end = start + do
		if end <= size {
			e.matrixMul(start, end, dv, pv)
			start = end
		} else {
			e.matrixMulRemain(start, size, dv, pv)
			start = size
		}
	}
	return
}

//go:noescape
func mulVectAVX2(tbl, d, p []byte)

//go:noescape
func mulVectAddAVX2(tbl, d, p []byte)

func (e *encAVX2) matrixMul(start, end int, dv, pv [][]byte) {
	d := e.data
	p := e.parity
	tbl := e.tbl
	off := 0
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			t := tbl[off : off+32]
			if i != 0 {
				mulVectAddAVX2(t, dv[i][start:end], pv[j][start:end])
			} else {
				mulVectAVX2(t, dv[0][start:end], pv[j][start:end])
			}
			off += 32
		}
	}
}

func (e *encAVX2) matrixMulRemain(start, end int, dv, pv [][]byte) {
	undone := end - start
	do := (undone >> 4) << 4
	d := e.data
	p := e.parity
	tbl := e.tbl
	if do >= 16 {
		end2 := start + do
		off := 0
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				t := tbl[off : off+32]
				if i != 0 {
					mulVectAddAVX2(t, dv[i][start:end2], pv[j][start:end2])
				} else {
					mulVectAVX2(t, dv[0][start:end2], pv[j][start:end2])
				}
				off += 32
			}
		}
		start = end
	}
	if undone > do {
		// may recalculate some data, but still improve a lot
		start2 := end - 16
		if start2 >= 0 {
			off := 0
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					t := tbl[off : off+32]
					if i != 0 {
						mulVectAddAVX2(t, dv[i][start2:end], pv[j][start2:end])
					} else {
						mulVectAVX2(t, dv[0][start2:end], pv[j][start2:end])
					}
					off += 32
				}
			}
		} else {
			g := e.gen
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					if i != 0 {
						mulVectAdd(g[j*d+i], dv[i][start:], pv[j][start:])
					} else {
						mulVect(g[j*d], dv[0][start:], pv[j][start:])
					}
				}
			}
		}
	}
}

// use generator-matrix but not tbls for encoding
// it's design for reconstructing
// for small vects, it cost to much time on initTbl, so drop it
// and for big vects, the tbls can't impact much, because the cache will be filled with vects' data
func (e *encAVX2) encodeGen(vects [][]byte) (err error) {
	d := e.data
	p := e.parity
	size, err := checkEnc(d, p, vects)
	if err != nil {
		return
	}
	dv := vects[:d]
	pv := vects[d:]
	start, end := 0, 0
	do := getDo(size)
	for start < size {
		end = start + do
		if end <= size {
			e.matrixMulGen(start, end, dv, pv)
			start = end
		} else {
			e.matrixMulRemainGen(start, size, dv, pv)
			start = size
		}
	}
	return
}

func (e *encAVX2) matrixMulGen(start, end int, dv, pv [][]byte) {
	d := e.data
	p := e.parity
	g := e.gen
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			t := lowhighTbl[g[j*d+i]][:]
			if i != 0 {
				mulVectAddAVX2(t, dv[i][start:end], pv[j][start:end])
			} else {
				mulVectAVX2(t, dv[0][start:end], pv[j][start:end])
			}
		}
	}
}

func (e *encAVX2) matrixMulRemainGen(start, end int, dv, pv [][]byte) {
	undone := end - start
	do := (undone >> 4) << 4
	d := e.data
	p := e.parity
	g := e.gen
	if do >= 16 {
		end2 := start + do
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				t := lowhighTbl[g[j*d+i]][:]
				if i != 0 {
					mulVectAddAVX2(t, dv[i][start:end2], pv[j][start:end2])
				} else {
					mulVectAVX2(t, dv[0][start:end2], pv[j][start:end2])
				}
			}
		}
		start = end
	}
	if undone > do {
		start2 := end - 16
		if start2 >= 0 {
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					t := lowhighTbl[g[j*d+i]][:]
					if i != 0 {
						mulVectAddAVX2(t, dv[i][start2:end], pv[j][start2:end])
					} else {
						mulVectAVX2(t, dv[0][start2:end], pv[j][start2:end])
					}
				}
			}
		} else {
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					if i != 0 {
						mulVectAdd(g[j*d+i], dv[i][start:], pv[j][start:])
					} else {
						mulVect(g[j*d], dv[0][start:], pv[j][start:])
					}
				}
			}
		}
	}
}

func (e *encAVX2) Reconstruct(vects [][]byte) (err error) {
	return e.reconstruct(vects, false)
}

func (e *encAVX2) ReconstructData(vects [][]byte) (err error) {
	return e.reconstruct(vects, true)
}

func (e *encAVX2) ReconstWithPos(vects [][]byte, has, dLost, pLost []int) error {
	return e.reconstWithPos(vects, has, dLost, pLost, false)
}

func (e *encAVX2) ReconstDataWithPos(vects [][]byte, has, dLost []int) error {
	return e.reconstWithPos(vects, has, dLost, nil, true)
}

func (e *encAVX2) makeGen(has, dLost []int) (gen []byte, err error) {
	d := e.data
	em := e.encode
	cnt := len(dLost)
	if !e.enableCache {
		matrixbuf := make([]byte, 4*d*d+cnt*d)
		m := matrixbuf[:d*d]
		for i, l := range has {
			copy(m[i*d:i*d+d], em[l*d:l*d+d])
		}
		raw := matrixbuf[d*d : 3*d*d]
		im := matrixbuf[3*d*d : 4*d*d]
		err2 := matrix(m).invert(raw, d, im)
		if err2 != nil {
			return nil, err2
		}
		g := matrixbuf[4*d*d:]
		for i, l := range dLost {
			copy(g[i*d:i*d+d], im[l*d:l*d+d])
		}
		return g, nil
	}
	var ikey uint32
	for _, p := range has {
		ikey += 1 << uint8(p)
	}
	e.inverseCache.RLock()
	v, ok := e.inverseCache.data[ikey]
	if ok {
		im := v
		g := make([]byte, cnt*d)
		for i, l := range dLost {
			copy(g[i*d:i*d+d], im[l*d:l*d+d])
		}
		e.inverseCache.RUnlock()
		return g, nil
	}
	e.inverseCache.RUnlock()
	matrixbuf := make([]byte, 4*d*d+cnt*d)
	m := matrixbuf[:d*d]
	for i, l := range has {
		copy(m[i*d:i*d+d], em[l*d:l*d+d])
	}
	raw := matrixbuf[d*d : 3*d*d]
	im := matrixbuf[3*d*d : 4*d*d]
	err2 := matrix(m).invert(raw, d, im)
	if err2 != nil {
		return nil, err2
	}
	e.inverseCache.Lock()
	e.inverseCache.data[ikey] = im
	e.inverseCache.Unlock()
	g := matrixbuf[4*d*d:]
	for i, l := range dLost {
		copy(g[i*d:i*d+d], im[l*d:l*d+d])
	}
	return g, nil
}

func (e *encAVX2) reconst(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
	d := e.data
	em := e.encode
	dCnt := len(dLost)
	size := len(vects[has[0]])
	if dCnt != 0 {
		vtmp := make([][]byte, d+dCnt)
		for i, p := range has {
			vtmp[i] = vects[p]
		}
		for i, p := range dLost {
			if len(vects[p]) == 0 {
				vects[p] = make([]byte, size)
			}
			vtmp[i+d] = vects[p]
		}
		g, err2 := e.makeGen(has, dLost)
		if err2 != nil {
			return
		}
		etmp := &encAVX2{data: d, parity: dCnt, gen: g}
		err2 = etmp.encodeGen(vtmp)
		if err2 != nil {
			return err2
		}
	}
	if dataOnly {
		return
	}
	pCnt := len(pLost)
	if pCnt != 0 {
		g := make([]byte, pCnt*d)
		for i, l := range pLost {
			copy(g[i*d:i*d+d], em[l*d:l*d+d])
		}
		vtmp := make([][]byte, d+pCnt)
		for i := 0; i < d; i++ {
			vtmp[i] = vects[i]
		}
		for i, p := range pLost {
			if len(vects[p]) == 0 {
				vects[p] = make([]byte, size)
			}
			vtmp[i+d] = vects[p]
		}
		etmp := &encAVX2{data: d, parity: pCnt, gen: g}
		err2 := etmp.encodeGen(vtmp)
		if err2 != nil {
			return err2
		}
	}
	return
}

func (e *encAVX2) reconstWithPos(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	if len(has) != d {
		return errors.New("rs.Reconst: not enough vects")
	}
	dCnt := len(dLost)
	if dCnt > p {
		return errors.New("rs.Reconst: not enough vects")
	}
	pCnt := len(pLost)
	if pCnt > p {
		return errors.New("rs.Reconst: not enough vects")
	}
	return e.reconst(vects, has, dLost, pLost, dataOnly)
}

func (e *encAVX2) reconstruct(vects [][]byte, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	t := d + p
	listBuf := make([]int, t+p)
	has := listBuf[:d]
	dLost := listBuf[d:t]
	pLost := listBuf[t : t+p]
	hasCnt, dCnt, pCnt := 0, 0, 0
	for i := 0; i < t; i++ {
		if vects[i] != nil {
			if hasCnt < d {
				has[hasCnt] = i
				hasCnt++
			}
		} else {
			if i < d {
				if dCnt < p {
					dLost[dCnt] = i
					dCnt++
				} else {
					return errors.New("rs.Reconst: not enough vects")
				}
			} else {
				if pCnt < p {
					pLost[pCnt] = i
					pCnt++
				} else {
					return errors.New("rs.Reconst: not enough vects")
				}
			}
		}
	}
	if hasCnt != d {
		return errors.New("rs.Reconst: not enough vects")
	}
	dLost = dLost[:dCnt]
	pLost = pLost[:pCnt]
	return e.reconst(vects, has, dLost, pLost, dataOnly)
}

func (e *encSSSE3) Encode(vects [][]byte) (err error) {
	d := e.data
	p := e.parity
	size, err := checkEnc(d, p, vects)
	if err != nil {
		return
	}
	dv := vects[:d]
	pv := vects[d:]
	start, end := 0, 0
	do := getDo(size)
	for start < size {
		end = start + do
		if end <= size {
			e.matrixMul(start, end, dv, pv)
			start = end
		} else {
			e.matrixMulRemain(start, size, dv, pv)
			start = size
		}
	}
	return
}

//go:noescape
func mulVectSSSE3(tbl, d, p []byte)

//go:noescape
func mulVectAddSSSE3(tbl, d, p []byte)

func (e *encSSSE3) matrixMul(start, end int, dv, pv [][]byte) {
	d := e.data
	p := e.parity
	tbl := e.tbl
	off := 0
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			t := tbl[off : off+32]
			if i != 0 {
				mulVectAddSSSE3(t, dv[i][start:end], pv[j][start:end])
			} else {
				mulVectSSSE3(t, dv[0][start:end], pv[j][start:end])
			}
			off += 32
		}
	}
}

func (e *encSSSE3) matrixMulRemain(start, end int, dv, pv [][]byte) {
	undone := end - start
	do := (undone >> 4) << 4
	d := e.data
	p := e.parity
	tbl := e.tbl
	if do >= 16 {
		end2 := start + do
		off := 0
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				t := tbl[off : off+32]
				if i != 0 {
					mulVectAddSSSE3(t, dv[i][start:end2], pv[j][start:end2])
				} else {
					mulVectSSSE3(t, dv[0][start:end2], pv[j][start:end2])
				}
				off += 32
			}
		}
		start = end
	}
	if undone > do {
		start2 := end - 16
		if start2 >= 0 {
			off := 0
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					t := tbl[off : off+32]
					if i != 0 {
						mulVectAddSSSE3(t, dv[i][start2:end], pv[j][start2:end])
					} else {
						mulVectSSSE3(t, dv[0][start2:end], pv[j][start2:end])
					}
					off += 32
				}
			}
		} else {
			g := e.gen
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					if i != 0 {
						mulVectAdd(g[j*d+i], dv[i][start:], pv[j][start:])
					} else {
						mulVect(g[j*d], dv[0][start:], pv[j][start:])
					}
				}
			}
		}
	}
}

// use generator-matrix but not tbls for encoding
// it's design for reconstructing
// for small vects, it cost to much time on initTbl, so drop it
// and for big vects, the tbls can't impact much, because the cache will be filled with vects' data
func (e *encSSSE3) encodeGen(vects [][]byte) (err error) {
	d := e.data
	p := e.parity
	size, err := checkEnc(d, p, vects)
	if err != nil {
		return
	}
	dv := vects[:d]
	pv := vects[d:]
	start, end := 0, 0
	do := getDo(size)
	for start < size {
		end = start + do
		if end <= size {
			e.matrixMulGen(start, end, dv, pv)
			start = end
		} else {
			e.matrixMulRemainGen(start, size, dv, pv)
			start = size
		}
	}
	return
}

func (e *encSSSE3) matrixMulGen(start, end int, dv, pv [][]byte) {
	d := e.data
	p := e.parity
	g := e.gen
	for i := 0; i < d; i++ {
		for j := 0; j < p; j++ {
			t := lowhighTbl[g[j*d+i]][:]
			if i != 0 {
				mulVectAddSSSE3(t, dv[i][start:end], pv[j][start:end])
			} else {
				mulVectSSSE3(t, dv[0][start:end], pv[j][start:end])
			}
		}
	}
}

func (e *encSSSE3) matrixMulRemainGen(start, end int, dv, pv [][]byte) {
	undone := end - start
	do := (undone >> 4) << 4
	d := e.data
	p := e.parity
	g := e.gen
	if do >= 16 {
		end2 := start + do
		for i := 0; i < d; i++ {
			for j := 0; j < p; j++ {
				t := lowhighTbl[g[j*d+i]][:]
				if i != 0 {
					mulVectAddSSSE3(t, dv[i][start:end2], pv[j][start:end2])
				} else {
					mulVectSSSE3(t, dv[0][start:end2], pv[j][start:end2])
				}
			}
		}
		start = end
	}
	if undone > do {
		start2 := end - 16
		if start2 >= 0 {
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					t := lowhighTbl[g[j*d+i]][:]
					if i != 0 {
						mulVectAddSSSE3(t, dv[i][start2:end], pv[j][start2:end])
					} else {
						mulVectSSSE3(t, dv[0][start2:end], pv[j][start2:end])
					}
				}
			}
		} else {
			for i := 0; i < d; i++ {
				for j := 0; j < p; j++ {
					if i != 0 {
						mulVectAdd(g[j*d+i], dv[i][start:], pv[j][start:])
					} else {
						mulVect(g[j*d], dv[0][start:], pv[j][start:])
					}
				}
			}
		}
	}
}

func (e *encSSSE3) Reconstruct(vects [][]byte) (err error) {
	return e.reconstruct(vects, false)
}

func (e *encSSSE3) ReconstructData(vects [][]byte) (err error) {
	return e.reconstruct(vects, true)
}

func (e *encSSSE3) ReconstWithPos(vects [][]byte, has, dLost, pLost []int) error {
	return e.reconstWithPos(vects, has, dLost, pLost, false)
}

func (e *encSSSE3) ReconstDataWithPos(vects [][]byte, has, dLost []int) error {
	return e.reconstWithPos(vects, has, dLost, nil, true)
}

func (e *encSSSE3) makeGen(has, dLost []int) (gen []byte, err error) {
	d := e.data
	em := e.encode
	cnt := len(dLost)
	if !e.enableCache {
		matrixbuf := make([]byte, 4*d*d+cnt*d)
		m := matrixbuf[:d*d]
		for i, l := range has {
			copy(m[i*d:i*d+d], em[l*d:l*d+d])
		}
		raw := matrixbuf[d*d : 3*d*d]
		im := matrixbuf[3*d*d : 4*d*d]
		err2 := matrix(m).invert(raw, d, im)
		if err2 != nil {
			return nil, err2
		}
		g := matrixbuf[4*d*d:]
		for i, l := range dLost {
			copy(g[i*d:i*d+d], im[l*d:l*d+d])
		}
		return g, nil
	}
	var ikey uint32
	for _, p := range has {
		ikey += 1 << uint8(p)
	}
	e.inverseCache.RLock()
	v, ok := e.inverseCache.data[ikey]
	if ok {
		im := v
		g := make([]byte, cnt*d)
		for i, l := range dLost {
			copy(g[i*d:i*d+d], im[l*d:l*d+d])
		}
		e.inverseCache.RUnlock()
		return g, nil
	}
	e.inverseCache.RUnlock()
	matrixbuf := make([]byte, 4*d*d+cnt*d)
	m := matrixbuf[:d*d]
	for i, l := range has {
		copy(m[i*d:i*d+d], em[l*d:l*d+d])
	}
	raw := matrixbuf[d*d : 3*d*d]
	im := matrixbuf[3*d*d : 4*d*d]
	err2 := matrix(m).invert(raw, d, im)
	if err2 != nil {
		return nil, err2
	}
	e.inverseCache.Lock()
	e.inverseCache.data[ikey] = im
	e.inverseCache.Unlock()
	g := matrixbuf[4*d*d:]
	for i, l := range dLost {
		copy(g[i*d:i*d+d], im[l*d:l*d+d])
	}
	return g, nil
}

func (e *encSSSE3) reconst(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
	d := e.data
	em := e.encode
	dCnt := len(dLost)
	size := len(vects[has[0]])
	if dCnt != 0 {
		vtmp := make([][]byte, d+dCnt)
		for i, p := range has {
			vtmp[i] = vects[p]
		}
		for i, p := range dLost {
			if len(vects[p]) == 0 {
				vects[p] = make([]byte, size)
			}
			vtmp[i+d] = vects[p]
		}
		g, err2 := e.makeGen(has, dLost)
		if err2 != nil {
			return
		}
		etmp := &encSSSE3{data: d, parity: dCnt, gen: g}
		err2 = etmp.encodeGen(vtmp)
		if err2 != nil {
			return err2
		}
	}
	if dataOnly {
		return
	}
	pCnt := len(pLost)
	if pCnt != 0 {
		g := make([]byte, pCnt*d)
		for i, l := range pLost {
			copy(g[i*d:i*d+d], em[l*d:l*d+d])
		}
		vtmp := make([][]byte, d+pCnt)
		for i := 0; i < d; i++ {
			vtmp[i] = vects[i]
		}
		for i, p := range pLost {
			if len(vects[p]) == 0 {
				vects[p] = make([]byte, size)
			}
			vtmp[i+d] = vects[p]
		}
		etmp := &encSSSE3{data: d, parity: pCnt, gen: g}
		err2 := etmp.encodeGen(vtmp)
		if err2 != nil {
			return err2
		}
	}
	return
}

func (e *encSSSE3) reconstWithPos(vects [][]byte, has, dLost, pLost []int, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	if len(has) != d {
		return errors.New("rs.Reconst: not enough vects")
	}
	dCnt := len(dLost)
	if dCnt > p {
		return errors.New("rs.Reconst: not enough vects")
	}
	pCnt := len(pLost)
	if pCnt > p {
		return errors.New("rs.Reconst: not enough vects")
	}
	return e.reconst(vects, has, dLost, pLost, dataOnly)
}

func (e *encSSSE3) reconstruct(vects [][]byte, dataOnly bool) (err error) {
	d := e.data
	p := e.parity
	t := d + p
	listBuf := make([]int, t+p)
	has := listBuf[:d]
	dLost := listBuf[d:t]
	pLost := listBuf[t : t+p]
	hasCnt, dCnt, pCnt := 0, 0, 0
	for i := 0; i < t; i++ {
		if vects[i] != nil {
			if hasCnt < d {
				has[hasCnt] = i
				hasCnt++
			}
		} else {
			if i < d {
				if dCnt < p {
					dLost[dCnt] = i
					dCnt++
				} else {
					return errors.New("rs.Reconst: not enough vects")
				}
			} else {
				if pCnt < p {
					pLost[pCnt] = i
					pCnt++
				} else {
					return errors.New("rs.Reconst: not enough vects")
				}
			}
		}
	}
	if hasCnt != d {
		return errors.New("rs.Reconst: not enough vects")
	}
	dLost = dLost[:dCnt]
	pLost = pLost[:pCnt]
	return e.reconst(vects, has, dLost, pLost, dataOnly)
}
