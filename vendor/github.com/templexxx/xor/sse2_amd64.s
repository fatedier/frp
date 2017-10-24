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

// func bytesSrc0(dst, src0, src1 []byte)
TEXT ·xorSrc0(SB), NOSPLIT, $0
	MOVQ  len+32(FP), LEN
	CMPQ  LEN, $0
	JE    ret
	MOVQ  dst+0(FP), DST
	MOVQ  src0+24(FP), SRC0
	MOVQ  src1+48(FP), SRC1
	TESTQ $15, LEN
	JNZ   not_aligned

aligned:
	MOVQ $0, POS

loop16b:
	MOVOU (SRC0)(POS*1), X0
	XORPD (SRC1)(POS*1), X0
	MOVOU X0, (DST)(POS*1)
	ADDQ  $16, POS
	CMPQ  LEN, POS
	JNE   loop16b
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
	TESTQ $15, LEN
	JZ    aligned

not_aligned:
	TESTQ $7, LEN
	JNE   loop_1b
	MOVQ  LEN, TMP1
	ANDQ  $15, TMP1

loop_8b:
	MOVQ -8(SRC0)(LEN*1), TMP2
	MOVQ -8(SRC1)(LEN*1), TMP3
	XORQ TMP2, TMP3
	MOVQ TMP3, -8(DST)(LEN*1)
	SUBQ $8, LEN
	SUBQ $8, TMP1
	JG   loop_8b

	CMPQ LEN, $16
	JGE  aligned
	RET

ret:
	RET

// func bytesSrc1(dst, src0, src1 []byte)
TEXT ·xorSrc1(SB), NOSPLIT, $0
	MOVQ  len+56(FP), LEN
	CMPQ  LEN, $0
	JE    ret
	MOVQ  dst+0(FP), DST
	MOVQ  src0+24(FP), SRC0
	MOVQ  src1+48(FP), SRC1
	TESTQ $15, LEN
	JNZ   not_aligned

aligned:
	MOVQ $0, POS

loop16b:
	MOVOU (SRC0)(POS*1), X0
	XORPD (SRC1)(POS*1), X0
	MOVOU X0, (DST)(POS*1)
	ADDQ  $16, POS
	CMPQ  LEN, POS
	JNE   loop16b
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
	TESTQ $15, LEN
	JZ    aligned

not_aligned:
	TESTQ $7, LEN
	JNE   loop_1b
	MOVQ  LEN, TMP1
	ANDQ  $15, TMP1

loop_8b:
	MOVQ -8(SRC0)(LEN*1), TMP2
	MOVQ -8(SRC1)(LEN*1), TMP3
	XORQ TMP2, TMP3
	MOVQ TMP3, -8(DST)(LEN*1)
	SUBQ $8, LEN
	SUBQ $8, TMP1
	JG   loop_8b

	CMPQ LEN, $16
	JGE  aligned
	RET

ret:
	RET

// func bytesSSE2mini(dst, src0, src1 []byte, size int)
TEXT ·bytesSSE2mini(SB), NOSPLIT, $0
	MOVQ  len+72(FP), LEN
	CMPQ  LEN, $0
	JE    ret
	MOVQ  dst+0(FP), DST
	MOVQ  src0+24(FP), SRC0
	MOVQ  src1+48(FP), SRC1
	TESTQ $15, LEN
	JNZ   not_aligned

aligned:
	MOVQ $0, POS

loop16b:
	MOVOU (SRC0)(POS*1), X0
	XORPD (SRC1)(POS*1), X0

	// MOVOU (SRC1)(POS*1), X4
	// PXOR X4, X0
	MOVOU X0, (DST)(POS*1)
	ADDQ  $16, POS
	CMPQ  LEN, POS
	JNE   loop16b
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
	TESTQ $15, LEN
	JZ    aligned

not_aligned:
	TESTQ $7, LEN
	JNE   loop_1b
	MOVQ  LEN, TMP1
	ANDQ  $15, TMP1

loop_8b:
	MOVQ -8(SRC0)(LEN*1), TMP2
	MOVQ -8(SRC1)(LEN*1), TMP3
	XORQ TMP2, TMP3
	MOVQ TMP3, -8(DST)(LEN*1)
	SUBQ $8, LEN
	SUBQ $8, TMP1
	JG   loop_8b

	CMPQ LEN, $16
	JGE  aligned
	RET

