//go:build amd64 && gc && !govet
#include "textflag.h"

// Per-64-bit-lane byte swap mask:
// [7 6 5 4 3 2 1 0 | 15 14 13 12 11 10 9 8]
DATA ·bswap64<>(SB)/8,  $0x0706050403020100
DATA ·bswap64<>+8(SB)/8, $0x0f0e0d0c0b0a0908
GLOBL ·bswap64<>(SB), RODATA, $16

// func mul128CLMUL(a, b Block) (lo, hi Block)
TEXT ·mul128CLMUL(SB), NOSPLIT, $0-64

    // Load a and b
    MOVOU a+0(FP), X0
    MOVOU b+16(FP), X1

    // Convert from little-endian polynomial to CLMUL basis
    PSHUFB ·bswap64<>(SB), X0
    PSHUFB ·bswap64<>(SB), X1

    // p00 = a0*b0
    MOVO X0, X2
    PCLMULQDQ $0x00, X1, X2

    // p11 = a1*b1
    MOVO X0, X3
    PCLMULQDQ $0x11, X1, X3

    // p01 = a0*b1
    MOVO X0, X4
    PCLMULQDQ $0x10, X1, X4

    // p10 = a1*b0
    MOVO X0, X5
    PCLMULQDQ $0x01, X1, X5

    // mid = p01 ^ p10
    PXOR X5, X4

    // lo = p00 ^ (mid << 64)
    MOVO X4, X6
    PSLLDQ $8, X6
    PXOR X6, X2

    // hi = p11 ^ (mid >> 64)
    MOVO X4, X7
    PSRLDQ $8, X7
    PXOR X7, X3

    // Convert back to little-endian polynomial basis
    PSHUFB ·bswap64<>(SB), X2
    PSHUFB ·bswap64<>(SB), X3

    // Store results
    MOVOU X2, lo+32(FP)
    MOVOU X3, hi+48(FP)

    RET
