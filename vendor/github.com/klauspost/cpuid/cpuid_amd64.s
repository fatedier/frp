// Copyright (c) 2015 Klaus Post, released under MIT License. See LICENSE file.

//+build amd64,!gccgo

// func asmCpuid(op uint32) (eax, ebx, ecx, edx uint32)
TEXT ·asmCpuid(SB), 7, $0
	XORQ CX, CX
	MOVL op+0(FP), AX
	CPUID
	MOVL AX, eax+8(FP)
	MOVL BX, ebx+12(FP)
	MOVL CX, ecx+16(FP)
	MOVL DX, edx+20(FP)
	RET

// func asmCpuidex(op, op2 uint32) (eax, ebx, ecx, edx uint32)
TEXT ·asmCpuidex(SB), 7, $0
	MOVL op+0(FP), AX
	MOVL op2+4(FP), CX
	CPUID
	MOVL AX, eax+8(FP)
	MOVL BX, ebx+12(FP)
	MOVL CX, ecx+16(FP)
	MOVL DX, edx+20(FP)
	RET

// func asmXgetbv(index uint32) (eax, edx uint32)
TEXT ·asmXgetbv(SB), 7, $0
	MOVL index+0(FP), CX
	BYTE $0x0f; BYTE $0x01; BYTE $0xd0 // XGETBV
	MOVL AX, eax+8(FP)
	MOVL DX, edx+12(FP)
	RET

// func asmRdtscpAsm() (eax, ebx, ecx, edx uint32)
TEXT ·asmRdtscpAsm(SB), 7, $0
	BYTE $0x0F; BYTE $0x01; BYTE $0xF9 // RDTSCP
	MOVL AX, eax+0(FP)
	MOVL BX, ebx+4(FP)
	MOVL CX, ecx+8(FP)
	MOVL DX, edx+12(FP)
	RET
