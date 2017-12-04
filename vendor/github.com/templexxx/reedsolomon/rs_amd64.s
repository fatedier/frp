// Reference: www.ssrc.ucsc.edu/Papers/plank-fast13.pdf

#include "textflag.h"

#define low_tbl Y0
#define high_tbl Y1
#define mask Y2
#define in0  Y3
#define in1  Y4
#define in2  Y5
#define in3  Y6
#define in4  Y7
#define in5  Y8
#define in0_h  Y10
#define in1_h  Y11
#define in2_h  Y12
#define in3_h  Y13
#define in4_h  Y14
#define in5_h  Y15

#define in  BX
#define out DI
#define len R8
#define pos R9

#define tmp0 R10

#define low_tblx X0
#define high_tblx X1
#define maskx X2
#define in0x X3
#define in0_hx X10
#define tmp0x  X9
#define tmp1x  X11
#define tmp2x  X12
#define tmp3x  X13


// func mulVectAVX2(tbl, d, p []byte)
TEXT ·mulVectAVX2(SB), NOSPLIT, $0
    MOVQ         i+24(FP), in
	MOVQ         o+48(FP), out
	MOVQ         tbl+0(FP), tmp0
	VMOVDQU      (tmp0), low_tblx
	VMOVDQU      16(tmp0), high_tblx
	MOVB         $0x0f, DX
	LONG         $0x2069e3c4; WORD $0x00d2   // VPINSRB $0x00, EDX, XMM2, XMM2
	VPBROADCASTB maskx, maskx
	MOVQ         in_len+32(FP), len
	TESTQ        $31, len
	JNZ          one16b

ymm:
    VINSERTI128  $1, low_tblx, low_tbl, low_tbl
    VINSERTI128  $1, high_tblx, high_tbl, high_tbl
    VINSERTI128  $1, maskx, mask, mask
    TESTQ        $255, len
    JNZ          not_aligned

// 256bytes/loop
aligned:
    MOVQ         $0, pos

loop256b:
	VMOVDQU (in)(pos*1), in0
	VPSRLQ  $4, in0, in0_h
	VPAND   mask, in0_h, in0_h
	VPAND   mask, in0, in0
	VPSHUFB in0_h, high_tbl, in0_h
	VPSHUFB in0, low_tbl, in0
	VPXOR   in0, in0_h, in0
	VMOVDQU in0, (out)(pos*1)

    VMOVDQU 32(in)(pos*1), in1
	VPSRLQ  $4, in1, in1_h
	VPAND   mask, in1_h, in1_h
	VPAND   mask, in1, in1
	VPSHUFB in1_h, high_tbl, in1_h
	VPSHUFB in1, low_tbl, in1
	VPXOR   in1, in1_h, in1
	VMOVDQU in1, 32(out)(pos*1)

    VMOVDQU 64(in)(pos*1), in2
	VPSRLQ  $4, in2, in2_h
	VPAND   mask, in2_h, in2_h
	VPAND   mask, in2, in2
	VPSHUFB in2_h, high_tbl, in2_h
	VPSHUFB in2, low_tbl, in2
	VPXOR   in2, in2_h, in2
	VMOVDQU in2, 64(out)(pos*1)

    VMOVDQU 96(in)(pos*1), in3
	VPSRLQ  $4, in3, in3_h
	VPAND   mask, in3_h, in3_h
	VPAND   mask, in3, in3
	VPSHUFB in3_h, high_tbl, in3_h
	VPSHUFB in3, low_tbl, in3
	VPXOR   in3, in3_h, in3
	VMOVDQU in3, 96(out)(pos*1)

    VMOVDQU 128(in)(pos*1), in4
	VPSRLQ  $4, in4, in4_h
	VPAND   mask, in4_h, in4_h
	VPAND   mask, in4, in4
	VPSHUFB in4_h, high_tbl, in4_h
	VPSHUFB in4, low_tbl, in4
	VPXOR   in4, in4_h, in4
	VMOVDQU in4, 128(out)(pos*1)

    VMOVDQU 160(in)(pos*1), in5
	VPSRLQ  $4, in5, in5_h
	VPAND   mask, in5_h, in5_h
	VPAND   mask, in5, in5
	VPSHUFB in5_h, high_tbl, in5_h
	VPSHUFB in5, low_tbl, in5
	VPXOR   in5, in5_h, in5
	VMOVDQU in5, 160(out)(pos*1)

    VMOVDQU 192(in)(pos*1), in0
	VPSRLQ  $4, in0, in0_h
	VPAND   mask, in0_h, in0_h
	VPAND   mask, in0, in0
	VPSHUFB in0_h, high_tbl, in0_h
	VPSHUFB in0, low_tbl, in0
	VPXOR   in0, in0_h, in0
	VMOVDQU in0, 192(out)(pos*1)

    VMOVDQU 224(in)(pos*1), in1
	VPSRLQ  $4, in1, in1_h
	VPAND   mask, in1_h, in1_h
	VPAND   mask, in1, in1
	VPSHUFB in1_h, high_tbl, in1_h
	VPSHUFB in1, low_tbl, in1
	VPXOR   in1, in1_h, in1
	VMOVDQU in1, 224(out)(pos*1)

	ADDQ    $256, pos
	CMPQ    len, pos
	JNE     loop256b
	VZEROUPPER
	RET

