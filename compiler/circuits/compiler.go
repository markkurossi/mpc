//
// Copyright (c) 2019-2020 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"
	"math"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
)

type Builtin func(cc *Compiler, a, b, r []*Wire) error

type Compiler struct {
	Params      *utils.Params
	Inputs      circuit.IO
	Outputs     circuit.IO
	InputWires  []*Wire
	OutputWires []*Wire
	Gates       []*Gate
	nextWireID  uint32
	pending     []*Gate
	assigned    []*Gate
	compiled    []circuit.Gate
	wiresX      map[string][]*Wire
	zeroWire    *Wire
	oneWire     *Wire
}

func NewIO(size int, name string) circuit.IO {
	return circuit.IO{
		circuit.IOArg{
			Name: name,
			Size: size,
		},
	}
}

func NewCompiler(params *utils.Params, inputs, outputs circuit.IO,
	inputWires, outputWires []*Wire) (*Compiler, error) {

	if inputs.Size() == 0 {
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

func (c *Compiler) ZeroWire() *Wire {
	if c.zeroWire == nil {
		c.zeroWire = NewWire()
		c.AddGate(NewBinary(circuit.XOR, c.InputWires[0], c.InputWires[0],
			c.zeroWire))
	}
	return c.zeroWire
}

func (c *Compiler) OneWire() *Wire {
	if c.oneWire == nil {
		c.oneWire = NewWire()
		c.AddGate(NewBinary(circuit.XNOR, c.InputWires[0], c.InputWires[0],
			c.oneWire))
	}
	return c.oneWire
}

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

func (c *Compiler) INV(i, o *Wire) {
	c.AddGate(NewBinary(circuit.XOR, i, c.OneWire(), o))
}

func (c *Compiler) ID(i, o *Wire) {
	c.AddGate(NewBinary(circuit.XOR, i, c.ZeroWire(), o))
}

func (c *Compiler) AddGate(gate *Gate) {
	c.Gates = append(c.Gates, gate)
}

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
			panic("Output already assigned")
		}
		w.ID = c.NextWireID()
	}

	// Compile circuit.
	for _, gate := range c.assigned {
		gate.Compile(c)
	}

	stats := make(map[circuit.Operation]int)
	for _, g := range c.compiled {
		count := stats[g.Op]
		count++
		stats[g.Op] = count
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
	UnassignedID uint32 = math.MaxUint32
)

type Wire struct {
	Output     bool
	ID         uint32
	NumOutputs uint32
	Input      *Gate
	Outputs    []*Gate
}

func (w *Wire) Assigned() bool {
	return w.ID != UnassignedID
}

func NewWire() *Wire {
	return &Wire{
		ID:      UnassignedID,
		Outputs: make([]*Gate, 0, 1),
	}
}

func MakeWires(bits int) []*Wire {
	result := make([]*Wire, bits)
	for i := 0; i < bits; i++ {
		result[i] = NewWire()
	}
	return result
}

func (w *Wire) String() string {
	return fmt.Sprintf("Wire{%p, Input:%v, Outputs:%d, Output=%v}",
		w, w.Input, w.NumOutputs, w.Output)
}

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

func (w *Wire) SetInput(gate *Gate) {
	if w.Input != nil {
		panic("Input gate already set")
	}
	w.Input = gate
}

func (w *Wire) AddOutput(gate *Gate) {
	w.Outputs = append(w.Outputs, gate)
	w.NumOutputs++
}

func (w *Wire) RemoveOutput(gate *Gate) {
	w.NumOutputs--
}
