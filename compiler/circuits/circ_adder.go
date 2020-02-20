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

	// w1 = XOR(A, B)
	compiler.AddGate(NewBinary(circuit.XOR, a, b, w1))

	// s = XOR(w1, cin)
	compiler.AddGate(NewBinary(circuit.XOR, w1, cin, s))

	// w2 = AND(w1, cin)
	compiler.AddGate(NewBinary(circuit.AND, w1, cin, w2))

	// w3 = AND(A, B)
	compiler.AddGate(NewBinary(circuit.AND, a, b, w3))

	if cout != nil {
		// cout = OR(w2, w3)
		compiler.AddGate(NewBinary(circuit.OR, w2, w3, cout))
	}
}

func NewAdder(compiler *Compiler, x, y, z []*Wire) error {
	if len(x) != len(y) || len(z) < len(x) || len(z) > len(x)+1 {
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
	return nil
}
