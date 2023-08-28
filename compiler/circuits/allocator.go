//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"

	"github.com/markkurossi/mpc/types"
)

// Allocator implements circuit wire and gate allocation.
type Allocator struct {
	block    []Wire
	ofs      int
	numWire  uint64
	numWires uint64
}

// NewAllocator creates a new circuit allocator.
func NewAllocator() *Allocator {
	return new(Allocator)
}

// Wire allocates a new Wire.
func (alloc *Allocator) Wire() *Wire {
	alloc.numWire++
	w := new(Wire)
	w.Reset(UnassignedID)
	return w
}

// Wires allocate an array of Wires.
func (alloc *Allocator) Wires(bits types.Size) []*Wire {
	alloc.numWires += uint64(bits)
	result := make([]*Wire, bits)
	for i := 0; i < int(bits); i++ {
		if alloc.ofs == 0 {
			alloc.ofs = 8192
			alloc.block = make([]Wire, alloc.ofs)
		}
		alloc.ofs--
		w := &alloc.block[alloc.ofs]

		w.id = UnassignedID
		result[i] = w
	}
	return result
}

// Debug print debugging information about the circuit allocator.
func (alloc *Allocator) Debug() {
	fmt.Printf("circuits.Allocator: #wire=%v, #wires=%v\n",
		alloc.numWire, alloc.numWires)
}
