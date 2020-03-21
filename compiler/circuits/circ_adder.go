//
// circ_adder.go
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

func NewHalfAdder(compiler *Compiler, a, b, s, c *Wire) {
	// S = XOR(A, B)
	compiler.AddGate(NewBinary(circuit.XOR, a, b, s))

	if c != nil {
		// C = AND(A, B)
		compiler.AddGate(NewBinary(circuit.AND, a, b, c))
	}
}

func NewFullAdder(compiler *Compiler, a, b, cin, s, cout *Wire) {
	w1 := NewWire()
	w2 := NewWire()
	w3 := NewWire()

	// s = a XOR b XOR cin
	// cout = cin XOR ((a XOR cin) AND (b XOR cin)).

	// w1 = XOR(b, cin)
	compiler.AddGate(NewBinary(circuit.XOR, b, cin, w1))

	// s = XOR(a, w1)
	compiler.AddGate(NewBinary(circuit.XOR, a, w1, s))

	if cout != nil {
		// w2 = XOR(a, cin)
		compiler.AddGate(NewBinary(circuit.XOR, a, cin, w2))

		// w3 = AND(w1, w2)
		compiler.AddGate(NewBinary(circuit.AND, w1, w2, w3))

		// cout = XOR(cin, w3)
		compiler.AddGate(NewBinary(circuit.XOR, cin, w3, cout))
	}
}

func NewAdder(compiler *Compiler, x, y, z []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(z) < len(x) {
		return fmt.Errorf("Invalid adder arguments: x=%d, y=%d, z=%d",
			len(x), len(y), len(z))
	}

	if len(x) == 1 {
		var cin *Wire
		if len(z) > 1 {
			cin = z[1]
		}
		NewHalfAdder(compiler, x[0], y[0], z[0], cin)
	} else {
		cin := NewWire()
		NewHalfAdder(compiler, x[0], y[0], z[0], cin)

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
				cout = NewWire()
			}

			NewFullAdder(compiler, x[i], y[i], cin, z[i], cout)

			cin = cout
		}
	}

	// Set all leftover bits to zero.
	for i := len(x) + 1; i < len(z); i++ {
		compiler.Zero(z[i])
	}

	return nil
}
