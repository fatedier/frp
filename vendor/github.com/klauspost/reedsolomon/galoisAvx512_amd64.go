//+build !noasm
//+build !appengine
//+build !gccgo

// Copyright 2015, Klaus Post, see LICENSE for details.
// Copyright 2019, Minio, Inc.

package reedsolomon

//go:noescape
func _galMulAVX512Parallel82(in, out [][]byte, matrix *[matrixSize82]byte, addTo bool)

//go:noescape
func _galMulAVX512Parallel84(in, out [][]byte, matrix *[matrixSize84]byte, addTo bool)

const (
	dimIn        = 8                            // Number of input rows processed simultaneously
	dimOut82     = 2                            // Number of output rows processed simultaneously for x2 routine
	dimOut84     = 4                            // Number of output rows processed simultaneously for x4 routine
	matrixSize82 = (16 + 16) * dimIn * dimOut82 // Dimension of slice of matrix coefficient passed into x2 routine
	matrixSize84 = (16 + 16) * dimIn * dimOut84 // Dimension of slice of matrix coefficient passed into x4 routine
)

// Construct block of matrix coefficients for 2 outputs rows in parallel
func setupMatrix82(matrixRows [][]byte, inputOffset, outputOffset int, matrix *[matrixSize82]byte) {
	offset := 0
	for c := inputOffset; c < inputOffset+dimIn; c++ {
		for iRow := outputOffset; iRow < outputOffset+dimOut82; iRow++ {
			if c < len(matrixRows[iRow]) {
				coeff := matrixRows[iRow][c]
				copy(matrix[offset*32:], mulTableLow[coeff][:])
				copy(matrix[offset*32+16:], mulTableHigh[coeff][:])
			} else {
				// coefficients not used for this input shard (so null out)
				v := matrix[offset*32 : offset*32+32]
				for i := range v {
					v[i] = 0
				}
			}
			offset += dimIn
			if offset >= dimIn*dimOut82 {
				offset -= dimIn*dimOut82 - 1
			}
		}
	}
}

// Construct block of matrix coefficients for 4 outputs rows in parallel
func setupMatrix84(matrixRows [][]byte, inputOffset, outputOffset int, matrix *[matrixSize84]byte) {
	offset := 0
	for c := inputOffset; c < inputOffset+dimIn; c++ {
		for iRow := outputOffset; iRow < outputOffset+dimOut84; iRow++ {
			if c < len(matrixRows[iRow]) {
				coeff := matrixRows[iRow][c]
				copy(matrix[offset*32:], mulTableLow[coeff][:])
				copy(matrix[offset*32+16:], mulTableHigh[coeff][:])
			} else {
				// coefficients not used for this input shard (so null out)
				v := matrix[offset*32 : offset*32+32]
				for i := range v {
					v[i] = 0
				}
			}
			offset += dimIn
			if offset >= dimIn*dimOut84 {
				offset -= dimIn*dimOut84 - 1
			}
		}
	}
}

// Invoke AVX512 routine for 2 output rows in parallel
func galMulAVX512Parallel82(in, out [][]byte, matrixRows [][]byte, inputOffset, outputOffset int) {
	done := len(in[0])
	if done == 0 {
		return
	}

	inputEnd := inputOffset + dimIn
	if inputEnd > len(in) {
		inputEnd = len(in)
	}
	outputEnd := outputOffset + dimOut82
	if outputEnd > len(out) {
		outputEnd = len(out)
	}

	matrix82 := [matrixSize82]byte{}
	setupMatrix82(matrixRows, inputOffset, outputOffset, &matrix82)
	addTo := inputOffset != 0 // Except for the first input column, add to previous results
	_galMulAVX512Parallel82(in[inputOffset:inputEnd], out[outputOffset:outputEnd], &matrix82, addTo)

	done = (done >> 6) << 6
	if len(in[0])-done == 0 {
		return
	}

	for c := inputOffset; c < inputOffset+dimIn; c++ {
		for iRow := outputOffset; iRow < outputOffset+dimOut82; iRow++ {
			if c < len(matrixRows[iRow]) {
				mt := mulTable[matrixRows[iRow][c]][:256]
				for i := done; i < len(in[0]); i++ {
					if c == 0 { // only set value for first input column
						out[iRow][i] = mt[in[c][i]]
					} else { // and add for all others
						out[iRow][i] ^= mt[in[c][i]]
					}
				}
			}
		}
	}
}

// Invoke AVX512 routine for 4 output rows in parallel
func galMulAVX512Parallel84(in, out [][]byte, matrixRows [][]byte, inputOffset, outputOffset int) {
	done := len(in[0])
	if done == 0 {
		return
	}

	inputEnd := inputOffset + dimIn
	if inputEnd > len(in) {
		inputEnd = len(in)
	}
	outputEnd := outputOffset + dimOut84
	if outputEnd > len(out) {
		outputEnd = len(out)
	}

	matrix84 := [matrixSize84]byte{}
	setupMatrix84(matrixRows, inputOffset, outputOffset, &matrix84)
	addTo := inputOffset != 0 // Except for the first input column, add to previous results
	_galMulAVX512Parallel84(in[inputOffset:inputEnd], out[outputOffset:outputEnd], &matrix84, addTo)

	done = (done >> 6) << 6
	if len(in[0])-done == 0 {
		return
	}

	for c := inputOffset; c < inputOffset+dimIn; c++ {
		for iRow := outputOffset; iRow < outputOffset+dimOut84; iRow++ {
			if c < len(matrixRows[iRow]) {
				mt := mulTable[matrixRows[iRow][c]][:256]
				for i := done; i < len(in[0]); i++ {
					if c == 0 { // only set value for first input column
						out[iRow][i] = mt[in[c][i]]
					} else { // and add for all others
						out[iRow][i] ^= mt[in[c][i]]
					}
				}
			}
		}
	}
}

// Perform the same as codeSomeShards, but taking advantage of
// AVX512 parallelism for up to 4x faster execution as compared to AVX2
func (r reedSolomon) codeSomeShardsAvx512(matrixRows, inputs, outputs [][]byte, outputCount, byteCount int) {
	outputRow := 0
	// First process (multiple) batches of 4 output rows in parallel
	for ; outputRow+dimOut84 <= len(outputs); outputRow += dimOut84 {
		for inputRow := 0; inputRow < len(inputs); inputRow += dimIn {
			galMulAVX512Parallel84(inputs, outputs, matrixRows, inputRow, outputRow)
		}
	}
	// Then process a (single) batch of 2 output rows in parallel
	if outputRow+dimOut82 <= len(outputs) {
		// fmt.Println(outputRow, len(outputs))
		for inputRow := 0; inputRow < len(inputs); inputRow += dimIn {
			galMulAVX512Parallel82(inputs, outputs, matrixRows, inputRow, outputRow)
		}
		outputRow += dimOut82
	}
	// Lastly, we may have a single output row left (for uneven parity)
	if outputRow < len(outputs) {
		for c := 0; c < r.DataShards; c++ {
			if c == 0 {
				galMulSlice(matrixRows[outputRow][c], inputs[c], outputs[outputRow], &r.o)
			} else {
				galMulSliceXor(matrixRows[outputRow][c], inputs[c], outputs[outputRow], &r.o)
			}
		}
	}
}
