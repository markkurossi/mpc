//
// circuits.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"github.com/markkurossi/mpc/circuit"
)

type Binary struct {
	Op circuit.Operation
	A  *Wire
	B  *Wire
	O  *Wire
}

func NewBinary(op circuit.Operation, a, b, o *Wire) *Binary {
	gate := &Binary{
		Op: op,
		A:  a,
		B:  b,
		O:  o,
	}
	a.AddOutput(gate)
	b.AddOutput(gate)
	o.SetInput(gate)

	return gate
}

func (g *Binary) String() string {
	return g.Op.String()
}

func (g *Binary) Visit(c *Compiler) {
	if g.A.Assigned && g.B.Assigned {
		c.pending = append(c.pending, g)
	}
}

func (g *Binary) Assign(c *Compiler) {
	g.O.Assign(c)
	c.assigned = append(c.assigned, g)
}

func (g *Binary) Compile(c *Compiler) {
	c.compiled = append(c.compiled, &circuit.Gate{
		ID: c.NextGateID(),
		Inputs: []circuit.Wire{
			circuit.Wire(g.A.ID),
			circuit.Wire(g.B.ID),
		},
		Outputs: []circuit.Wire{
			circuit.Wire(g.O.ID),
		},
		Op: g.Op,
	})
}

type INV struct {
	I *Wire
	O *Wire
}

func NewINV(i, o *Wire) *INV {
	gate := &INV{
		I: i,
		O: o,
	}
	i.AddOutput(gate)
	o.SetInput(gate)

	return gate
}

func (g *INV) String() string {
	return "inv"
}

func (g *INV) Visit(c *Compiler) {
	if g.I.Assigned {
		c.pending = append(c.pending, g)
	}
}

func (g *INV) Assign(c *Compiler) {
	g.O.Assign(c)
	c.assigned = append(c.assigned, g)
}

func (g *INV) Compile(c *Compiler) {
	c.compiled = append(c.compiled, &circuit.Gate{
		ID: c.NextGateID(),
		Inputs: []circuit.Wire{
			circuit.Wire(g.I.ID),
		},
		Outputs: []circuit.Wire{
			circuit.Wire(g.O.ID),
		},
		Op: circuit.INV,
	})
}

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

func NewHalfSubtractor(compiler *Compiler, a, b, diff, bout *Wire) {
	w1 := NewWire()

	compiler.AddGate(NewBinary(circuit.XOR, a, b, diff))
	compiler.AddGate(NewINV(a, w1))
	compiler.AddGate(NewBinary(circuit.AND, w1, b, bout))
}

func NewFullSubtractor(compiler *Compiler, a, b, bin, diff, bout *Wire) {
	w3 := NewWire()
	w4 := NewWire()
	w5 := NewWire()
	w6 := NewWire()
	w7 := NewWire()

	compiler.AddGate(NewBinary(circuit.XOR, a, b, w3))
	compiler.AddGate(NewBinary(circuit.XOR, bin, w3, diff))
	compiler.AddGate(NewINV(a, w4))
	compiler.AddGate(NewBinary(circuit.AND, b, w4, w5))
	compiler.AddGate(NewINV(w3, w6))
	compiler.AddGate(NewBinary(circuit.AND, bin, w6, w7))
	compiler.AddGate(NewBinary(circuit.OR, w5, w7, bout))
}
