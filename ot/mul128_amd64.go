//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

//go:build amd64 && gc

package ot

//go:noescape
func mul128CLMUL(a, b *Label, lo, hi *Label)

func mul128(a, b Label) (Label, Label) {
	var lo, hi Label
	mul128CLMUL(&a, &b, &lo, &hi)
	return lo, hi
}
