//+build !noasm
//+build !appengine

// Copyright 2015, Klaus Post, see LICENSE for details.

package reedsolomon

//go:noescape
func galMulSSSE3(low, high, in, out []byte)

//go:noescape
func galMulSSSE3Xor(low, high, in, out []byte)

//go:noescape
func galMulAVX2Xor(low, high, in, out []byte)

//go:noescape
func galMulAVX2(low, high, in, out []byte)

// This is what the assembler rountes does in blocks of 16 bytes:
/*
func galMulSSSE3(low, high, in, out []byte) {
	for n, input := range in {
		l := input & 0xf
		h := input >> 4
		out[n] = low[l] ^ high[h]
	}
}

func galMulSSSE3Xor(low, high, in, out []byte) {
	for n, input := range in {
		l := input & 0xf
		h := input >> 4
		out[n] ^= low[l] ^ high[h]
	}
}
*/

func galMulSlice(c byte, in, out []byte, ssse3, avx2 bool) {
	var done int
	if avx2 {
		galMulAVX2(mulTableLow[c][:], mulTableHigh[c][:], in, out)
		done = (len(in) >> 5) << 5
	} else if ssse3 {
		galMulSSSE3(mulTableLow[c][:], mulTableHigh[c][:], in, out)
		done = (len(in) >> 4) << 4
	}
	remain := len(in) - done
	if remain > 0 {
		mt := mulTable[c]
		for i := done; i < len(in); i++ {
			out[i] = mt[in[i]]
		}
	}
}

func galMulSliceXor(c byte, in, out []byte, ssse3, avx2 bool) {
	var done int
	if avx2 {
		galMulAVX2Xor(mulTableLow[c][:], mulTableHigh[c][:], in, out)
		done = (len(in) >> 5) << 5
	} else if ssse3 {
		galMulSSSE3Xor(mulTableLow[c][:], mulTableHigh[c][:], in, out)
		done = (len(in) >> 4) << 4
	}
	remain := len(in) - done
	if remain > 0 {
		mt := mulTable[c]
		for i := done; i < len(in); i++ {
			out[i] ^= mt[in[i]]
		}
	}
}
