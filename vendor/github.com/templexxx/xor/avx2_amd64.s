#include "textflag.h"

// addr of mem
#define DST BX
#define SRC SI
#define SRC0 TMP4
#define SRC1 TMP5

// loop args
// num of vect
#define VECT CX
#define LEN DX
// pos of matrix
#define POS R8

// tmp store
// num of vect or ...
#define TMP1 R9
// pos of matrix or ...
#define TMP2 R10
// store addr of data/parity or ...
#define TMP3 R11
#define TMP4 R12
#define TMP5 R13
#define TMP6 R14

// func bytesAVX2mini(dst, src0, src1 []byte, size int)
TEXT ·bytesAVX2mini(SB), NOSPLIT, $0
	MOVQ  len+72(FP), LEN
	CMPQ  LEN, $0
	JE    ret
	MOVQ  dst+0(FP), DST
	MOVQ  src0+24(FP), SRC0
	MOVQ  src1+48(FP), SRC1
	TESTQ $31, LEN
	JNZ   not_aligned

aligned:
	MOVQ $0, POS

loop32b:
	VMOVDQU (SRC0)(POS*1), Y0
	VPXOR   (SRC1)(POS*1), Y0, Y0
	VMOVDQU Y0, (DST)(POS*1)
	ADDQ    $32, POS
	CMPQ    LEN, POS
	JNE     loop32b
	VZEROUPPER
	RET

loop_1b:
	MOVB  -1(SRC0)(LEN*1), TMP1
	MOVB  -1(SRC1)(LEN*1), TMP2
	XORB  TMP1, TMP2
	MOVB  TMP2, -1(DST)(LEN*1)
	SUBQ  $1, LEN
	TESTQ $7, LEN
	JNZ   loop_1b
	CMPQ  LEN, $0
	JE    ret
	TESTQ $31, LEN
	JZ    aligned

not_aligned:
	TESTQ $7, LEN
	JNE   loop_1b
	MOVQ  LEN, TMP1
	ANDQ  $31, TMP1

loop_8b:
	MOVQ -8(SRC0)(LEN*1), TMP2
	MOVQ -8(SRC1)(LEN*1), TMP3
	XORQ TMP2, TMP3
	MOVQ TMP3, -8(DST)(LEN*1)
	SUBQ $8, LEN
	SUBQ $8, TMP1
	JG   loop_8b

	CMPQ LEN, $32
	JGE  aligned
	RET

ret:
	RET

// func bytesAVX2small(dst, src0, src1 []byte, size int)
TEXT ·bytesAVX2small(SB), NOSPLIT, $0
	MOVQ  len+72(FP), LEN
	CMPQ  LEN, $0
	JE    ret
	MOVQ  dst+0(FP), DST
	MOVQ  src0+24(FP), SRC0
	MOVQ  src1+48(FP), SRC1
	TESTQ $127, LEN
	JNZ   not_aligned

aligned:
	MOVQ $0, POS

loop128b:
	VMOVDQU (SRC0)(POS*1), Y0
	VMOVDQU 32(SRC0)(POS*1), Y1
	VMOVDQU 64(SRC0)(POS*1), Y2
	VMOVDQU 96(SRC0)(POS*1), Y3
	VPXOR   (SRC1)(POS*1), Y0, Y0
	VPXOR   32(SRC1)(POS*1), Y1, Y1
	VPXOR   64(SRC1)(POS*1), Y2, Y2
	VPXOR   96(SRC1)(POS*1), Y3, Y3
	VMOVDQU Y0, (DST)(POS*1)
	VMOVDQU Y1, 32(DST)(POS*1)
	VMOVDQU Y2, 64(DST)(POS*1)
	VMOVDQU Y3, 96(DST)(POS*1)

	ADDQ $128, POS
	CMPQ LEN, POS
	JNE  loop128b
	VZEROUPPER
	RET

loop_1b:
	MOVB  -1(SRC0)(LEN*1), TMP1
	MOVB  -1(SRC1)(LEN*1), TMP2
	XORB  TMP1, TMP2
	MOVB  TMP2, -1(DST)(LEN*1)
	SUBQ  $1, LEN
	TESTQ $7, LEN
	JNZ   loop_1b
	CMPQ  LEN, $0
	JE    ret
	TESTQ $127, LEN
	JZ    aligned

not_aligned:
	TESTQ $7, LEN
	JNE   loop_1b
	MOVQ  LEN, TMP1
	ANDQ  $127, TMP1

loop_8b:
	MOVQ -8(SRC0)(LEN*1), TMP2
	MOVQ -8(SRC1)(LEN*1), TMP3
	XORQ TMP2, TMP3
	MOVQ TMP3, -8(DST)(LEN*1)
	SUBQ $8, LEN
	SUBQ $8, TMP1
	JG   loop_8b

	CMPQ LEN, $128
	JGE  aligned
	RET

ret:
	RET

