//+build !amd64 noasm appengine

// Copyright 2015, Klaus Post, see LICENSE for details.

package reedsolomon

func galMulSlice(c byte, in, out []byte, ssse3, avx2 bool) {
	mt := mulTable[c]
	for n, input := range in {
		out[n] = mt[input]
	}
}

func galMulSliceXor(c byte, in, out []byte, ssse3, avx2 bool) {
	mt := mulTable[c]
	for n, input := range in {
		out[n] ^= mt[input]
	}
}
