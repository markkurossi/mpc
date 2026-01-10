//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package ot

// vectorInnPrdtSumNoRed computes the GF(2^128) inner product
// of vectors a and b without modular reduction.
// It returns the 256-bit result as two 128-bit blocks.
func vectorInnPrdtSumNoRed(a, b []Label) (Label, Label) {
	var r1, r2 Label // zero initialized

	n := len(a)
	for i := 0; i < n; i++ {
		lo, hi := mul128(a[i], b[i])
		r1.Xor(lo)
		r2.Xor(hi)
	}
	return r1, r2
}

func clmul64(a, b uint64) (lo, hi uint64) {
	for i := 0; i < 64; i++ {
		if (b>>i)&1 != 0 {
			if i == 0 {
				lo ^= a
			} else {
				lo ^= a << i
				hi ^= a >> (64 - i)
			}
		}
	}
	return
}
