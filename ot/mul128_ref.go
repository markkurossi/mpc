//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package ot

func mul128Ref(a, b Label) (lo, hi Label) {
	var r [256]bool

	get := func(x Label, i int) bool {
		if i < 64 {
			return (x.D0>>i)&1 == 1
		}
		return (x.D1>>(i-64))&1 == 1
	}

	for i := 0; i < 128; i++ {
		if !get(a, i) {
			continue
		}
		for j := 0; j < 128; j++ {
			if get(b, j) {
				r[i+j] = !r[i+j]
			}
		}
	}

	for i := 0; i < 128; i++ {
		if r[i] {
			if i < 64 {
				lo.D0 |= 1 << i
			} else {
				lo.D1 |= 1 << (i - 64)
			}
		}
	}
	for i := 128; i < 256; i++ {
		if r[i] {
			if i < 192 {
				hi.D0 |= 1 << (i - 128)
			} else {
				hi.D1 |= 1 << (i - 192)
			}
		}
	}
	return
}
