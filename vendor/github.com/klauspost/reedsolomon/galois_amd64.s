//+build !noasm !appengine

// Copyright 2015, Klaus Post, see LICENSE for details.

// Based on http://www.snia.org/sites/default/files2/SDC2013/presentations/NewThinking/EthanMiller_Screaming_Fast_Galois_Field%20Arithmetic_SIMD%20Instructions.pdf
// and http://jerasure.org/jerasure/gf-complete/tree/master

// func galMulSSSE3Xor(low, high, in, out []byte)
TEXT ·galMulSSSE3Xor(SB), 7, $0
	MOVQ   low+0(FP), SI     // SI: &low
	MOVQ   high+24(FP), DX   // DX: &high
	MOVOU  (SI), X6          // X6 low
	MOVOU  (DX), X7          // X7: high
	MOVQ   $15, BX           // BX: low mask
	MOVQ   BX, X8
	PXOR   X5, X5
	MOVQ   in+48(FP), SI     // R11: &in
	MOVQ   in_len+56(FP), R9 // R9: len(in)
	MOVQ   out+72(FP), DX    // DX: &out
	PSHUFB X5, X8            // X8: lomask (unpacked)
	SHRQ   $4, R9            // len(in) / 16
	CMPQ   R9, $0
	JEQ    done_xor

loopback_xor:
	MOVOU  (SI), X0     // in[x]
	MOVOU  (DX), X4     // out[x]
	MOVOU  X0, X1       // in[x]
	MOVOU  X6, X2       // low copy
	MOVOU  X7, X3       // high copy
	PSRLQ  $4, X1       // X1: high input
	PAND   X8, X0       // X0: low input
	PAND   X8, X1       // X0: high input
	PSHUFB X0, X2       // X2: mul low part
	PSHUFB X1, X3       // X3: mul high part
	PXOR   X2, X3       // X3: Result
	PXOR   X4, X3       // X3: Result xor existing out
	MOVOU  X3, (DX)     // Store
	ADDQ   $16, SI      // in+=16
	ADDQ   $16, DX      // out+=16
	SUBQ   $1, R9
	JNZ    loopback_xor

done_xor:
	RET

// func galMulSSSE3(low, high, in, out []byte)
TEXT ·galMulSSSE3(SB), 7, $0
	MOVQ   low+0(FP), SI     // SI: &low
	MOVQ   high+24(FP), DX   // DX: &high
	MOVOU  (SI), X6          // X6 low
	MOVOU  (DX), X7          // X7: high
	MOVQ   $15, BX           // BX: low mask
	MOVQ   BX, X8
	PXOR   X5, X5
	MOVQ   in+48(FP), SI     // R11: &in
	MOVQ   in_len+56(FP), R9 // R9: len(in)
	MOVQ   out+72(FP), DX    // DX: &out
	PSHUFB X5, X8            // X8: lomask (unpacked)
	SHRQ   $4, R9            // len(in) / 16
	CMPQ   R9, $0
	JEQ    done

loopback:
	MOVOU  (SI), X0 // in[x]
	MOVOU  X0, X1   // in[x]
	MOVOU  X6, X2   // low copy
	MOVOU  X7, X3   // high copy
	PSRLQ  $4, X1   // X1: high input
	PAND   X8, X0   // X0: low input
	PAND   X8, X1   // X0: high input
	PSHUFB X0, X2   // X2: mul low part
	PSHUFB X1, X3   // X3: mul high part
	PXOR   X2, X3   // X3: Result
	MOVOU  X3, (DX) // Store
	ADDQ   $16, SI  // in+=16
	ADDQ   $16, DX  // out+=16
	SUBQ   $1, R9
	JNZ    loopback

done:
	RET

// func galMulAVX2Xor(low, high, in, out []byte)
TEXT ·galMulAVX2Xor(SB), 7, $0
	MOVQ  low+0(FP), SI     // SI: &low
	MOVQ  high+24(FP), DX   // DX: &high
	MOVQ  $15, BX           // BX: low mask
	MOVQ  BX, X5
	MOVOU (SI), X6          // X6 low
	MOVOU (DX), X7          // X7: high
	MOVQ  in_len+56(FP), R9 // R9: len(in)

	LONG $0x384de3c4; WORD $0x01f6 // VINSERTI128 YMM6, YMM6, XMM6, 1 ; low
	LONG $0x3845e3c4; WORD $0x01ff // VINSERTI128 YMM7, YMM7, XMM7, 1 ; high
	LONG $0x787d62c4; BYTE $0xc5   // VPBROADCASTB YMM8, XMM5         ; X8: lomask (unpacked)

	SHRQ  $5, R9         // len(in) /32
	MOVQ  out+72(FP), DX // DX: &out
	MOVQ  in+48(FP), SI  // R11: &in
	TESTQ R9, R9
	JZ    done_xor_avx2

