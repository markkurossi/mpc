//
// circ_multiplier.go
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

// NewMultiplier creates a multiplier circuit implementing x*y=z. This
// function implements Array Multiplier Circuit.
func NewMultiplier(compiler *Compiler, x, y, z []*Wire) error {
	if len(x) != len(y) || len(x)+len(y) != len(z) {
		return fmt.Errorf("Invalid multiplier arguments: x=%d, y=%d, z=%d",
			len(x), len(y), len(z))
	}

	// One bit multiplication is AND.
	if len(x) == 1 {
		compiler.AddGate(NewBinary(circuit.AND, x[0], y[0], z[0]))
		return nil
	}

	var sums []*Wire

	// Construct Y0 sums
	for i, xn := range x {
		var s *Wire
		if i == 0 {
			s = z[0]
		} else {
			s = NewWire()
			sums = append(sums, s)
		}
		compiler.AddGate(NewBinary(circuit.AND, xn, y[0], s))
	}

	// Construct len(y)-2 intermediate layers
	var j int
	for j = 1; j+1 < len(y); j++ {
		// ANDs for y(j)
		var ands []*Wire
		for _, xn := range x {
			wire := NewWire()
			compiler.AddGate(NewBinary(circuit.AND, xn, y[j], wire))
			ands = append(ands, wire)
		}

		// Compute next sums.
		var nsums []*Wire
		var c *Wire
		for i := 0; i < len(ands); i++ {
			cout := NewWire()

			var s *Wire
			if i == 0 {
				s = z[j]
			} else {
				s = NewWire()
				nsums = append(nsums, s)
			}

			if i == 0 {
				NewHalfAdder(compiler, ands[i], sums[i], s, cout)
			} else if i >= len(sums) {
				NewHalfAdder(compiler, ands[i], c, s, cout)
			} else {
				NewFullAdder(compiler, ands[i], sums[i], c, s, cout)
			}
			c = cout
		}
		// New sums with carry as the highest bit.
		sums = append(nsums, c)
	}

	// Construct final layer.
	var c *Wire
	for i, xn := range x {
		and := NewWire()
		compiler.AddGate(NewBinary(circuit.AND, xn, y[j], and))

		var cout *Wire
		if i+1 >= len(sums) {
			cout = z[len(z)-1]
		} else {
			cout = NewWire()
		}

		if i == 0 {
			NewHalfAdder(compiler, and, sums[i], z[j+i], cout)
		} else {
			NewFullAdder(compiler, and, sums[i], c, z[j+i], cout)
		}
		c = cout
	}

	return nil
}
