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

type Gate struct {
	Op       circuit.Operation
	Visited  bool
	Compiled bool
	Dead     bool
	A        *Wire
	B        *Wire
	O        *Wire
}

func NewBinary(op circuit.Operation, a, b, o *Wire) *Gate {
	gate := &Gate{
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

func (g *Gate) String() string {
	return fmt.Sprintf("%s %d %d %d", g.Op, g.A.ID, g.B.ID, g.O.ID)
}

func (g *Gate) Visit(c *Compiler) {
	if !g.Dead && !g.Visited && g.A.Assigned() && g.B.Assigned() {
		g.Visited = true
		c.pending = append(c.pending, g)
	}
}

func (g *Gate) Prune() bool {
	if g.Dead || g.O.Output || g.O.NumOutputs > 0 {
		return false
	}
	g.Dead = true
	g.A.RemoveOutput(g)
	g.B.RemoveOutput(g)
	return true
}

func (g *Gate) Assign(c *Compiler) {
	if !g.Dead {
		g.O.Assign(c)
		c.assigned = append(c.assigned, g)
	}
}

func (g *Gate) Compile(c *Compiler) {
	if g.Dead || g.Compiled {
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