loopback_xor_avx2:
	LONG $0x066ffec5             // VMOVDQU YMM0, [rsi]
	LONG $0x226ffec5             // VMOVDQU YMM4, [rdx]
	LONG $0xd073f5c5; BYTE $0x04 // VPSRLQ  YMM1, YMM0, 4   ; X1: high input
	LONG $0xdb7dc1c4; BYTE $0xc0 // VPAND   YMM0, YMM0, YMM8      ; X0: low input
	LONG $0xdb75c1c4; BYTE $0xc8 // VPAND   YMM1, YMM1, YMM8      ; X1: high input
	LONG $0x004de2c4; BYTE $0xd0 // VPSHUFB  YMM2, YMM6, YMM0   ; X2: mul low part
	LONG $0x0045e2c4; BYTE $0xd9 // VPSHUFB  YMM3, YMM7, YMM1   ; X2: mul high part
	LONG $0xdbefedc5             // VPXOR   YMM3, YMM2, YMM3    ; X3: Result
	LONG $0xe4efe5c5             // VPXOR   YMM4, YMM3, YMM4    ; X4: Result
	LONG $0x227ffec5             // VMOVDQU [rdx], YMM4

	ADDQ $32, SI           // in+=32
	ADDQ $32, DX           // out+=32
	SUBQ $1, R9
	JNZ  loopback_xor_avx2

done_xor_avx2:
	// VZEROUPPER
	BYTE $0xc5; BYTE $0xf8; BYTE $0x77
	RET

// func galMulAVX2(low, high, in, out []byte)
TEXT ·galMulAVX2(SB), 7, $0
	MOVQ  low+0(FP), SI     // SI: &low
	MOVQ  high+24(FP), DX   // DX: &high
	MOVQ  $15, BX           // BX: low mask
	MOVQ  BX, X5
	MOVOU (SI), X6          // X6 low
	MOVOU (DX), X7          // X7: high
	MOVQ  in_len+56(FP), R9 // R9: len(in)

	LONG $0x384de3c4; WORD $0x01f6 // VINSERTI128 YMM6, YMM6, XMM6, 1 ; low
	LONG $0x3845e3c4; WORD $0x01ff // VINSERTI128 YMM7, YMM7, XMM7, 1 ; high
	LONG $0x787d62c4; BYTE $0xc5   // VPBROADCASTB YMM8, XMM5         ; X8: lomask (unpacked)

	SHRQ  $5, R9         // len(in) /32
	MOVQ  out+72(FP), DX // DX: &out
	MOVQ  in+48(FP), SI  // R11: &in
	TESTQ R9, R9
	JZ    done_avx2

loopback_avx2:
	LONG $0x066ffec5             // VMOVDQU YMM0, [rsi]
	LONG $0xd073f5c5; BYTE $0x04 // VPSRLQ  YMM1, YMM0, 4   ; X1: high input
	LONG $0xdb7dc1c4; BYTE $0xc0 // VPAND   YMM0, YMM0, YMM8      ; X0: low input
	LONG $0xdb75c1c4; BYTE $0xc8 // VPAND   YMM1, YMM1, YMM8      ; X1: high input
	LONG $0x004de2c4; BYTE $0xd0 // VPSHUFB  YMM2, YMM6, YMM0   ; X2: mul low part
	LONG $0x0045e2c4; BYTE $0xd9 // VPSHUFB  YMM3, YMM7, YMM1   ; X2: mul high part
	LONG $0xe3efedc5             // VPXOR   YMM4, YMM2, YMM3    ; X4: Result
	LONG $0x227ffec5             // VMOVDQU [rdx], YMM4

	ADDQ $32, SI       // in+=32
	ADDQ $32, DX       // out+=32
	SUBQ $1, R9
	JNZ  loopback_avx2

done_avx2:

	BYTE $0xc5; BYTE $0xf8; BYTE $0x77 // VZEROUPPER
	RET
