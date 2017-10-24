package xor

// SIMD Extensions
const (
	none = iota
	avx2
	// first introduced by Intel with the initial version of the Pentium 4 in 2001
	// so I think we can assume all amd64 has sse2
	sse2
)

var extension = none

// Bytes : chose the shortest one as xor size
// it's better to use it for big data ( > 64bytes )
func Bytes(dst, src0, src1 []byte) {
	size := len(dst)
	if size > len(src0) {
		size = len(src0)
	}
	if size > len(src1) {
		size = len(src1)
	}
	xorBytes(dst, src0, src1, size)
}

// BytesSameLen : all slice's length must be equal
// cut size branch, save time for small data
func BytesSameLen(dst, src0, src1 []byte) {
	xorSrc1(dst, src0, src1)
}

// BytesSrc0 : src1 >= src0, dst >= src0
// xor src0's len bytes
func BytesSrc0(dst, src0, src1 []byte) {
	xorSrc0(dst, src0, src1)
}

// BytesSrc1 : src0 >= src1, dst >= src1
// xor src1's len bytes
func BytesSrc1(dst, src0, src1 []byte) {
	xorSrc1(dst, src0, src1)
}

// Matrix : all slice's length must be equal && != 0
// len(src) must >= 2
func Matrix(dst []byte, src [][]byte) {
	xorMatrix(dst, src)
}
