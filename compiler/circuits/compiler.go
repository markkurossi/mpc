//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"
	"time"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
)

// Builtin implements a buitin circuit that uses input wires a and b
// and returns the circuit result in r.
type Builtin func(cc *Compiler, a, b, r []*Wire) error

// Compiler implements binary circuit compiler.
type Compiler struct {
	Params          *utils.Params
	OutputsAssigned bool
	Inputs          circuit.IO
	Outputs         circuit.IO
	InputWires      []*Wire
	OutputWires     []*Wire
	Gates           []*Gate
	nextWireID      uint32
	pending         []*Gate
	assigned        []*Gate
	compiled        []circuit.Gate
	wiresX          map[string][]*Wire
	invI0Wire       *Wire
	zeroWire        *Wire
	oneWire         *Wire
}

// NewCompiler creates a new circuit compiler for the specified
// circuit input and output values.
func NewCompiler(params *utils.Params, inputs, outputs circuit.IO,
	inputWires, outputWires []*Wire) (*Compiler, error) {

	if len(inputWires) == 0 {
		return nil, fmt.Errorf("no inputs defined")
	}
	return &Compiler{
		Params:      params,
		Inputs:      inputs,
		Outputs:     outputs,
		InputWires:  inputWires,
		OutputWires: outputWires,
		Gates:       make([]*Gate, 0, 65536),
	}, nil
}

// InvI0Wire returns a wire holding value INV(input[0]).
func (c *Compiler) InvI0Wire() *Wire {
	if c.invI0Wire == nil {
		c.invI0Wire = NewWire()
		c.AddGate(NewINV(c.InputWires[0], c.invI0Wire))
	}
	return c.invI0Wire
}

// ZeroWire returns a wire holding value 0.
func (c *Compiler) ZeroWire() *Wire {
	if c.zeroWire == nil {
		c.zeroWire = NewWire()
		c.AddGate(NewBinary(circuit.AND, c.InputWires[0], c.InvI0Wire(),
			c.zeroWire))
		c.zeroWire.SetValue(Zero)
	}
	return c.zeroWire
}

// OneWire returns a wire holding value 1.
func (c *Compiler) OneWire() *Wire {
	if c.oneWire == nil {
		c.oneWire = NewWire()
		c.AddGate(NewBinary(circuit.OR, c.InputWires[0], c.InvI0Wire(),
			c.oneWire))
		c.oneWire.SetValue(One)
	}
	return c.oneWire
}

// ZeroPad pads the argument wires x and y with zero values so that
// the resulting wires have the same number of bits.
func (c *Compiler) ZeroPad(x, y []*Wire) ([]*Wire, []*Wire) {
	if len(x) == len(y) {
		return x, y
	}

	max := len(x)
	if len(y) > max {
		max = len(y)
	}

	rx := make([]*Wire, max)
	for i := 0; i < max; i++ {
		if i < len(x) {
			rx[i] = x[i]
		} else {
			rx[i] = c.ZeroWire()
		}
	}

	ry := make([]*Wire, max)
	for i := 0; i < max; i++ {
		if i < len(y) {
			ry[i] = y[i]
		} else {
			ry[i] = c.ZeroWire()
		}
	}

	return rx, ry
}

// ShiftLeft shifts the size number of bits of the input wires w,
// count bits left.
func (c *Compiler) ShiftLeft(w []*Wire, size, count int) []*Wire {
	result := make([]*Wire, size)

	if count < size {
		copy(result[count:], w)
	}
	for i := 0; i < count; i++ {
		result[i] = c.ZeroWire()
	}
	for i := count + len(w); i < size; i++ {
		result[i] = c.ZeroWire()
	}
	return result
}

// INV creates an inverse wire inverting the input wire i's value to
// the output wire o.
func (c *Compiler) INV(i, o *Wire) {
	c.AddGate(NewBinary(circuit.XOR, i, c.OneWire(), o))
}

// ID creates an identity wire passing the input wire i's value to the
// output wire o.
func (c *Compiler) ID(i, o *Wire) {
	c.AddGate(NewBinary(circuit.XOR, i, c.ZeroWire(), o))
}

// AddGate adds a get into the circuit.
func (c *Compiler) AddGate(gate *Gate) {
	c.Gates = append(c.Gates, gate)
}

// SetNextWireID sets the next unique wire ID to use.
func (c *Compiler) SetNextWireID(next uint32) {
	c.nextWireID = next
}

// NextWireID returns the next unique wire ID.
func (c *Compiler) NextWireID() uint32 {
	ret := c.nextWireID
	c.nextWireID++
	return ret
}

