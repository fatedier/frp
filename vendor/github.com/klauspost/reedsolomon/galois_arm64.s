//+build !noasm !appengine !gccgo

// Copyright 2015, Klaus Post, see LICENSE for details.
// Copyright 2017, Minio, Inc.

// Use github.com/minio/asm2plan9s on this file to assemble ARM instructions to
// the opcodes of their Plan9 equivalents

// polynomial multiplication
#define POLYNOMIAL_MULTIPLICATION \
	WORD $0x0e3ce340 \ // pmull  v0.8h,v26.8b,v28.8b
	WORD $0x4e3ce346 \ // pmull2 v6.8h,v26.16b,v28.16b
	WORD $0x0e3ce36c \ // pmull  v12.8h,v27.8b,v28.8b
	WORD $0x4e3ce372 // pmull2 v18.8h,v27.16b,v28.16b

// first reduction
#define FIRST_REDUCTION \
	WORD $0x0f088402 \ // shrn  v2.8b, v0.8h, #8
	WORD $0x0f0884c8 \ // shrn  v8.8b, v6.8h, #8
	WORD $0x0f08858e \ // shrn  v14.8b, v12.8h, #8
	WORD $0x0f088654 \ // shrn  v20.8b, v18.8h, #8
	WORD $0x0e22e3c3 \ // pmull v3.8h,v30.8b,v2.8b
	WORD $0x0e28e3c9 \ // pmull v9.8h,v30.8b,v8.8b
	WORD $0x0e2ee3cf \ // pmull v15.8h,v30.8b,v14.8b
	WORD $0x0e34e3d5 \ // pmull v21.8h,v30.8b,v20.8b
	WORD $0x6e201c60 \ // eor   v0.16b,v3.16b,v0.16b
	WORD $0x6e261d26 \ // eor   v6.16b,v9.16b,v6.16b
	WORD $0x6e2c1dec \ // eor   v12.16b,v15.16b,v12.16b
	WORD $0x6e321eb2 // eor   v18.16b,v21.16b,v18.16b

// second reduction
#define SECOND_REDUCTION \
	WORD $0x0f088404 \ // shrn  v4.8b, v0.8h, #8
	WORD $0x0f0884ca \ // shrn  v10.8b, v6.8h, #8
	WORD $0x0f088590 \ // shrn  v16.8b, v12.8h, #8
	WORD $0x0f088656 \ // shrn  v22.8b, v18.8h, #8
	WORD $0x6e241c44 \ // eor   v4.16b,v2.16b,v4.16b
	WORD $0x6e2a1d0a \ // eor   v10.16b,v8.16b,v10.16b
	WORD $0x6e301dd0 \ // eor   v16.16b,v14.16b,v16.16b
	WORD $0x6e361e96 \ // eor   v22.16b,v20.16b,v22.16b
	WORD $0x0e24e3c5 \ // pmull v5.8h,v30.8b,v4.8b
	WORD $0x0e2ae3cb \ // pmull v11.8h,v30.8b,v10.8b
	WORD $0x0e30e3d1 \ // pmull v17.8h,v30.8b,v16.8b
	WORD $0x0e36e3d7 \ // pmull v23.8h,v30.8b,v22.8b
	WORD $0x6e201ca0 \ // eor   v0.16b,v5.16b,v0.16b
	WORD $0x6e261d61 \ // eor   v1.16b,v11.16b,v6.16b
	WORD $0x6e2c1e22 \ // eor   v2.16b,v17.16b,v12.16b
	WORD $0x6e321ee3 // eor   v3.16b,v23.16b,v18.16b

// func galMulNEON(c uint64, in, out []byte)
TEXT ·galMulNEON(SB), 7, $0
	MOVD c+0(FP), R0
	MOVD in_base+8(FP), R1
	MOVD in_len+16(FP), R2   // length of message
	MOVD out_base+32(FP), R5
	SUBS $32, R2
	BMI  complete

	// Load constants table pointer
	MOVD $·constants(SB), R3

	// and load constants into v30 & v31
	WORD $0x4c40a07e // ld1    {v30.16b-v31.16b}, [x3]

	WORD $0x4e010c1c // dup    v28.16b, w0

loop:
	// Main loop
	WORD $0x4cdfa83a // ld1   {v26.4s-v27.4s}, [x1], #32

	POLYNOMIAL_MULTIPLICATION

	FIRST_REDUCTION

	SECOND_REDUCTION

	// combine results
	WORD $0x4e1f2000 // tbl v0.16b,{v0.16b,v1.16b},v31.16b
	WORD $0x4e1f2041 // tbl v1.16b,{v2.16b,v3.16b},v31.16b

	// Store result
	WORD $0x4c9faca0 // st1    {v0.2d-v1.2d}, [x5], #32

	SUBS $32, R2
	BPL  loop

complete:
	RET

// func galMulXorNEON(c uint64, in, out []byte)
TEXT ·galMulXorNEON(SB), 7, $0
	MOVD c+0(FP), R0
	MOVD in_base+8(FP), R1
	MOVD in_len+16(FP), R2   // length of message
	MOVD out_base+32(FP), R5
	SUBS $32, R2
	BMI  completeXor

	// Load constants table pointer
	MOVD $·constants(SB), R3

	// and load constants into v30 & v31
	WORD $0x4c40a07e // ld1    {v30.16b-v31.16b}, [x3]

	WORD $0x4e010c1c // dup    v28.16b, w0

loopXor:
	// Main loop
	WORD $0x4cdfa83a // ld1   {v26.4s-v27.4s}, [x1], #32
	WORD $0x4c40a8b8 // ld1   {v24.4s-v25.4s}, [x5]

	POLYNOMIAL_MULTIPLICATION

	FIRST_REDUCTION

	SECOND_REDUCTION

	// combine results
	WORD $0x4e1f2000 // tbl v0.16b,{v0.16b,v1.16b},v31.16b
	WORD $0x4e1f2041 // tbl v1.16b,{v2.16b,v3.16b},v31.16b

	// Xor result and store
	WORD $0x6e381c00 // eor v0.16b,v0.16b,v24.16b
	WORD $0x6e391c21 // eor v1.16b,v1.16b,v25.16b
	WORD $0x4c9faca0 // st1   {v0.2d-v1.2d}, [x5], #32

	SUBS $32, R2
	BPL  loopXor

completeXor:
	RET

// Constants table
//   generating polynomial is 29 (= 0x1d)
DATA ·constants+0x0(SB)/8, $0x1d1d1d1d1d1d1d1d
DATA ·constants+0x8(SB)/8, $0x1d1d1d1d1d1d1d1d
//   constant for TBL instruction
DATA ·constants+0x10(SB)/8, $0x0e0c0a0806040200
DATA ·constants+0x18(SB)/8, $0x1e1c1a1816141210

GLOBL ·constants(SB), 8, $32
