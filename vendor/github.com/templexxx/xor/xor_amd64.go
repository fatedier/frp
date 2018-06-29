package xor

import "github.com/templexxx/cpufeat"

func init() {
	getEXT()
}

func getEXT() {
	if cpufeat.X86.HasAVX2 {
		extension = avx2
	} else {
		extension = sse2
	}
	return
}

func xorBytes(dst, src0, src1 []byte, size int) {
	switch extension {
	case avx2:
		bytesAVX2(dst, src0, src1, size)
	default:
		bytesSSE2(dst, src0, src1, size)
	}
}

// non-temporal hint store
const nontmp = 8 * 1024
const avx2loopsize = 128

func bytesAVX2(dst, src0, src1 []byte, size int) {
	if size < avx2loopsize {
		bytesAVX2mini(dst, src0, src1, size)
	} else if size >= avx2loopsize && size <= nontmp {
		bytesAVX2small(dst, src0, src1, size)
	} else {
		bytesAVX2big(dst, src0, src1, size)
	}
}

const sse2loopsize = 64

func bytesSSE2(dst, src0, src1 []byte, size int) {
	if size < sse2loopsize {
		bytesSSE2mini(dst, src0, src1, size)
	} else if size >= sse2loopsize && size <= nontmp {
		bytesSSE2small(dst, src0, src1, size)
	} else {
		bytesSSE2big(dst, src0, src1, size)
	}
}

func xorMatrix(dst []byte, src [][]byte) {
	switch extension {
	case avx2:
		matrixAVX2(dst, src)
	default:
		matrixSSE2(dst, src)
	}
}

func matrixAVX2(dst []byte, src [][]byte) {
	size := len(dst)
	if size > nontmp {
		matrixAVX2big(dst, src)
	} else {
		matrixAVX2small(dst, src)
	}
}

func matrixSSE2(dst []byte, src [][]byte) {
	size := len(dst)
	if size > nontmp {
		matrixSSE2big(dst, src)
	} else {
		matrixSSE2small(dst, src)
	}
}

//go:noescape
func xorSrc0(dst, src0, src1 []byte)

//go:noescape
func xorSrc1(dst, src0, src1 []byte)

//go:noescape
func bytesAVX2mini(dst, src0, src1 []byte, size int)

//go:noescape
func bytesAVX2big(dst, src0, src1 []byte, size int)

//go:noescape
func bytesAVX2small(dst, src0, src1 []byte, size int)

//go:noescape
func bytesSSE2mini(dst, src0, src1 []byte, size int)

//go:noescape
func bytesSSE2small(dst, src0, src1 []byte, size int)

//go:noescape
func bytesSSE2big(dst, src0, src1 []byte, size int)

//go:noescape
func matrixAVX2small(dst []byte, src [][]byte)

//go:noescape
func matrixAVX2big(dst []byte, src [][]byte)

//go:noescape
func matrixSSE2small(dst []byte, src [][]byte)

//go:noescape
func matrixSSE2big(dst []byte, src [][]byte)

//go:noescape
func hasAVX2() bool

//go:noescape
func hasSSE2() bool
