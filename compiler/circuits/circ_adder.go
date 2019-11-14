//
// circ_adder.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"github.com/markkurossi/mpc/circuit"
)

func NewHalfAdder(compiler *Compiler, a, b, s, c *Wire) {
	// S = XOR(A, B)
	compiler.AddGate(NewBinary(circuit.XOR, a, b, s))

	// C = AND(A, B)
	compiler.AddGate(NewBinary(circuit.AND, a, b, c))
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

	// cout = OR(w2, w3)
	compiler.AddGate(NewBinary(circuit.OR, w2, w3, cout))
}
