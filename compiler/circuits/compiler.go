//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"
	"math"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/types"
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

// ZeroWire returns a wire holding value 0.
func (c *Compiler) ZeroWire() *Wire {
	if c.zeroWire == nil {
		c.zeroWire = NewWire()
		c.AddGate(NewBinary(circuit.XOR, c.InputWires[0], c.InputWires[0],
			c.zeroWire))
	}
	return c.zeroWire
}

// OneWire returns a wire holding value 1.
func (c *Compiler) OneWire() *Wire {
	if c.oneWire == nil {
		c.oneWire = NewWire()
		c.AddGate(NewBinary(circuit.XNOR, c.InputWires[0], c.InputWires[0],
			c.oneWire))
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
			w.ID = c.NextWireID()
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

const (
	// UnassignedID identifies an unassigned wire ID.
	UnassignedID uint32 = math.MaxUint32
)

// Wire implements a wire connecting binary gates.
type Wire struct {
	Output     bool
	ID         uint32
	NumOutputs uint32
	Input      *Gate
	Outputs    []*Gate
}

// Assigned tests if the wire is assigned with an unique ID.
func (w *Wire) Assigned() bool {
	return w.ID != UnassignedID
}

// NewWire creates an unassigned wire.
func NewWire() *Wire {
	return &Wire{
		ID:      UnassignedID,
		Outputs: make([]*Gate, 0, 1),
	}
}

// MakeWires creates bits number of wires.
func MakeWires(bits types.Size) []*Wire {
	result := make([]*Wire, bits)
	for i := 0; i < int(bits); i++ {
		result[i] = NewWire()
	}
	return result
}

func (w *Wire) String() string {
	return fmt.Sprintf("Wire{%x, Input:%v, Outputs:%d, Output=%v}",
		w.ID, w.Input, w.NumOutputs, w.Output)
}

// Assign assings wire ID.
func (w *Wire) Assign(c *Compiler) {
	if w.Output {
		return
	}
	if !w.Assigned() {
		w.ID = c.NextWireID()
	}
	for _, output := range w.Outputs {
		output.Visit(c)
	}
}

// SetInput sets the wire's input gate.
func (w *Wire) SetInput(gate *Gate) {
	if w.Input != nil {
		panic("Input gate already set")
	}
	w.Input = gate
}

// AddOutput adds gate to the wire's output gates.
func (w *Wire) AddOutput(gate *Gate) {
	w.Outputs = append(w.Outputs, gate)
	w.NumOutputs++
}

// RemoveOutput removes gate from the wire's output gates.
func (w *Wire) RemoveOutput(gate *Gate) {
	w.NumOutputs--
}
