// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kcp

import (
	"runtime"
	"unsafe"
)

const wordSize = int(unsafe.Sizeof(uintptr(0)))
const supportsUnaligned = runtime.GOARCH == "386" || runtime.GOARCH == "amd64" || runtime.GOARCH == "ppc64" || runtime.GOARCH == "ppc64le" || runtime.GOARCH == "s390x"

// fastXORBytes xors in bulk. It only works on architectures that
// support unaligned read/writes.
func fastXORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}

	w := n / wordSize
	if w > 0 {
		wordBytes := w * wordSize
		fastXORWords(dst[:wordBytes], a[:wordBytes], b[:wordBytes])
	}

	for i := (n - n%wordSize); i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}

	return n
}

func safeXORBytes(dst, a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
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
	return n
}

// xorBytes xors the bytes in a and b. The destination is assumed to have enough
// space. Returns the number of bytes xor'd.
func xorBytes(dst, a, b []byte) int {
	if supportsUnaligned {
		return fastXORBytes(dst, a, b)
	}
	// TODO(hanwen): if (dst, a, b) have common alignment
	// we could still try fastXORBytes. It is not clear
	// how often this happens, and it's only worth it if
	// the block encryption itself is hardware
	// accelerated.
	return safeXORBytes(dst, a, b)
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

func xorWords(dst, a, b []byte) {
	if supportsUnaligned {
		fastXORWords(dst, a, b)
	} else {
		safeXORBytes(dst, a, b)
	}
}
