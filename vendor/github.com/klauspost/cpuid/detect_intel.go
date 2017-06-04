// Copyright (c) 2015 Klaus Post, released under MIT License. See LICENSE file.

// +build 386,!gccgo amd64,!gccgo

package cpuid

func asmCpuid(op uint32) (eax, ebx, ecx, edx uint32)
func asmCpuidex(op, op2 uint32) (eax, ebx, ecx, edx uint32)
func asmXgetbv(index uint32) (eax, edx uint32)
func asmRdtscpAsm() (eax, ebx, ecx, edx uint32)

func initCPU() {
	cpuid = asmCpuid
	cpuidex = asmCpuidex
	xgetbv = asmXgetbv
	rdtscpAsm = asmRdtscpAsm
}
