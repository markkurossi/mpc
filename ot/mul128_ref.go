//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package ot

func mul128Ref(a, b Label) (lo, hi Label) {
	var r [256]bool

	for i := 0; i < 128; i++ {
		if a.Bit(i) == 0 {
			continue
		}
		for j := 0; j < 128; j++ {
			if b.Bit(j) == 1 {
				r[i+j] = !r[i+j]
			}
		}
	}

	for i := 0; i < 128; i++ {
		if r[i] {
			lo.SetBit(i, 1)
		}
	}
	for i := 128; i < 256; i++ {
		if r[i] {
			hi.SetBit(i-128, 1)
		}
	}
	return
}
