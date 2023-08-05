//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"
	"math"

	"github.com/markkurossi/mpc/types"
)

const (
	// UnassignedID identifies an unassigned wire ID.
	UnassignedID uint32 = math.MaxUint32
	outputMask          = 0b10000000000000000000000000000000
	valueMask           = 0b01100000000000000000000000000000
	numMask             = 0b00011111111111111111111111111111
	valueShift          = 29
)

// Wire implements a wire connecting binary gates.
type Wire struct {
	ovnum   uint32
	id      uint32
	input   *Gate
	outputs []*Gate
}

// WireValue defines wire values.
type WireValue uint8

// Possible wire values.
const (
	Unknown WireValue = iota
	Zero
	One
)

func (v WireValue) String() string {
	switch v {
	case Zero:
		return "0"
	case One:
		return "1"
	default:
		return "?"
	}
}

// NewWire creates an unassigned wire.
func NewWire() *Wire {
	w := new(Wire)
	w.Reset(UnassignedID)
	return w
}

// MakeWires creates bits number of wires.
func MakeWires(bits types.Size) []*Wire {
	result := make([]*Wire, bits)
	for i := 0; i < int(bits); i++ {
		result[i] = NewWire()
	}
	return result
}

// Reset resets the wire with the new ID.
func (w *Wire) Reset(id uint32) {
	w.SetOutput(false)
	w.SetValue(Unknown)
	w.SetID(id)
	w.input = nil
	w.DisconnectOutputs()
}

// ID returns the wire ID.
func (w *Wire) ID() uint32 {
	return w.id
}

// SetID sets the wire ID.
func (w *Wire) SetID(id uint32) {
	w.id = id
}

// Assigned tests if the wire is assigned with an unique ID.
func (w *Wire) Assigned() bool {
	return w.id != UnassignedID
}

// Output tests if the wire is an output wire.
func (w *Wire) Output() bool {
	return w.ovnum&outputMask != 0
}

// SetOutput sets the wire output flag.
func (w *Wire) SetOutput(output bool) {
	if output {
		w.ovnum |= outputMask
	} else {
		w.ovnum &^= outputMask
	}
}

// Value returns the wire value.
func (w *Wire) Value() WireValue {
	return WireValue((w.ovnum & valueMask) >> valueShift)
}

// SetValue sets the wire value.
func (w *Wire) SetValue(value WireValue) {
	w.ovnum &^= valueMask
	w.ovnum |= (uint32(value) << valueShift) & valueMask
}

// NumOutputs returns the number of output gates assigned to the wire.
func (w *Wire) NumOutputs() uint32 {
	return w.ovnum & numMask
}

// SetNumOutputs sets the number of output gates assigned to the wire.
func (w *Wire) SetNumOutputs(num uint32) {
	if num > numMask {
		panic("too big circuit, wire outputs overflow")
	}
	w.ovnum &^= numMask
	w.ovnum |= num
}

// DisconnectOutputs disconnects wire from its output gates.
func (w *Wire) DisconnectOutputs() {
	w.SetNumOutputs(0)
	w.outputs = w.outputs[0:0]
}

func (w *Wire) String() string {
	return fmt.Sprintf("Wire{%x, Input:%s, Value:%s, Outputs:%v, Output=%v}",
		w.ID(), w.input, w.Value(), w.outputs, w.Output())
}

// Assign assings wire ID.
func (w *Wire) Assign(c *Compiler) {
	if w.Output() {
		return
	}
	if !w.Assigned() {
		w.id = c.NextWireID()
	}
	w.ForEachOutput(func(gate *Gate) {
		gate.Visit(c)
	})
}

// Input returns the wire's input gate.
func (w *Wire) Input() *Gate {
	return w.input
}

// SetInput sets the wire's input gate.
func (w *Wire) SetInput(gate *Gate) {
	if w.input != nil {
		panic("Input gate already set")
	}
	w.input = gate
}

// IsInput tests if the wire is an input wire.
func (w *Wire) IsInput() bool {
	return w.input == nil
}

// ForEachOutput calls the argument function for each output gate of
// the wire.
func (w *Wire) ForEachOutput(f func(gate *Gate)) {
	for _, gate := range w.outputs {
		f(gate)
	}
}

// AddOutput adds gate to the wire's output gates.
func (w *Wire) AddOutput(gate *Gate) {
	w.outputs = append(w.outputs, gate)
	w.SetNumOutputs(w.NumOutputs() + 1)
}

// RemoveOutput removes gate from the wire's output gates.
func (w *Wire) RemoveOutput(gate *Gate) {
	w.SetNumOutputs(w.NumOutputs() - 1)
}