not_aligned:
    MOVQ    len, tmp0
    ANDQ    $255, tmp0

loop32b:
    VMOVDQU -32(in)(len*1), in0
	VPSRLQ  $4, in0, in0_h
	VPAND   mask, in0_h, in0_h
	VPAND   mask, in0, in0
	VPSHUFB in0_h, high_tbl, in0_h
	VPSHUFB in0, low_tbl, in0
	VPXOR   in0, in0_h, in0
	VMOVDQU in0, -32(out)(len*1)
	SUBQ    $32, len
	SUBQ    $32, tmp0
	JG      loop32b
	CMPQ    len, $256
	JGE     aligned
	VZEROUPPER
	RET

one16b:
    VMOVDQU  -16(in)(len*1), in0x
    VPSRLQ   $4, in0x, in0_hx
    VPAND    maskx, in0x, in0x
    VPAND    maskx, in0_hx, in0_hx
    VPSHUFB  in0_hx, high_tblx, in0_hx
    VPSHUFB  in0x, low_tblx, in0x
    VPXOR    in0x, in0_hx, in0x
	VMOVDQU  in0x, -16(out)(len*1)
	SUBQ     $16, len
	CMPQ     len, $0
	JNE      ymm
	RET

// func mulVectAddAVX2(tbl, d, p []byte)
TEXT ·mulVectAddAVX2(SB), NOSPLIT, $0
    MOVQ         i+24(FP), in
	MOVQ         o+48(FP), out
	MOVQ         tbl+0(FP), tmp0
	VMOVDQU      (tmp0), low_tblx
	VMOVDQU      16(tmp0), high_tblx
	MOVB         $0x0f, DX
	LONG         $0x2069e3c4; WORD $0x00d2
	VPBROADCASTB maskx, maskx
	MOVQ         in_len+32(FP), len
	TESTQ        $31, len
	JNZ          one16b

ymm:
    VINSERTI128  $1, low_tblx, low_tbl, low_tbl
    VINSERTI128  $1, high_tblx, high_tbl, high_tbl
    VINSERTI128  $1, maskx, mask, mask
    TESTQ        $255, len
    JNZ          not_aligned

aligned:
    MOVQ         $0, pos

