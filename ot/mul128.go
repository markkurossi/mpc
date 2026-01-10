//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

//go:build !amd64 || !gc

package ot

func mul128(a, b Label) (lo, hi Label) {
	return mul128Generic(a, b)
}