ret:
	RET

// func bytesSSE2small(dst, src0, src1 []byte, size int)
TEXT ·bytesSSE2small(SB), NOSPLIT, $0
	MOVQ  len+72(FP), LEN
	CMPQ  LEN, $0
	JE    ret
	MOVQ  dst+0(FP), DST
	MOVQ  src0+24(FP), SRC0
	MOVQ  src1+48(FP), SRC1
	TESTQ $63, LEN
	JNZ   not_aligned

aligned:
	MOVQ $0, POS

loop64b:
	MOVOU (SRC0)(POS*1), X0
	MOVOU 16(SRC0)(POS*1), X1
	MOVOU 32(SRC0)(POS*1), X2
	MOVOU 48(SRC0)(POS*1), X3

	MOVOU (SRC1)(POS*1), X4
	MOVOU 16(SRC1)(POS*1), X5
	MOVOU 32(SRC1)(POS*1), X6
	MOVOU 48(SRC1)(POS*1), X7

	PXOR X4, X0
	PXOR X5, X1
	PXOR X6, X2
	PXOR X7, X3

	MOVOU X0, (DST)(POS*1)
	MOVOU X1, 16(DST)(POS*1)
	MOVOU X2, 32(DST)(POS*1)
	MOVOU X3, 48(DST)(POS*1)

	ADDQ $64, POS
	CMPQ LEN, POS
	JNE  loop64b
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
	TESTQ $63, LEN
	JZ    aligned

not_aligned:
	TESTQ $7, LEN
	JNE   loop_1b
	MOVQ  LEN, TMP1
	ANDQ  $63, TMP1

loop_8b:
	MOVQ -8(SRC0)(LEN*1), TMP2
	MOVQ -8(SRC1)(LEN*1), TMP3
	XORQ TMP2, TMP3
	MOVQ TMP3, -8(DST)(LEN*1)
	SUBQ $8, LEN
	SUBQ $8, TMP1
	JG   loop_8b

	CMPQ LEN, $64
	JGE  aligned
	RET

ret:
	RET

// func bytesSSE2big(dst, src0, src1 []byte, size int)
TEXT ·bytesSSE2big(SB), NOSPLIT, $0
	MOVQ  len+72(FP), LEN
	CMPQ  LEN, $0
	JE    ret
	MOVQ  dst+0(FP), DST
	MOVQ  src0+24(FP), SRC0
	MOVQ  src1+48(FP), SRC1
	TESTQ $63, LEN
	JNZ   not_aligned

aligned:
	MOVQ $0, POS

loop64b:
	MOVOU (SRC0)(POS*1), X0
	MOVOU 16(SRC0)(POS*1), X1
	MOVOU 32(SRC0)(POS*1), X2
	MOVOU 48(SRC0)(POS*1), X3

	MOVOU (SRC1)(POS*1), X4
	MOVOU 16(SRC1)(POS*1), X5
	MOVOU 32(SRC1)(POS*1), X6
	MOVOU 48(SRC1)(POS*1), X7

	PXOR X4, X0
	PXOR X5, X1
	PXOR X6, X2
	PXOR X7, X3

	LONG $0xe70f4266; WORD $0x0304             // MOVNTDQ
	LONG $0xe70f4266; WORD $0x034c; BYTE $0x10
	LONG $0xe70f4266; WORD $0x0354; BYTE $0x20
	LONG $0xe70f4266; WORD $0x035c; BYTE $0x30

	ADDQ $64, POS
	CMPQ LEN, POS
	JNE  loop64b
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
	TESTQ $63, LEN
	JZ    aligned

not_aligned:
	TESTQ $7, LEN
	JNE   loop_1b
	MOVQ  LEN, TMP1
	ANDQ  $63, TMP1

loop_8b:
	MOVQ -8(SRC0)(LEN*1), TMP2
	MOVQ -8(SRC1)(LEN*1), TMP3
	XORQ TMP2, TMP3
	MOVQ TMP3, -8(DST)(LEN*1)
	SUBQ $8, LEN
	SUBQ $8, TMP1
	JG   loop_8b

	CMPQ LEN, $64
	JGE  aligned
	RET

ret:
	RET