// ConstPropagate propagates constant wire values in the circuit and
// short circuits gates if their output does not depend on the gate's
// logical operation.
func (c *Compiler) ConstPropagate() {
	var stats circuit.Stats

	start := time.Now()

	for _, g := range c.Gates {
		switch g.Op {
		case circuit.XOR:
			if (g.A.Value() == Zero && g.B.Value() == Zero) ||
				(g.A.Value() == One && g.B.Value() == One) {
				g.O.SetValue(Zero)
				stats[g.Op]++
			} else if (g.A.Value() == Zero && g.B.Value() == One) ||
				(g.A.Value() == One && g.B.Value() == Zero) {
				g.O.SetValue(One)
				stats[g.Op]++
			} else if g.A.Value() == Zero {
				// O = B
				stats[g.Op]++
				g.ShortCircuit(g.B)
			} else if g.B.Value() == Zero {
				// O = A
				stats[g.Op]++
				g.ShortCircuit(g.A)
			}

		case circuit.XNOR:
			if (g.A.Value() == Zero && g.B.Value() == Zero) ||
				(g.A.Value() == One && g.B.Value() == One) {
				g.O.SetValue(One)
				stats[g.Op]++
			} else if (g.A.Value() == Zero && g.B.Value() == One) ||
				(g.A.Value() == One && g.B.Value() == Zero) {
				g.O.SetValue(Zero)
				stats[g.Op]++
			}

		case circuit.AND:
			if g.A.Value() == Zero || g.B.Value() == Zero {
				g.O.SetValue(Zero)
				stats[g.Op]++
			} else if g.A.Value() == One && g.B.Value() == One {
				g.O.SetValue(One)
				stats[g.Op]++
			} else if g.A.Value() == One {
				// O = B
				stats[g.Op]++
				g.ShortCircuit(g.B)
			} else if g.B.Value() == One {
				// O = A
				stats[g.Op]++
				g.ShortCircuit(g.A)
			}

		case circuit.OR:
			if g.A.Value() == One || g.B.Value() == One {
				g.O.SetValue(One)
				stats[g.Op]++
			} else if g.A.Value() == Zero && g.B.Value() == Zero {
				g.O.SetValue(Zero)
				stats[g.Op]++
			} else if g.A.Value() == Zero {
				// O = B
				stats[g.Op]++
				g.ShortCircuit(g.B)
			} else if g.B.Value() == Zero {
				// O = A
				stats[g.Op]++
				g.ShortCircuit(g.A)
			}

		case circuit.INV:
			if g.A.Value() == One {
				g.O.SetValue(Zero)
				stats[g.Op]++
			} else if g.A.Value() == Zero {
				g.O.SetValue(One)
				stats[g.Op]++
			}
		}

		if g.A.Value() == Zero {
			g.A.RemoveOutput(g)
			g.A = c.ZeroWire()
			g.A.AddOutput(g)
		} else if g.A.Value() == One {
			g.A.RemoveOutput(g)
			g.A = c.OneWire()
			g.A.AddOutput(g)
		}
		if g.B != nil {
			if g.B.Value() == Zero {
				g.B.RemoveOutput(g)
				g.B = c.ZeroWire()
				g.B.AddOutput(g)
			} else if g.B.Value() == One {
				g.B.RemoveOutput(g)
				g.B = c.OneWire()
				g.B.AddOutput(g)
			}
		}
	}

	elapsed := time.Since(start)

	if c.Params.Diagnostics && stats.Count() > 0 {
		fmt.Printf(" - ConstPropagate:      %12s: %d/%d (%.2f%%)\n",
			elapsed, stats.Count(), len(c.Gates),
			float64(stats.Count())/float64(len(c.Gates))*100)
	}
}

// ShortCircuitXORZero short circuits input to output where input is
// XOR'ed to zero.
func (c *Compiler) ShortCircuitXORZero() {
	var stats circuit.Stats

	start := time.Now()

	for _, g := range c.Gates {
		if g.Op != circuit.XOR {
			continue
		}
		if g.A.Value() == Zero && !g.B.IsInput() &&
			g.B.Input().O.NumOutputs() == 1 {

			g.B.Input().ResetOutput(g.O)

			// Disconnect gate's output wire.
			g.O = NewWire()

			stats[g.Op]++
		}
		if g.B.Value() == Zero && !g.A.IsInput() &&
			g.A.Input().O.NumOutputs() == 1 {

			g.A.Input().ResetOutput(g.O)

			// Disconnect gate's output wire.
			g.O = NewWire()

			stats[g.Op]++
		}
	}

	elapsed := time.Since(start)

	if c.Params.Diagnostics && stats.Count() > 0 {
		fmt.Printf(" - ShortCircuitXORZero: %12s: %d/%d (%.2f%%)\n",
			elapsed, stats.Count(), len(c.Gates),
			float64(stats.Count())/float64(len(c.Gates))*100)
	}
}

// Prune removes all gates whose output wires are unused.
func (c *Compiler) Prune() int {

	n := make([]*Gate, len(c.Gates))
	nPos := len(n)

	for i := len(c.Gates) - 1; i >= 0; i-- {
		g := c.Gates[i]
		if !g.Prune() {
			nPos--
			n[nPos] = g
		}
	}
	c.Gates = n[nPos:]

	return nPos
}

// Compile compiles the circuit.
func (c *Compiler) Compile() *circuit.Circuit {
	if len(c.pending) != 0 {
		panic("Compile: pending set")
	}
	c.pending = make([]*Gate, 0, len(c.Gates))
	if len(c.assigned) != 0 {
		panic("Compile: assigned set")
	}
	c.assigned = make([]*Gate, 0, len(c.Gates))
	if len(c.compiled) != 0 {
		panic("Compile: compiled set")
	}
	c.compiled = make([]circuit.Gate, 0, len(c.Gates))

	for _, w := range c.InputWires {
		w.Assign(c)
	}
	for len(c.pending) > 0 {
		gate := c.pending[0]
		c.pending = c.pending[1:]
		gate.Assign(c)
	}
	// Assign outputs.
	for _, w := range c.OutputWires {
		if w.Assigned() {
			if !c.OutputsAssigned {
				panic("Output already assigned")
			}
		} else {
			w.SetID(c.NextWireID())
		}
	}

	// Compile circuit.
	for _, gate := range c.assigned {
		gate.Compile(c)
	}

	var stats circuit.Stats
	for _, g := range c.compiled {
		stats[g.Op]++
	}

	result := &circuit.Circuit{
		NumGates: len(c.compiled),
		NumWires: int(c.nextWireID),
		Inputs:   c.Inputs,
		Outputs:  c.Outputs,
		Gates:    c.compiled,
		Stats:    stats,
	}

	return result
}