// func bytesAVX2big(dst, src0, src1 []byte, size int)
TEXT ·bytesAVX2big(SB), NOSPLIT, $0
	MOVQ  len+72(FP), LEN
	CMPQ  LEN, $0
	JE    ret
	MOVQ  dst+0(FP), DST
	MOVQ  src0+24(FP), SRC0
	MOVQ  src1+48(FP), SRC1
	TESTQ $127, LEN
	JNZ   not_aligned

aligned:
	MOVQ $0, POS

loop128b:
	VMOVDQU (SRC0)(POS*1), Y0
	VMOVDQU 32(SRC0)(POS*1), Y1
	VMOVDQU 64(SRC0)(POS*1), Y2
	VMOVDQU 96(SRC0)(POS*1), Y3
	VPXOR   (SRC1)(POS*1), Y0, Y0
	VPXOR   32(SRC1)(POS*1), Y1, Y1
	VPXOR   64(SRC1)(POS*1), Y2, Y2
	VPXOR   96(SRC1)(POS*1), Y3, Y3
	LONG    $0xe77da1c4; WORD $0x0304
	LONG    $0xe77da1c4; WORD $0x034c; BYTE $0x20
	LONG    $0xe77da1c4; WORD $0x0354; BYTE $0x40
	LONG    $0xe77da1c4; WORD $0x035c; BYTE $0x60

	ADDQ $128, POS
	CMPQ LEN, POS
	JNE  loop128b
	SFENCE
	VZEROUPPER
	RET

loop_1b:
	MOVB  -1(SRC0)(LEN*1), TMP1
	MOVB  -1(SRC1)(LEN*1), TMP2
	XORB  TMP1, TMP2
	MOVB  TMP2, -1(DST)(LEN*1)
	SUBQ  $1, LEN
	TESTQ $7, LEN
	JNZ   loop_1b
	CMPQ  LEN, $0
	JE    ret
	TESTQ $127, LEN
	JZ    aligned

not_aligned:
	TESTQ $7, LEN
	JNE   loop_1b
	MOVQ  LEN, TMP1
	ANDQ  $127, TMP1

loop_8b:
	MOVQ -8(SRC0)(LEN*1), TMP2
	MOVQ -8(SRC1)(LEN*1), TMP3
	XORQ TMP2, TMP3
	MOVQ TMP3, -8(DST)(LEN*1)
	SUBQ $8, LEN
	SUBQ $8, TMP1
	JG   loop_8b

	CMPQ LEN, $128
	JGE  aligned
	RET

ret:
	RET

// func matrixAVX2small(dst []byte, src [][]byte)
TEXT ·matrixAVX2small(SB), NOSPLIT, $0
	MOVQ  dst+0(FP), DST
	MOVQ  src+24(FP), SRC
	MOVQ  vec+32(FP), VECT
	MOVQ  len+8(FP), LEN
	TESTQ $127, LEN
	JNZ   not_aligned

aligned:
	MOVQ $0, POS

loop128b:
	MOVQ    VECT, TMP1
	SUBQ    $2, TMP1
	MOVQ    $0, TMP2
	MOVQ    (SRC)(TMP2*1), TMP3
	MOVQ    TMP3, TMP4
	VMOVDQU (TMP3)(POS*1), Y0
	VMOVDQU 32(TMP4)(POS*1), Y1
	VMOVDQU 64(TMP3)(POS*1), Y2
	VMOVDQU 96(TMP4)(POS*1), Y3

next_vect:
	ADDQ    $24, TMP2
	MOVQ    (SRC)(TMP2*1), TMP3
	MOVQ    TMP3, TMP4
	VMOVDQU (TMP3)(POS*1), Y4
	VMOVDQU 32(TMP4)(POS*1), Y5
	VMOVDQU 64(TMP3)(POS*1), Y6
	VMOVDQU 96(TMP4)(POS*1), Y7
	VPXOR   Y4, Y0, Y0
	VPXOR   Y5, Y1, Y1
	VPXOR   Y6, Y2, Y2
	VPXOR   Y7, Y3, Y3
	SUBQ    $1, TMP1
	JGE     next_vect

	VMOVDQU Y0, (DST)(POS*1)
	VMOVDQU Y1, 32(DST)(POS*1)
	VMOVDQU Y2, 64(DST)(POS*1)
	VMOVDQU Y3, 96(DST)(POS*1)

	ADDQ $128, POS
	CMPQ LEN, POS
	JNE  loop128b
	VZEROUPPER
	RET

loop_1b:
	MOVQ VECT, TMP1
	MOVQ $0, TMP2
	MOVQ (SRC)(TMP2*1), TMP3
	SUBQ $2, TMP1
	MOVB -1(TMP3)(LEN*1), TMP5

next_vect_1b:
	ADDQ $24, TMP2
	MOVQ (SRC)(TMP2*1), TMP3
	MOVB -1(TMP3)(LEN*1), TMP6
	XORB TMP6, TMP5
	SUBQ $1, TMP1
	JGE  next_vect_1b

	MOVB  TMP5, -1(DST)(LEN*1)
	SUBQ  $1, LEN
	TESTQ $7, LEN
	JNZ   loop_1b

	CMPQ  LEN, $0
	JE    ret
	TESTQ $127, LEN
	JZ    aligned

