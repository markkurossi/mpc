//
// gates.go
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

type Binary struct {
	Op       circuit.Operation
	A        *Wire
	B        *Wire
	O        *Wire
	Compiled bool
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
	return fmt.Sprintf("%s %d %d %d", g.Op, g.A.ID, g.B.ID, g.O.ID)
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
	if g.Compiled {
		return
	}
	g.Compiled = true
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
	return fmt.Sprintf("INV %d %d", g.I.ID, g.O.ID)
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
