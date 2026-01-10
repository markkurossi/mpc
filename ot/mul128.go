//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

//go:build !amd64 || !gc

package ot

func mul128(a, b Block) (lo, hi Block) {
	return mul128Generic(a, b)
}
