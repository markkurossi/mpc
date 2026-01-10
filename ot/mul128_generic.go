//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package ot

func mul128Generic(a, b Label) (lo, hi Label) {
	a0, a1 := a.D0, a.D1
	b0, b1 := b.D0, b.D1

	p00lo, p00hi := clmul64(a0, b0)
	p01lo, p01hi := clmul64(a0, b1)
	p10lo, p10hi := clmul64(a1, b0)
	p11lo, p11hi := clmul64(a1, b1)

	midLo := p01lo ^ p10lo
	midHi := p01hi ^ p10hi

	lo.D0 = p00lo
	lo.D1 = p00hi ^ midLo

	hi.D0 = midHi ^ p11lo
	hi.D1 = p11hi

	return
}
