// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xor

import (
	"runtime"
	"unsafe"
)

const wordSize = int(unsafe.Sizeof(uintptr(0)))
const supportsUnaligned = runtime.GOARCH == "386" || runtime.GOARCH == "amd64" || runtime.GOARCH == "ppc64" || runtime.GOARCH == "ppc64le" || runtime.GOARCH == "s390x"

// xor the bytes in a and b. The destination is assumed to have enough space.
func bytesNoSIMD(dst, a, b []byte, size int) {
	if supportsUnaligned {
		fastXORBytes(dst, a, b, size)
	} else {
		// TODO(hanwen): if (dst, a, b) have common alignment
		// we could still try fastXORBytes. It is not clear
		// how often this happens, and it's only worth it if
		// the block encryption itself is hardware
		// accelerated.
		safeXORBytes(dst, a, b, size)
	}
}

// split slice for cache-friendly
const unitSize = 16 * 1024

func matrixNoSIMD(dst []byte, src [][]byte) {
	size := len(src[0])
	start := 0
	do := unitSize
	for start < size {
		end := start + do
		if end <= size {
			partNoSIMD(start, end, dst, src)
			start = start + do
		} else {
			partNoSIMD(start, size, dst, src)
			start = size
		}
	}
}

// split vect will improve performance with big data by reducing cache pollution
func partNoSIMD(start, end int, dst []byte, src [][]byte) {
	bytesNoSIMD(dst[start:end], src[0][start:end], src[1][start:end], end-start)
	for i := 2; i < len(src); i++ {
		bytesNoSIMD(dst[start:end], dst[start:end], src[i][start:end], end-start)
	}
}

// fastXORBytes xor in bulk. It only works on architectures that
// support unaligned read/writes.
func fastXORBytes(dst, a, b []byte, n int) {
	w := n / wordSize
	if w > 0 {
		wordBytes := w * wordSize
		fastXORWords(dst[:wordBytes], a[:wordBytes], b[:wordBytes])
	}
	for i := n - n%wordSize; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
}

func safeXORBytes(dst, a, b []byte, n int) {
	ex := n % 8
	for i := 0; i < ex; i++ {
		dst[i] = a[i] ^ b[i]
	}

	for i := ex; i < n; i += 8 {
		_dst := dst[i : i+8]
		_a := a[i : i+8]
		_b := b[i : i+8]
		_dst[0] = _a[0] ^ _b[0]
		_dst[1] = _a[1] ^ _b[1]
		_dst[2] = _a[2] ^ _b[2]
		_dst[3] = _a[3] ^ _b[3]

		_dst[4] = _a[4] ^ _b[4]
		_dst[5] = _a[5] ^ _b[5]
		_dst[6] = _a[6] ^ _b[6]
		_dst[7] = _a[7] ^ _b[7]
	}
}

// fastXORWords XORs multiples of 4 or 8 bytes (depending on architecture.)
// The arguments are assumed to be of equal length.
func fastXORWords(dst, a, b []byte) {
	dw := *(*[]uintptr)(unsafe.Pointer(&dst))
	aw := *(*[]uintptr)(unsafe.Pointer(&a))
	bw := *(*[]uintptr)(unsafe.Pointer(&b))
	n := len(b) / wordSize
	ex := n % 8
	for i := 0; i < ex; i++ {
		dw[i] = aw[i] ^ bw[i]
	}

	for i := ex; i < n; i += 8 {
		_dw := dw[i : i+8]
		_aw := aw[i : i+8]
		_bw := bw[i : i+8]
		_dw[0] = _aw[0] ^ _bw[0]
		_dw[1] = _aw[1] ^ _bw[1]
		_dw[2] = _aw[2] ^ _bw[2]
		_dw[3] = _aw[3] ^ _bw[3]
		_dw[4] = _aw[4] ^ _bw[4]
		_dw[5] = _aw[5] ^ _bw[5]
		_dw[6] = _aw[6] ^ _bw[6]
		_dw[7] = _aw[7] ^ _bw[7]
	}
}
