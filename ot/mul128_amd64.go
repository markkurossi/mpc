//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

//go:build amd64 && gc

package ot

//go:noescape
func mul128CLMUL(a, b *Block, lo, hi *Block)

func mul128(a, b Block) (Block, Block) {
	var lo, hi Block
	mul128CLMUL(&a, &b, &lo, &hi)
	return lo, hi
}
