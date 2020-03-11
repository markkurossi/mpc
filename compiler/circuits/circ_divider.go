//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"

	"github.com/markkurossi/mpc/circuit"
)

type Fraction struct {
	Wires []*Wire
	Size  int
	CC    *Compiler
}

func (f Fraction) String() string {
	return fmt.Sprintf("%d[%d]", f.Size, len(f.Wires))
}

func (f Fraction) Wire(i int) *Wire {
	index := f.Size - 1 - i

	var w *Wire
	if index < len(f.Wires) {
		w = f.Wires[index]
	} else {
		w = NewWire()
		f.CC.Zero(w)
	}
	return w
}

func NewDivider(compiler *Compiler, xa, da, qa, ra []*Wire) error {
	if len(qa) < len(da) {
		return fmt.Errorf("invalid divider arguments: d=%d, q=%d, r=%d",
			len(da), len(qa), len(ra))
	}

	x := Fraction{
		Wires: xa,
		Size:  len(xa)*2 - 1,
		CC:    compiler,
	}
	d := Fraction{
		Wires: da,
		Size:  len(xa),
		CC:    compiler,
	}
	q := Fraction{
		Wires: qa,
		Size:  len(xa),
		CC:    compiler,
	}

	fmt.Printf("*** divider: x=%s, d=%s, q=%s, r=%d\n", x, d, q, len(ra))

	t := NewWire()
	compiler.One(t)

	rIn := make([]*Wire, len(xa))
	rOut := make([]*Wire, len(xa))

	// Init for the first row.
	for i := 1; i < len(xa); i++ {
		rOut[i] = x.Wire(i - 1)
	}

	// Generate matrix.
	for y := 0; y < len(xa); y++ {
		// Init rIn.
		copy(rIn, rOut[1:])

		rIn[len(xa)-1] = x.Wire(len(xa) - 1 + y)

		// XORs left-to-right.
		for x := 0; x < len(xa); x++ {
			rOut[x] = NewWire()

			compiler.AddGate(NewBinary(circuit.XOR, t, d.Wire(x), rOut[x]))
		}

		// Adders right-to-left.
		for x := len(xa) - 1; x >= 0; x-- {
			c := NewWire()

			var ro *Wire
			if y+1 >= len(xa) && len(ra) > x {
				ro = ra[x]
			} else {
				ro = NewWire()
			}
			NewFullAdder(compiler, rOut[x], rIn[x], t, ro, c)
			rOut[x] = ro
			t = c
		}

		// Quotient y
		w := NewWire()
		compiler.AddGate(NewINV(t, w))
		compiler.AddGate(NewINV(w, q.Wire(y)))
	}

	// Extra output bits to zero.
	for i := len(xa); i < len(qa); i++ {
		compiler.Zero(qa[i])
	}
	for i := len(xa); i < len(ra); i++ {
		compiler.Zero(ra[i])
	}

	return nil
}
