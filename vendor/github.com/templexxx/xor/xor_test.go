package xor

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestVerifyBytesNoSIMD(t *testing.T) {
	for i := 1; i <= unitSize+16+2; i++ {
		if !verifyBytesNoSIMD(i) {
			t.Fatal("xor fault ", "size:", i)
		}
	}
}

func verifyBytesNoSIMD(size int) bool {
	dst := make([]byte, size)
	src0 := make([]byte, size)
	src1 := make([]byte, size)
	expect := make([]byte, size)
	rand.Seed(7)
	fillRandom(src0)
	rand.Seed(8)
	fillRandom(src1)
	for i := 0; i < size; i++ {
		expect[i] = src0[i] ^ src1[i]
	}
	xorBytes(dst, src0, src1, size)
	return bytes.Equal(expect, dst)
}

func TestVerifyBytes(t *testing.T) {
	for i := 1; i <= unitSize+16+2; i++ {
		if !verifyBytes(i) {
			t.Fatal("xor fault ", "size:", i)
		}
	}
}

func verifyBytes(size int) bool {
	dst := make([]byte, size)
	src0 := make([]byte, size)
	src1 := make([]byte, size)
	expect := make([]byte, size)
	rand.Seed(7)
	fillRandom(src0)
	rand.Seed(8)
	fillRandom(src1)
	for i := 0; i < size; i++ {
		expect[i] = src0[i] ^ src1[i]
	}
	xorBytes(dst, src0, src1, size)
	return bytes.Equal(expect, dst)
}

func TestVerifyBytesSrc1(t *testing.T) {
	for i := 1; i <= unitSize+16+2; i++ {
		if !verifyBytesSrc1(i) {
			t.Fatal("xor fault ", "size:", i)
		}
	}
}
func verifyBytesSrc1(size int) bool {
	dst := make([]byte, size)
	src0 := make([]byte, size)
	src1 := make([]byte, size)
	expect := make([]byte, size)
	rand.Seed(7)
	fillRandom(src0)
	rand.Seed(8)
	fillRandom(src1)
	for i := 0; i < size; i++ {
		expect[i] = src0[i] ^ src1[i]
	}
	xorSrc0(dst, src0, src1)
	return bytes.Equal(expect, dst)
}

func TestVerifyMatrixNoSIMD(t *testing.T) {
	for i := 1; i <= unitSize+16+2; i++ {
		if !verifyMatrixNoSIMD(i) {
			t.Fatal("xor fault ", "size:", i)
		}
	}
}

func verifyMatrixNoSIMD(size int) bool {
	numSRC := 3
	dst := make([]byte, size)
	expect := make([]byte, size)
	src := make([][]byte, numSRC)
	for i := 0; i < numSRC; i++ {
		src[i] = make([]byte, size)
		rand.Seed(int64(i))
		fillRandom(src[i])
	}
	for i := 0; i < size; i++ {
		expect[i] = src[0][i] ^ src[1][i]
	}
	for i := 2; i < numSRC; i++ {
		for j := 0; j < size; j++ {
			expect[j] ^= src[i][j]
		}
	}
	matrixNoSIMD(dst, src)
	return bytes.Equal(expect, dst)
}

func TestVerifyMatrix(t *testing.T) {
	for i := 1; i <= unitSize+16+2; i++ {
		if !verifyMatrix(i) {
			t.Fatal("xor fault ", "size:", i)
		}
	}
}

func verifyMatrix(size int) bool {
	numSRC := 3
	dst := make([]byte, size)
	expect := make([]byte, size)
	src := make([][]byte, numSRC)
	for i := 0; i < numSRC; i++ {
		src[i] = make([]byte, size)
		rand.Seed(int64(i))
		fillRandom(src[i])
	}
	for i := 0; i < size; i++ {
		expect[i] = src[0][i] ^ src[1][i]
	}
	for i := 2; i < numSRC; i++ {
		for j := 0; j < size; j++ {
			expect[j] ^= src[i][j]
		}
	}
	xorMatrix(dst, src)
	return bytes.Equal(expect, dst)
}

