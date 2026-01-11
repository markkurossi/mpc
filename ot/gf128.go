//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package ot

// vectorInnPrdtSumNoRed computes the GF(2^128) inner product of
// vectors a and b without modular reduction. It multiplies
// corresponding elements up to min(len(a), len(b)) and XORs the
// 256-bit products together. The result is returned as two 128-bit
// Labels (low and high halves).
func vectorInnPrdtSumNoRed(a, b []Label) (Label, Label) {
	var r1, r2 Label

	n := len(a)
	if n > len(b) {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		lo, hi := mul128(a[i], b[i])
		r1.Xor(lo)
		r2.Xor(hi)
	}
	return r1, r2
}
