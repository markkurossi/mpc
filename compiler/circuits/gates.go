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
	Visited  bool
	Compiled bool
	A        *Wire
	B        *Wire
	O        *Wire
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
	if !g.Visited && g.A.Assigned && g.B.Assigned {
		g.Visited = true
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
	c.compiled = append(c.compiled, circuit.Gate{
		Input0: circuit.Wire(g.A.ID),
		Input1: circuit.Wire(g.B.ID),
		Output: circuit.Wire(g.O.ID),
		Op:     g.Op,
	})
}