func BenchmarkBytesNoSIMDx12B(b *testing.B) {
	benchmarkBytesNoSIMD(b, 12)
}
func BenchmarkBytes12B(b *testing.B) {
	benchmarkBytesMini(b, 12)
}
func BenchmarkBytesNoSIMD16B(b *testing.B) {
	benchmarkBytesNoSIMD(b, 16)
}
func BenchmarkBytes16B(b *testing.B) {
	benchmarkBytesMini(b, 16)
}
func BenchmarkBytesNoSIMD24B(b *testing.B) {
	benchmarkBytesNoSIMD(b, 24)
}
func BenchmarkBytes24B(b *testing.B) {
	benchmarkBytesMini(b, 24)
}
func BenchmarkBytesNoSIMD32B(b *testing.B) {
	benchmarkBytesNoSIMD(b, 32)
}
func BenchmarkBytes32B(b *testing.B) {
	benchmarkBytesMini(b, 32)
}
func benchmarkBytesMini(b *testing.B, size int) {
	src0 := make([]byte, size)
	src1 := make([]byte, size)
	dst := make([]byte, size)
	rand.Seed(int64(0))
	fillRandom(src0)
	rand.Seed(int64(1))
	fillRandom(src1)
	BytesSrc1(dst, src0, src1)
	b.SetBytes(int64(size) * 2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BytesSrc1(dst, src0, src1)
	}
}

func BenchmarkBytesNoSIMD1K(b *testing.B) {
	benchmarkBytesNoSIMD(b, 1024)
}
func BenchmarkBytesNoSIMD16K(b *testing.B) {
	benchmarkBytesNoSIMD(b, 16*1024)
}
func BenchmarkBytesNoSIMD16M(b *testing.B) {
	benchmarkBytesNoSIMD(b, 16*1024*1024)
}
func benchmarkBytesNoSIMD(b *testing.B, size int) {
	src1 := make([]byte, size)
	src2 := make([]byte, size)
	dst := make([]byte, size)
	rand.Seed(int64(0))
	fillRandom(src1)
	rand.Seed(int64(1))
	fillRandom(src2)
	bytesNoSIMD(dst, src1, src2, size)
	b.SetBytes(int64(size) * 2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bytesNoSIMD(dst, src1, src2, size)
	}
}

func BenchmarkBytes1K(b *testing.B) {
	benchmarkBytes(b, 1024)
}
func BenchmarkBytes16K(b *testing.B) {
	benchmarkBytes(b, 16*1024)
}
func BenchmarkBytes16M(b *testing.B) {
	benchmarkBytes(b, 16*1024*1024)
}

// compare with bytes
func BenchmarkMatrix2x1K(b *testing.B) {
	benchmarkMatrix(b, 2, 1024)
}
func BenchmarkMatrix2x16K(b *testing.B) {
	benchmarkMatrix(b, 2, 16*1024)
}
func BenchmarkMatrix2x16M(b *testing.B) {
	benchmarkMatrix(b, 2, 16*1024*1024)
}
func benchmarkBytes(b *testing.B, size int) {
	src1 := make([]byte, size)
	src2 := make([]byte, size)
	dst := make([]byte, size)
	rand.Seed(int64(0))
	fillRandom(src1)
	rand.Seed(int64(1))
	fillRandom(src2)
	xorBytes(dst, src1, src2, size)
	b.SetBytes(int64(size) * 2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xorBytes(dst, src1, src2, size)
	}
}

func BenchmarkMatrixNoSIMD5x1K(b *testing.B) {
	benchmarkMatrixNoSIMD(b, 5, 1024)
}
func BenchmarkMatrixNoSIMD5x16K(b *testing.B) {
	benchmarkMatrixNoSIMD(b, 5, 16*1024)
}
func BenchmarkMatrixNoSIMD5x16M(b *testing.B) {
	benchmarkMatrixNoSIMD(b, 5, 16*1024*1024)
}
func benchmarkMatrixNoSIMD(b *testing.B, numSRC, size int) {
	src := make([][]byte, numSRC)
	dst := make([]byte, size)
	for i := 0; i < numSRC; i++ {
		rand.Seed(int64(i))
		src[i] = make([]byte, size)
		fillRandom(src[i])
	}
	matrixNoSIMD(dst, src)
	b.SetBytes(int64(size * numSRC))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matrixNoSIMD(dst, src)
	}
}

func BenchmarkMatrix5x1K(b *testing.B) {
	benchmarkMatrix(b, 5, 1024)
}
func BenchmarkMatrix5x16K(b *testing.B) {
	benchmarkMatrix(b, 5, 16*1024)
}
func BenchmarkMatrix5x16M(b *testing.B) {
	benchmarkMatrix(b, 5, 16*1024*1024)
}
func benchmarkMatrix(b *testing.B, numSRC, size int) {
	src := make([][]byte, numSRC)
	dst := make([]byte, size)
	for i := 0; i < numSRC; i++ {
		rand.Seed(int64(i))
		src[i] = make([]byte, size)
		fillRandom(src[i])
	}
	xorMatrix(dst, src)
	b.SetBytes(int64(size * numSRC))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		xorMatrix(dst, src)
	}
}

func fillRandom(p []byte) {
	for i := 0; i < len(p); i += 7 {
		val := rand.Int63()
		for j := 0; i+j < len(p) && j < 7; j++ {
			p[i+j] = byte(val)
			val >>= 8
		}
	}
}