loop256b:
    VMOVDQU (in)(pos*1), in0
	VPSRLQ  $4, in0, in0_h
	VPAND   mask, in0_h, in0_h
	VPAND   mask, in0, in0
	VPSHUFB in0_h, high_tbl, in0_h
	VPSHUFB in0, low_tbl, in0
	VPXOR   in0, in0_h, in0
	VPXOR   (out)(pos*1), in0, in0
	VMOVDQU in0, (out)(pos*1)

    VMOVDQU 32(in)(pos*1), in1
	VPSRLQ  $4, in1, in1_h
	VPAND   mask, in1_h, in1_h
	VPAND   mask, in1, in1
	VPSHUFB in1_h, high_tbl, in1_h
	VPSHUFB in1, low_tbl, in1
	VPXOR   in1, in1_h, in1
	VPXOR   32(out)(pos*1), in1, in1
	VMOVDQU in1, 32(out)(pos*1)

    VMOVDQU 64(in)(pos*1), in2
	VPSRLQ  $4, in2, in2_h
	VPAND   mask, in2_h, in2_h
	VPAND   mask, in2, in2
	VPSHUFB in2_h, high_tbl, in2_h
	VPSHUFB in2, low_tbl, in2
	VPXOR   in2, in2_h, in2
	VPXOR   64(out)(pos*1), in2, in2
	VMOVDQU in2, 64(out)(pos*1)

    VMOVDQU 96(in)(pos*1), in3
	VPSRLQ  $4, in3, in3_h
	VPAND   mask, in3_h, in3_h
	VPAND   mask, in3, in3
	VPSHUFB in3_h, high_tbl, in3_h
	VPSHUFB in3, low_tbl, in3
	VPXOR   in3, in3_h, in3
	VPXOR   96(out)(pos*1), in3, in3
	VMOVDQU in3, 96(out)(pos*1)

    VMOVDQU 128(in)(pos*1), in4
	VPSRLQ  $4, in4, in4_h
	VPAND   mask, in4_h, in4_h
	VPAND   mask, in4, in4
	VPSHUFB in4_h, high_tbl, in4_h
	VPSHUFB in4, low_tbl, in4
	VPXOR   in4, in4_h, in4
	VPXOR   128(out)(pos*1), in4, in4
	VMOVDQU in4, 128(out)(pos*1)

    VMOVDQU 160(in)(pos*1), in5
	VPSRLQ  $4, in5, in5_h
	VPAND   mask, in5_h, in5_h
	VPAND   mask, in5, in5
	VPSHUFB in5_h, high_tbl, in5_h
	VPSHUFB in5, low_tbl, in5
	VPXOR   in5, in5_h, in5
	VPXOR   160(out)(pos*1), in5, in5
	VMOVDQU in5, 160(out)(pos*1)

    VMOVDQU 192(in)(pos*1), in0
	VPSRLQ  $4, in0, in0_h
	VPAND   mask, in0_h, in0_h
	VPAND   mask, in0, in0
	VPSHUFB in0_h, high_tbl, in0_h
	VPSHUFB in0, low_tbl, in0
	VPXOR   in0, in0_h, in0
	VPXOR   192(out)(pos*1), in0, in0
	VMOVDQU in0, 192(out)(pos*1)

    VMOVDQU 224(in)(pos*1), in1
	VPSRLQ  $4, in1, in1_h
	VPAND   mask, in1_h, in1_h
	VPAND   mask, in1, in1
	VPSHUFB in1_h, high_tbl, in1_h
	VPSHUFB in1, low_tbl, in1
	VPXOR   in1, in1_h, in1
	VPXOR   224(out)(pos*1), in1, in1
	VMOVDQU in1, 224(out)(pos*1)

	ADDQ    $256, pos
	CMPQ    len, pos
	JNE     loop256b
	VZEROUPPER
	RET

not_aligned:
    MOVQ    len, tmp0
    ANDQ    $255, tmp0

