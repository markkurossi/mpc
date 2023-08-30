//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package circuits

// NewDivider creates a division circuit computing r=a/b, q=a%b.
func NewDivider(cc *Compiler, a, b, q, r []*Wire) error {
	a, b = cc.ZeroPad(a, b)

	rIn := make([]*Wire, len(b)+1)
	rOut := make([]*Wire, len(b)+1)

	// Init bINV.
	bINV := make([]*Wire, len(b))
	for i := 0; i < len(b); i++ {
		bINV[i] = cc.Calloc.Wire()
		cc.INV(b[i], bINV[i])
	}

	// Init for the first row.
	for i := 0; i < len(b); i++ {
		rOut[i] = cc.ZeroWire()
	}

	// Generate matrix.
	for y := 0; y < len(a); y++ {
		// Init rIn.
		rIn[0] = a[len(a)-1-y]
		copy(rIn[1:], rOut)

		// Adders from b{0} to b{n-1}, 0
		cIn := cc.OneWire()
		for x := 0; x < len(b)+1; x++ {
			var bw *Wire
			if x < len(b) {
				bw = bINV[x]
			} else {
				bw = cc.OneWire() // INV(0)
			}
			co := cc.Calloc.Wire()
			ro := cc.Calloc.Wire()
			NewFullAdder(cc, rIn[x], bw, cIn, ro, co)
			rOut[x] = ro
			cIn = co
		}

		// Quotient y.
		if len(a)-1-y < len(q) {
			w := cc.Calloc.Wire()
			cc.INV(cIn, w)
			cc.INV(w, q[len(a)-1-y])
		}

		// MUXes from high to low bit.
		for x := len(b); x >= 0; x-- {
			var ro *Wire
			if y+1 >= len(a) && x < len(r) {
				ro = r[x]
			} else {
				ro = cc.Calloc.Wire()
			}

			err := NewMUX(cc, []*Wire{cIn}, rOut[x:x+1], rIn[x:x+1],
				[]*Wire{ro})
			if err != nil {
				return err
			}
			rOut[x] = ro
		}
	}

	// Set extra quotient bits to zero.
	for y := len(a); y < len(q); y++ {
		q[y] = cc.ZeroWire()
	}

	// Set extra remainder bits to zero.
	for x := len(b); x < len(r); x++ {
		r[x] = cc.ZeroWire()
	}

	return nil
}
