//
// circ_subtractor.go
//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"github.com/markkurossi/mpc/circuit"
)

// NewFullSubtractor creates a full subtractor circuit.
func NewFullSubtractor(compiler *Compiler, x, y, cin, d, cout *Wire) {
	w1 := NewWire()
	compiler.AddGate(NewBinary(circuit.XNOR, y, cin, w1))
	compiler.AddGate(NewBinary(circuit.XNOR, x, w1, d))

	if cout != nil {
		w2 := NewWire()
		compiler.AddGate(NewBinary(circuit.XOR, x, cin, w2))

		w3 := NewWire()
		compiler.AddGate(NewBinary(circuit.AND, w1, w2, w3))

		compiler.AddGate(NewBinary(circuit.XOR, w3, cin, cout))
	}
}

// NewSubtractor creates a new subtractor circuit implementing z=x-y.
func NewSubtractor(compiler *Compiler, x, y, z []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(x) > len(z) {
		x = x[0:len(z)]
		y = y[0:len(z)]
	}
	cin := compiler.ZeroWire()

	for i := 0; i < len(x); i++ {
		var cout *Wire
		if i+1 >= len(x) {
			if i+1 >= len(z) {
				// N-N=N, overflow, drop carry bit.
				cout = nil
			} else {
				cout = z[i+1]
			}
		} else {
			cout = NewWire()
		}

		// Note y-x here.
		NewFullSubtractor(compiler, y[i], x[i], cin, z[i], cout)

		cin = cout
	}
	for i := len(x) + 1; i < len(z); i++ {
		z[i] = compiler.ZeroWire()
	}
	return nil
}
