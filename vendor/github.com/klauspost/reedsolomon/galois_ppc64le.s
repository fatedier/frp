//+build !noasm !appengine !gccgo

// Copyright 2015, Klaus Post, see LICENSE for details.
// Copyright 2018, Minio, Inc.

#include "textflag.h"

#define LOW       R3
#define HIGH      R4
#define IN        R5
#define LEN       R6
#define OUT       R7
#define CONSTANTS R8
#define OFFSET    R9
#define OFFSET1   R10
#define OFFSET2   R11

#define X6        VS34
#define X6_       V2
#define X7        VS35
#define X7_       V3
#define MSG       VS36
#define MSG_      V4
#define MSG_HI    VS37
#define MSG_HI_   V5
#define RESULT    VS38
#define RESULT_   V6
#define ROTATE    VS39
#define ROTATE_   V7
#define MASK      VS40
#define MASK_     V8
#define FLIP      VS41
#define FLIP_     V9


// func galMulPpc(low, high, in, out []byte)
TEXT ·galMulPpc(SB), NOFRAME|NOSPLIT, $0-96
    MOVD    low+0(FP), LOW
    MOVD    high+24(FP), HIGH
    MOVD    in+48(FP), IN
    MOVD    in_len+56(FP), LEN
    MOVD    out+72(FP), OUT

    MOVD    $16, OFFSET1
    MOVD    $32, OFFSET2

    MOVD    $·constants(SB), CONSTANTS
    LXVD2X  (CONSTANTS)(R0), ROTATE
    LXVD2X  (CONSTANTS)(OFFSET1), MASK
    LXVD2X  (CONSTANTS)(OFFSET2), FLIP

    LXVD2X  (LOW)(R0), X6
    LXVD2X  (HIGH)(R0), X7
    VPERM   X6_, V31, FLIP_, X6_
    VPERM   X7_, V31, FLIP_, X7_

    MOVD    $0, OFFSET

loop:
    LXVD2X  (IN)(OFFSET), MSG

    VSRB    MSG_, ROTATE_, MSG_HI_
    VAND    MSG_, MASK_, MSG_
    VPERM   X6_, V31, MSG_, MSG_
    VPERM   X7_, V31, MSG_HI_, MSG_HI_

    VXOR    MSG_, MSG_HI_, MSG_

    STXVD2X MSG, (OUT)(OFFSET)

    ADD     $16, OFFSET, OFFSET
    CMP     LEN, OFFSET
    BGT     loop
    RET


// func galMulPpcXorlow, high, in, out []byte)
TEXT ·galMulPpcXor(SB), NOFRAME|NOSPLIT, $0-96
    MOVD    low+0(FP), LOW
    MOVD    high+24(FP), HIGH
    MOVD    in+48(FP), IN
    MOVD    in_len+56(FP), LEN
    MOVD    out+72(FP), OUT

    MOVD    $16, OFFSET1
    MOVD    $32, OFFSET2

    MOVD    $·constants(SB), CONSTANTS
    LXVD2X  (CONSTANTS)(R0), ROTATE
    LXVD2X  (CONSTANTS)(OFFSET1), MASK
    LXVD2X  (CONSTANTS)(OFFSET2), FLIP

    LXVD2X  (LOW)(R0), X6
    LXVD2X  (HIGH)(R0), X7
    VPERM   X6_, V31, FLIP_, X6_
    VPERM   X7_, V31, FLIP_, X7_

    MOVD    $0, OFFSET

loopXor:
    LXVD2X  (IN)(OFFSET), MSG
    LXVD2X  (OUT)(OFFSET), RESULT

    VSRB    MSG_, ROTATE_, MSG_HI_
    VAND    MSG_, MASK_, MSG_
    VPERM   X6_, V31, MSG_, MSG_
    VPERM   X7_, V31, MSG_HI_, MSG_HI_

    VXOR    MSG_, MSG_HI_, MSG_
    VXOR    MSG_, RESULT_, RESULT_

    STXVD2X RESULT, (OUT)(OFFSET)

    ADD     $16, OFFSET, OFFSET
    CMP     LEN, OFFSET
    BGT     loopXor
    RET

DATA ·constants+0x0(SB)/8, $0x0404040404040404
DATA ·constants+0x8(SB)/8, $0x0404040404040404
DATA ·constants+0x10(SB)/8, $0x0f0f0f0f0f0f0f0f
DATA ·constants+0x18(SB)/8, $0x0f0f0f0f0f0f0f0f
DATA ·constants+0x20(SB)/8, $0x0706050403020100
DATA ·constants+0x28(SB)/8, $0x0f0e0d0c0b0a0908

GLOBL ·constants(SB), 8, $48