loop32b:
    VMOVDQU -32(in)(len*1), in0
	VPSRLQ  $4, in0, in0_h
	VPAND   mask, in0_h, in0_h
	VPAND   mask, in0, in0
	VPSHUFB in0_h, high_tbl, in0_h
	VPSHUFB in0, low_tbl, in0
	VPXOR   in0, in0_h, in0
	VPXOR   -32(out)(len*1), in0, in0
	VMOVDQU in0, -32(out)(len*1)
	SUBQ    $32, len
	SUBQ    $32, tmp0
	JG      loop32b
	CMPQ    len, $256
	JGE     aligned
	VZEROUPPER
	RET

one16b:
    VMOVDQU  -16(in)(len*1), in0x
    VPSRLQ   $4, in0x, in0_hx
    VPAND    maskx, in0x, in0x
    VPAND    maskx, in0_hx, in0_hx
    VPSHUFB  in0_hx, high_tblx, in0_hx
    VPSHUFB  in0x, low_tblx, in0x
    VPXOR    in0x, in0_hx, in0x
    VPXOR    -16(out)(len*1), in0x, in0x
	VMOVDQU  in0x, -16(out)(len*1)
	SUBQ     $16, len
	CMPQ     len, $0
	JNE      ymm
	RET

// func mulVectSSSE3(tbl, d, p []byte)
TEXT ·mulVectSSSE3(SB), NOSPLIT, $0
    MOVQ    i+24(FP), in
	MOVQ    o+48(FP), out
	MOVQ    tbl+0(FP), tmp0
	MOVOU   (tmp0), low_tblx
	MOVOU   16(tmp0), high_tblx
    MOVB    $15, tmp0
    MOVQ    tmp0, maskx
    PXOR    tmp0x, tmp0x
   	PSHUFB  tmp0x, maskx
	MOVQ    in_len+32(FP), len
	SHRQ    $4, len

loop:
	MOVOU  (in), in0x
	MOVOU  in0x, in0_hx
	PSRLQ  $4, in0_hx
	PAND   maskx, in0x
	PAND   maskx, in0_hx
	MOVOU  low_tblx, tmp1x
	MOVOU  high_tblx, tmp2x
	PSHUFB in0x, tmp1x
	PSHUFB in0_hx, tmp2x
	PXOR   tmp1x, tmp2x
	MOVOU  tmp2x, (out)
	ADDQ   $16, in
	ADDQ   $16, out
	SUBQ   $1, len
	JNZ    loop
	RET

// func mulVectAddSSSE3(tbl, d, p []byte)
TEXT ·mulVectAddSSSE3(SB), NOSPLIT, $0
    MOVQ    i+24(FP), in
	MOVQ    o+48(FP), out
	MOVQ    tbl+0(FP), tmp0
	MOVOU   (tmp0), low_tblx
	MOVOU   16(tmp0), high_tblx
    MOVB    $15, tmp0
    MOVQ    tmp0, maskx
    PXOR    tmp0x, tmp0x
   	PSHUFB  tmp0x, maskx
	MOVQ    in_len+32(FP), len
	SHRQ    $4, len

loop:
	MOVOU  (in), in0x
	MOVOU  in0x, in0_hx
	PSRLQ  $4, in0_hx
	PAND   maskx, in0x
	PAND   maskx, in0_hx
	MOVOU  low_tblx, tmp1x
	MOVOU  high_tblx, tmp2x
	PSHUFB in0x, tmp1x
	PSHUFB in0_hx, tmp2x
	PXOR   tmp1x, tmp2x
	MOVOU  (out), tmp3x
	PXOR   tmp3x, tmp2x
	MOVOU  tmp2x, (out)
	ADDQ   $16, in
	ADDQ   $16, out
	SUBQ   $1, len
	JNZ    loop
	RET

// func copy32B(dst, src []byte)
TEXT ·copy32B(SB), NOSPLIT, $0
    MOVQ dst+0(FP), SI
    MOVQ src+24(FP), DX
    MOVOU (DX), X0
    MOVOU 16(DX), X1
    MOVOU X0, (SI)
    MOVOU X1, 16(SI)
    RET
	