not_aligned:
	TESTQ $7, LEN
	JNE   loop_1b
	MOVQ  LEN, TMP4
	ANDQ  $127, TMP4

loop_8b:
	MOVQ VECT, TMP1
	MOVQ $0, TMP2
	MOVQ (SRC)(TMP2*1), TMP3
	SUBQ $2, TMP1
	MOVQ -8(TMP3)(LEN*1), TMP5

next_vect_8b:
	ADDQ $24, TMP2
	MOVQ (SRC)(TMP2*1), TMP3
	MOVQ -8(TMP3)(LEN*1), TMP6
	XORQ TMP6, TMP5
	SUBQ $1, TMP1
	JGE  next_vect_8b

	MOVQ TMP5, -8(DST)(LEN*1)
	SUBQ $8, LEN
	SUBQ $8, TMP4
	JG   loop_8b

	CMPQ LEN, $128
	JGE  aligned
	RET

ret:
	RET

// func matrixAVX2big(dst []byte, src [][]byte)
TEXT ·matrixAVX2big(SB), NOSPLIT, $0
	MOVQ  dst+0(FP), DST
	MOVQ  src+24(FP), SRC
	MOVQ  vec+32(FP), VECT
	MOVQ  len+8(FP), LEN
	TESTQ $127, LEN
	JNZ   not_aligned

aligned:
	MOVQ $0, POS

loop128b:
	MOVQ    VECT, TMP1
	SUBQ    $2, TMP1
	MOVQ    $0, TMP2
	MOVQ    (SRC)(TMP2*1), TMP3
	MOVQ    TMP3, TMP4
	VMOVDQU (TMP3)(POS*1), Y0
	VMOVDQU 32(TMP4)(POS*1), Y1
	VMOVDQU 64(TMP3)(POS*1), Y2
	VMOVDQU 96(TMP4)(POS*1), Y3

next_vect:
	ADDQ    $24, TMP2
	MOVQ    (SRC)(TMP2*1), TMP3
	MOVQ    TMP3, TMP4
	VMOVDQU (TMP3)(POS*1), Y4
	VMOVDQU 32(TMP4)(POS*1), Y5
	VMOVDQU 64(TMP3)(POS*1), Y6
	VMOVDQU 96(TMP4)(POS*1), Y7
	VPXOR   Y4, Y0, Y0
	VPXOR   Y5, Y1, Y1
	VPXOR   Y6, Y2, Y2
	VPXOR   Y7, Y3, Y3
	SUBQ    $1, TMP1
	JGE     next_vect

	LONG $0xe77da1c4; WORD $0x0304             // VMOVNTDQ  go1.8 has
	LONG $0xe77da1c4; WORD $0x034c; BYTE $0x20
	LONG $0xe77da1c4; WORD $0x0354; BYTE $0x40
	LONG $0xe77da1c4; WORD $0x035c; BYTE $0x60

	ADDQ $128, POS
	CMPQ LEN, POS
	JNE  loop128b
	VZEROUPPER
	RET

loop_1b:
	MOVQ VECT, TMP1
	MOVQ $0, TMP2
	MOVQ (SRC)(TMP2*1), TMP3
	SUBQ $2, TMP1
	MOVB -1(TMP3)(LEN*1), TMP5

next_vect_1b:
	ADDQ $24, TMP2
	MOVQ (SRC)(TMP2*1), TMP3
	MOVB -1(TMP3)(LEN*1), TMP6
	XORB TMP6, TMP5
	SUBQ $1, TMP1
	JGE  next_vect_1b

	MOVB  TMP5, -1(DST)(LEN*1)
	SUBQ  $1, LEN
	TESTQ $7, LEN
	JNZ   loop_1b

	CMPQ  LEN, $0
	JE    ret
	TESTQ $127, LEN
	JZ    aligned

not_aligned:
	TESTQ $7, LEN
	JNE   loop_1b
	MOVQ  LEN, TMP4
	ANDQ  $127, TMP4

loop_8b:
	MOVQ VECT, TMP1
	MOVQ $0, TMP2
	MOVQ (SRC)(TMP2*1), TMP3
	SUBQ $2, TMP1
	MOVQ -8(TMP3)(LEN*1), TMP5

next_vect_8b:
	ADDQ $24, TMP2
	MOVQ (SRC)(TMP2*1), TMP3
	MOVQ -8(TMP3)(LEN*1), TMP6
	XORQ TMP6, TMP5
	SUBQ $1, TMP1
	JGE  next_vect_8b

	MOVQ TMP5, -8(DST)(LEN*1)
	SUBQ $8, LEN
	SUBQ $8, TMP4
	JG   loop_8b

	CMPQ LEN, $128
	JGE  aligned
	RET

ret:
	RET

