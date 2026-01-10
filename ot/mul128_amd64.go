//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

//go:build amd64 && gc

package ot

func mul128CLMUL(a, b Block) (lo, hi Block)

//go:nosplit
func mul128(a, b Block) (Block, Block) {
	return mul128CLMUL(a, b)
}
