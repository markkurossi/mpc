//go:build amd64 && gc
#include "textflag.h"

// Per-64-bit-lane byte swap mask:
// [7 6 5 4 3 2 1 0 | 15 14 13 12 11 10 9 8]
DATA ·bswap64<>(SB)/8,  $0x0706050403020100
DATA ·bswap64<>+8(SB)/8, $0x0f0e0d0c0b0a0908
GLOBL ·bswap64<>(SB), RODATA, $16


// func mul128CLMUL(a, b *Block, lo, hi *Block)
TEXT ·mul128CLMUL(SB), NOSPLIT, $0-32

    // Load *a into X0
    MOVQ a+0(FP), AX
    MOVOU (AX), X0

    // Load *b into X1
    MOVQ b+8(FP), BX
    MOVOU (BX), X1

    // Convert to CLMUL polynomial basis
    PSHUFB ·bswap64<>(SB), X0
    PSHUFB ·bswap64<>(SB), X1

    // -----------------------------------------
    // Split lanes:
    // a = [a1 | a0]
    // b = [b1 | b0]
    // -----------------------------------------

    // z0 = a0*b0
    MOVO X0, X2
    PCLMULQDQ $0x00, X1, X2      // z0

    // z2 = a1*b1
    MOVO X0, X3
    PCLMULQDQ $0x11, X1, X3      // z2

    // t0 = a0 ^ a1
    MOVO X0, X4
    PSRLDQ $8, X4
    PXOR X0, X4                 // X4 = a0^a1 in low lane

    // t1 = b0 ^ b1
    MOVO X1, X5
    PSRLDQ $8, X5
    PXOR X1, X5                 // X5 = b0^b1 in low lane

    // z1 = (a0^a1)*(b0^b1)
    PCLMULQDQ $0x00, X5, X4     // z1

    // z1 ^= z0 ^ z2
    PXOR X2, X4
    PXOR X3, X4

    // -----------------------------------------
    // Assemble 256-bit result
    // -----------------------------------------

    // lo = z0 ^ (z1 << 64)
    MOVO X4, X6
    PSLLDQ $8, X6
    PXOR X6, X2                 // X2 = lo

    // hi = z2 ^ (z1 >> 64)
    MOVO X4, X7
    PSRLDQ $8, X7
    PXOR X7, X3                 // X3 = hi

    // Convert back from CLMUL basis
    PSHUFB ·bswap64<>(SB), X2
    PSHUFB ·bswap64<>(SB), X3

    // Store *lo
    MOVQ lo+16(FP), CX
    MOVOU X2, (CX)

    // Store *hi
    MOVQ hi+24(FP), DX
    MOVOU X3, (DX)

    RET