// func matrixSSE2small(dst []byte, src [][]byte)
TEXT ·matrixSSE2small(SB), NOSPLIT, $0
	MOVQ  dst+0(FP), DST
	MOVQ  src+24(FP), SRC
	MOVQ  vec+32(FP), VECT
	MOVQ  len+8(FP), LEN
	TESTQ $63, LEN
	JNZ   not_aligned

aligned:
	MOVQ $0, POS

loop64b:
	MOVQ  VECT, TMP1
	SUBQ  $2, TMP1
	MOVQ  $0, TMP2
	MOVQ  (SRC)(TMP2*1), TMP3
	MOVQ  TMP3, TMP4
	MOVOU (TMP3)(POS*1), X0
	MOVOU 16(TMP4)(POS*1), X1
	MOVOU 32(TMP3)(POS*1), X2
	MOVOU 48(TMP4)(POS*1), X3

next_vect:
	ADDQ  $24, TMP2
	MOVQ  (SRC)(TMP2*1), TMP3
	MOVQ  TMP3, TMP4
	MOVOU (TMP3)(POS*1), X4
	MOVOU 16(TMP4)(POS*1), X5
	MOVOU 32(TMP3)(POS*1), X6
	MOVOU 48(TMP4)(POS*1), X7
	PXOR  X4, X0
	PXOR  X5, X1
	PXOR  X6, X2
	PXOR  X7, X3
	SUBQ  $1, TMP1
	JGE   next_vect

	MOVOU X0, (DST)(POS*1)
	MOVOU X1, 16(DST)(POS*1)
	MOVOU X2, 32(DST)(POS*1)
	MOVOU X3, 48(DST)(POS*1)

	ADDQ $64, POS
	CMPQ LEN, POS
	JNE  loop64b
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
	TESTQ $63, LEN
	JZ    aligned

not_aligned:
	TESTQ $7, LEN
	JNE   loop_1b
	MOVQ  LEN, TMP4
	ANDQ  $63, TMP4

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

	CMPQ LEN, $64
	JGE  aligned
	RET

ret:
	RET

// func matrixSSE2big(dst []byte, src [][]byte)
TEXT ·matrixSSE2big(SB), NOSPLIT, $0
	MOVQ  dst+0(FP), DST
	MOVQ  src+24(FP), SRC
	MOVQ  vec+32(FP), VECT
	MOVQ  len+8(FP), LEN
	TESTQ $63, LEN
	JNZ   not_aligned

aligned:
	MOVQ $0, POS

loop64b:
	MOVQ  VECT, TMP1
	SUBQ  $2, TMP1
	MOVQ  $0, TMP2
	MOVQ  (SRC)(TMP2*1), TMP3
	MOVQ  TMP3, TMP4
	MOVOU (TMP3)(POS*1), X0
	MOVOU 16(TMP4)(POS*1), X1
	MOVOU 32(TMP3)(POS*1), X2
	MOVOU 48(TMP4)(POS*1), X3

next_vect:
	ADDQ  $24, TMP2
	MOVQ  (SRC)(TMP2*1), TMP3
	MOVQ  TMP3, TMP4
	MOVOU (TMP3)(POS*1), X4
	MOVOU 16(TMP4)(POS*1), X5
	MOVOU 32(TMP3)(POS*1), X6
	MOVOU 48(TMP4)(POS*1), X7
	PXOR  X4, X0
	PXOR  X5, X1
	PXOR  X6, X2
	PXOR  X7, X3
	SUBQ  $1, TMP1
	JGE   next_vect

	LONG $0xe70f4266; WORD $0x0304
	LONG $0xe70f4266; WORD $0x034c; BYTE $0x10
	LONG $0xe70f4266; WORD $0x0354; BYTE $0x20
	LONG $0xe70f4266; WORD $0x035c; BYTE $0x30

	ADDQ $64, POS
	CMPQ LEN, POS
	JNE  loop64b
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
	TESTQ $63, LEN
	JZ    aligned

not_aligned:
	TESTQ $7, LEN
	JNE   loop_1b
	MOVQ  LEN, TMP4
	ANDQ  $63, TMP4

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

	CMPQ LEN, $64
	JGE  aligned
	RET

ret:
	RET

TEXT ·hasSSE2(SB), NOSPLIT, $0
	XORQ AX, AX
	INCL AX
	CPUID
	SHRQ $26, DX
	ANDQ $1, DX
	MOVB DX, ret+0(FP)
	RET

