// +build !amd64 noasm

package xor

func xorBytes(dst, src0, src1 []byte, size int) {
	bytesNoSIMD(dst, src0, src1, size)
}

func xorMatrix(dst []byte, src [][]byte) {
	matrixNoSIMD(dst, src)
}

func xorSrc0(dst, src0, src1 []byte) {
	bytesNoSIMD(dst, src0, src1, len(src0))
}

func xorSrc1(dst, src0, src1 []byte) {
	bytesNoSIMD(dst, src0, src1, len(src1))
}
