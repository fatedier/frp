// Copyright (c) 2015 Klaus Post, released under MIT License. See LICENSE file.

// +build !amd64,!386 gccgo

package cpuid

func initCPU() {
	cpuid = func(op uint32) (eax, ebx, ecx, edx uint32) {
		return 0, 0, 0, 0
	}

	cpuidex = func(op, op2 uint32) (eax, ebx, ecx, edx uint32) {
		return 0, 0, 0, 0
	}

	xgetbv = func(index uint32) (eax, edx uint32) {
		return 0, 0
	}

	rdtscpAsm = func() (eax, ebx, ecx, edx uint32) {
		return 0, 0, 0, 0
	}
}
