//+build !amd64 noasm appengine gccgo
//+build !arm64 noasm appengine gccgo
//+build !ppc64le noasm appengine gccgo

// Copyright 2015, Klaus Post, see LICENSE for details.

package reedsolomon

func galMulSlice(c byte, in, out []byte, o *options) {
	mt := mulTable[c][:256]
	out = out[:len(in)]
	for n, input := range in {
		out[n] = mt[input]
	}
}

func galMulSliceXor(c byte, in, out []byte, o *options) {
	mt := mulTable[c][:256]
	out = out[:len(in)]
	for n, input := range in {
		out[n] ^= mt[input]
	}
}

// slice galois add
func sliceXor(in, out []byte, sse2 bool) {
	for n, input := range in {
		out[n] ^= input
	}
}

func (r reedSolomon) codeSomeShardsAvx512(matrixRows, inputs, outputs [][]byte, outputCount, byteCount int) {
	panic("unreachable")
}
