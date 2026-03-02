//
// circ_adder.go
//
// Copyright (c) 2019-2026 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"math"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
)

// NewHalfAdder creates a half adder circuit.
func NewHalfAdder(cc *Compiler, a, b, s, c *Wire) {
	// S = XOR(A, B)
	cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, a, b, s))

	if c != nil {
		// C = AND(A, B)
		cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, a, b, c))
	}
}

// NewFullAdder creates a full adder circuit
func NewFullAdder(cc *Compiler, a, b, cin, s, cout *Wire) {
	w1 := cc.Calloc.Wire()
	w2 := cc.Calloc.Wire()
	w3 := cc.Calloc.Wire()

	// s = a XOR b XOR cin
	// cout = cin XOR ((a XOR cin) AND (b XOR cin)).

	// w1 = XOR(b, cin)
	cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, b, cin, w1))

	// s = XOR(a, w1)
	cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, a, w1, s))

	if cout != nil {
		// w2 = XOR(a, cin)
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, a, cin, w2))

		// w3 = AND(w1, w2)
		cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, w1, w2, w3))

		// cout = XOR(cin, w3)
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, cin, w3, cout))
	}
}

// NewAdder creates a new adder circuit implementing z=x+y.
func NewAdder(cc *Compiler, x, y, z []*Wire) error {
	if cc.Params.Target == utils.TargetGMW {
		return NewKoggeStoneAdder(cc, x, y, z)
	}

	x, y = cc.ZeroPad(x, y)
	if len(x) > len(z) {
		x = x[0:len(z)]
		y = y[0:len(z)]
	}

	if len(x) == 1 {
		var cin *Wire
		if len(z) > 1 {
			cin = z[1]
		}
		NewHalfAdder(cc, x[0], y[0], z[0], cin)
	} else {
		cin := cc.Calloc.Wire()
		NewHalfAdder(cc, x[0], y[0], z[0], cin)

		for i := 1; i < len(x); i++ {
			var cout *Wire
			if i+1 >= len(x) {
				if i+1 >= len(z) {
					// N+N=N, overflow, drop carry bit.
					cout = nil
				} else {
					cout = z[len(x)]
				}
			} else {
				cout = cc.Calloc.Wire()
			}

			NewFullAdder(cc, x[i], y[i], cin, z[i], cout)

			cin = cout
		}
	}

	// Set all leftover bits to zero.
	for i := len(x) + 1; i < len(z); i++ {
		z[i] = cc.ZeroWire()
	}

	return nil
}

// NewKoggeStoneAdder creates a Kogge-Stone parallel prefix adder.  It
// achieves O(log n) AND-depth at the cost of a larger gate count.
func NewKoggeStoneAdder(cc *Compiler, x, y, z []*Wire) error {
	n := len(x)
	if n < len(y) {
		n = len(y)
	}
	if len(z) > n {
		n++
	}
	x = cc.Pad(x, n)
	y = cc.Pad(y, n)

	if len(x) > len(z) {
		x = x[0:len(z)]
		y = y[0:len(z)]
		n = len(z)
	}
	p := make([]*Wire, n)
	g := make([]*Wire, n)

	// Pre-processing.
	for i := 0; i < n; i++ {
		p[i] = cc.Calloc.Wire()
		g[i] = cc.Calloc.Wire()

		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, x[i], y[i], p[i]))
		cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, x[i], y[i], g[i]))
	}

	// Prefix network (logarithmic depth).
	numStages := int(math.Ceil(math.Log2(float64(n))))
	for s := 0; s < numStages; s++ {
		newP := make([]*Wire, n)
		newG := make([]*Wire, n)
		shift := 1 << s

		for i := 0; i < n; i++ {
			if i < shift {
				newP[i], newG[i] = p[i], g[i]
			} else {
				// Black cell logic.

				newG[i] = cc.Calloc.Wire()
				newP[i] = cc.Calloc.Wire()
				and := cc.Calloc.Wire()

				cc.AddGate(cc.Calloc.BinaryGate(
					circuit.AND, p[i], g[i-shift], and))
				cc.OR(g[i], and, newG[i])
				cc.AddGate(cc.Calloc.BinaryGate(
					circuit.AND, p[i], p[i-shift], newP[i]))
			}
		}
		p, g = newP, newG
	}

	// Post-processing.
	for i := 0; i < n; i++ {
		if i == 0 {
			cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, x[i], y[i], z[i]))
		} else {
			xor := cc.Calloc.Wire()
			cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, x[i], y[i], xor))
			cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, xor, g[i-1], z[i]))
		}
	}
	// Set all leftover bits to zero.
	for i := n; i < len(z); i++ {
		z[i] = cc.ZeroWire()
	}

	return nil
}
