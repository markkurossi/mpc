//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package ot

func mul128Generic(a, b Block) (lo, hi Block) {
	a0, a1 := a.Lo, a.Hi
	b0, b1 := b.Lo, b.Hi

	p00lo, p00hi := clmul64(a0, b0)
	p01lo, p01hi := clmul64(a0, b1)
	p10lo, p10hi := clmul64(a1, b0)
	p11lo, p11hi := clmul64(a1, b1)

	midLo := p01lo ^ p10lo
	midHi := p01hi ^ p10hi

	lo.Lo = p00lo
	lo.Hi = p00hi ^ midLo

	hi.Lo = midHi ^ p11lo
	hi.Hi = p11hi

	return
}
