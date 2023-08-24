//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/types"
)

type WireAllocator interface {
	// Allocated tests if the wires have been allocated for the value.
	Allocated(v Value) bool

	// NextWireID allocated and returns the next unassigned wire ID.
	NextWireID() uint32

	// Wires allocates unassigned wires for the argument value.
	Wires(v Value, bits types.Size) ([]*circuits.Wire, error)

	// AssignedWires allocates assigned wires for the argument value.
	AssignedWires(v Value, bits types.Size) ([]*circuits.Wire, error)

	// SetWires allocates wire IDs for the value's wires.
	SetWires(v Value, w []*circuits.Wire)

	// GCWires recycles the wires of the argument value. The wires
	// must have been previously allocated with Wires, AssignedWires,
	// or SetWires; the function panics if the wires have not been
	// allocated.
	GCWires(v Value)

	// Debug prints debugging information about the wire allocator.
	Debug()
}
