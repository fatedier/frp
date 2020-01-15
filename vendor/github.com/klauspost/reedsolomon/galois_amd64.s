//+build !noasm !appengine !gccgo

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
	MOVQ   SI, AX
	MOVQ   DX, BX
	ANDQ   $15, AX
	ANDQ   $15, BX
	CMPQ   R9, $0
	JEQ    done_xor
	ORQ    AX, BX
	CMPQ   BX, $0
	JNZ    loopback_xor

loopback_xor_aligned:
	MOVOA  (SI), X0             // in[x]
	MOVOA  (DX), X4             // out[x]
	MOVOA  X0, X1               // in[x]
	MOVOA  X6, X2               // low copy
	MOVOA  X7, X3               // high copy
	PSRLQ  $4, X1               // X1: high input
	PAND   X8, X0               // X0: low input
	PAND   X8, X1               // X0: high input
	PSHUFB X0, X2               // X2: mul low part
	PSHUFB X1, X3               // X3: mul high part
	PXOR   X2, X3               // X3: Result
	PXOR   X4, X3               // X3: Result xor existing out
	MOVOA  X3, (DX)             // Store
	ADDQ   $16, SI              // in+=16
	ADDQ   $16, DX              // out+=16
	SUBQ   $1, R9
	JNZ    loopback_xor_aligned
	JMP    done_xor

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
	MOVQ   SI, AX
	MOVQ   DX, BX
	SHRQ   $4, R9            // len(in) / 16
	ANDQ   $15, AX
	ANDQ   $15, BX
	CMPQ   R9, $0
	JEQ    done
	ORQ    AX, BX
	CMPQ   BX, $0
	JNZ    loopback

loopback_aligned:
	MOVOA  (SI), X0         // in[x]
	MOVOA  X0, X1           // in[x]
	MOVOA  X6, X2           // low copy
	MOVOA  X7, X3           // high copy
	PSRLQ  $4, X1           // X1: high input
	PAND   X8, X0           // X0: low input
	PAND   X8, X1           // X0: high input
	PSHUFB X0, X2           // X2: mul low part
	PSHUFB X1, X3           // X3: mul high part
	PXOR   X2, X3           // X3: Result
	MOVOA  X3, (DX)         // Store
	ADDQ   $16, SI          // in+=16
	ADDQ   $16, DX          // out+=16
	SUBQ   $1, R9
	JNZ    loopback_aligned
	JMP    done

loopback:
	MOVOU  (SI), X0 // in[x]
	MOVOU  X0, X1   // in[x]
	MOVOA  X6, X2   // low copy
	MOVOA  X7, X3   // high copy
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
	MOVOU (SI), X6          // X6: low
	MOVOU (DX), X7          // X7: high
	MOVQ  in_len+56(FP), R9 // R9: len(in)

	VINSERTI128  $1, X6, Y6, Y6 // low
	VINSERTI128  $1, X7, Y7, Y7 // high
	VPBROADCASTB X5, Y8         // Y8: lomask (unpacked)

	SHRQ  $5, R9         // len(in) / 32
	MOVQ  out+72(FP), DX // DX: &out
	MOVQ  in+48(FP), SI  // SI: &in
	TESTQ R9, R9
	JZ    done_xor_avx2

loopback_xor_avx2:
	VMOVDQU (SI), Y0
	VMOVDQU (DX), Y4
	VPSRLQ  $4, Y0, Y1 // Y1: high input
	VPAND   Y8, Y0, Y0 // Y0: low input
	VPAND   Y8, Y1, Y1 // Y1: high input
	VPSHUFB Y0, Y6, Y2 // Y2: mul low part
	VPSHUFB Y1, Y7, Y3 // Y3: mul high part
	VPXOR   Y3, Y2, Y3 // Y3: Result
	VPXOR   Y4, Y3, Y4 // Y4: Result
	VMOVDQU Y4, (DX)

	ADDQ $32, SI           // in+=32
	ADDQ $32, DX           // out+=32
	SUBQ $1, R9
	JNZ  loopback_xor_avx2

done_xor_avx2:
	VZEROUPPER
	RET

// func galMulAVX2(low, high, in, out []byte)
TEXT ·galMulAVX2(SB), 7, $0
	MOVQ  low+0(FP), SI     // SI: &low
	MOVQ  high+24(FP), DX   // DX: &high
	MOVQ  $15, BX           // BX: low mask
	MOVQ  BX, X5
	MOVOU (SI), X6          // X6: low
	MOVOU (DX), X7          // X7: high
	MOVQ  in_len+56(FP), R9 // R9: len(in)

	VINSERTI128  $1, X6, Y6, Y6 // low
	VINSERTI128  $1, X7, Y7, Y7 // high
	VPBROADCASTB X5, Y8         // Y8: lomask (unpacked)

	SHRQ  $5, R9         // len(in) / 32
	MOVQ  out+72(FP), DX // DX: &out
	MOVQ  in+48(FP), SI  // SI: &in
	TESTQ R9, R9
	JZ    done_avx2

loopback_avx2:
	VMOVDQU (SI), Y0
	VPSRLQ  $4, Y0, Y1 // Y1: high input
	VPAND   Y8, Y0, Y0 // Y0: low input
	VPAND   Y8, Y1, Y1 // Y1: high input
	VPSHUFB Y0, Y6, Y2 // Y2: mul low part
	VPSHUFB Y1, Y7, Y3 // Y3: mul high part
	VPXOR   Y3, Y2, Y4 // Y4: Result
	VMOVDQU Y4, (DX)

	ADDQ $32, SI       // in+=32
	ADDQ $32, DX       // out+=32
	SUBQ $1, R9
	JNZ  loopback_avx2

done_avx2:
	VZEROUPPER
	RET

// func sSE2XorSlice(in, out []byte)
TEXT ·sSE2XorSlice(SB), 7, $0
	MOVQ in+0(FP), SI     // SI: &in
	MOVQ in_len+8(FP), R9 // R9: len(in)
	MOVQ out+24(FP), DX   // DX: &out
	SHRQ $4, R9           // len(in) / 16
	CMPQ R9, $0
	JEQ  done_xor_sse2

loopback_xor_sse2:
	MOVOU (SI), X0          // in[x]
	MOVOU (DX), X1          // out[x]
	PXOR  X0, X1
	MOVOU X1, (DX)
	ADDQ  $16, SI           // in+=16
	ADDQ  $16, DX           // out+=16
	SUBQ  $1, R9
	JNZ   loopback_xor_sse2

done_xor_sse2:
	RET
