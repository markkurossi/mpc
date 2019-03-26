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

type Compiler struct {
	N1         int
	N2         int
	N3         int
	Inputs     []*Wire
	Outputs    []*Wire
	Gates      []Gate
	nextWireID uint32
	nextGateID uint32
	pending    []Gate
	assigned   []Gate
	compiled   []*circuit.Gate
}

func NewCompiler(n1, n2, n3 int) *Compiler {
	result := &Compiler{
		N1: n1,
		N2: n2,
		N3: n3,
	}

	for i := 0; i < n1+n2; i++ {
		result.Inputs = append(result.Inputs, NewWire())
	}
	for i := 0; i < n3; i++ {
		result.Outputs = append(result.Outputs, NewOutputWire())
	}

	return result
}

func (c *Compiler) AddGate(gate Gate) {
	c.Gates = append(c.Gates, gate)
}

func (c *Compiler) NextWireID() uint32 {
	ret := c.nextWireID
	c.nextWireID++
	return ret
}

func (c *Compiler) NextGateID() uint32 {
	ret := c.nextGateID
	c.nextGateID++
	return ret
}

func (c *Compiler) Compile() *circuit.Circuit {
	for _, w := range c.Inputs {
		w.Assign(c)
	}
	for len(c.pending) > 0 {
		gate := c.pending[0]
		c.pending = c.pending[1:]
		gate.Assign(c)
	}
	// Assign outputs.
	for _, w := range c.Outputs {
		if w.Assigned {
			panic("Output already assigned")
		}
		w.ID = c.NextWireID()
		w.Assigned = true
	}

	// Compile circuit.
	for _, gate := range c.assigned {
		gate.Compile(c)
	}

	result := &circuit.Circuit{
		NumGates: int(c.nextGateID),
		NumWires: int(c.nextWireID),
		N1:       c.N1,
		N2:       c.N2,
		N3:       c.N3,
		Gates:    make(map[int]*circuit.Gate),
	}

	for _, gate := range c.compiled {
		result.Gates[int(gate.ID)] = gate
	}

	return result
}

type Wire struct {
	Output   bool
	Assigned bool
	ID       uint32
	Input    Gate
	Outputs  []Gate
}

func NewWire() *Wire {
	return &Wire{}
}

func NewOutputWire() *Wire {
	return &Wire{
		Output: true,
	}
}

func (w *Wire) Assign(c *Compiler) {
	if w.Output {
		return
	}
	if !w.Assigned {
		w.ID = c.NextWireID()
		w.Assigned = true
	}
	for _, output := range w.Outputs {
		output.Visit(c)
	}
}

func (w *Wire) SetInput(gate Gate) {
	if w.Input != nil {
		panic("Input gate already set")
	}
	w.Input = gate
}

func (w *Wire) AddOutput(gate Gate) {
	w.Outputs = append(w.Outputs, gate)
}

type Gate interface {
	String() string
	Visit(c *Compiler)
	Assign(c *Compiler)
	Compile(c *Compiler)
}

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

func (g *INV) String() string {
	return "inv"
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
