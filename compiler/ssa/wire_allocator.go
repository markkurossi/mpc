//
// Copyright (c) 2020-2023 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"sort"

	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/types"
)

// Wires allocates unassigned wires for the argument value.
func (prog *Program) Wires(v string, bits types.Size) (
	[]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	alloc, ok := prog.wires[v]
	if !ok {
		alloc = prog.allocWires(bits)
		prog.wires[v] = alloc
	}
	return alloc.Wires, nil
}

// AssignedWires allocates assigned wires for the argument value.
func (prog *Program) AssignedWires(v string, bits types.Size) (
	[]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	alloc, ok := prog.wires[v]
	if !ok {
		alloc = prog.allocWires(bits)
		prog.wires[v] = alloc

		// Assign wire IDs.
		if alloc.Base == circuits.UnassignedID {
			alloc.Base = prog.nextWireID
			for i := 0; i < int(bits); i++ {
				alloc.Wires[i].SetID(prog.nextWireID + uint32(i))
			}
			prog.nextWireID += uint32(bits)
		}
	}

	return alloc.Wires, nil
}

type wireAlloc struct {
	Base  uint32
	Wires []*circuits.Wire
}

func (prog *Program) allocWires(bits types.Size) *wireAlloc {
	result := &wireAlloc{
		Base: circuits.UnassignedID,
	}

	fl, ok := prog.freeWires[bits]
	if ok && len(fl) > 0 {
		result.Wires = fl[len(fl)-1]
		result.Base = result.Wires[0].ID()
		prog.freeWires[bits] = fl[:len(fl)-1]
		prog.flHit++
	} else {
		result.Wires = circuits.MakeWires(bits)
		prog.flMiss++
	}

	return result
}

func (prog *Program) recycleWires(alloc *wireAlloc) {
	if alloc.Base == circuits.UnassignedID {
		alloc.Base = alloc.Wires[0].ID()
	}
	// Clear wires and reassign their IDs.
	bits := types.Size(len(alloc.Wires))
	for i := 0; i < int(bits); i++ {
		alloc.Wires[i].Reset(alloc.Base + uint32(i))
	}

	fl := prog.freeWires[bits]
	fl = append(fl, alloc.Wires)
	prog.freeWires[bits] = fl
	if false {
		fmt.Printf("FL: %d: ", bits)
		for k, v := range prog.freeWires {
			fmt.Printf(" %d:%d", k, len(v))
		}
		fmt.Println()
	}
}

// SetWires allocates wire IDs for the value's wires.
func (prog *Program) SetWires(v string, w []*circuits.Wire) error {
	_, ok := prog.wires[v]
	if ok {
		return fmt.Errorf("wires already set for %v", v)
	}
	alloc := &wireAlloc{
		Wires: w,
	}
	if len(w) == 0 {
		alloc.Base = circuits.UnassignedID
	} else {
		alloc.Base = w[0].ID()
	}

	prog.wires[v] = alloc

	return nil
}

// StreamDebug prints debugging information about the circuit
// streaming.
func (prog *Program) StreamDebug() {
	total := float64(prog.flHit + prog.flMiss)
	fmt.Printf("Wire freelist: hit=%v (%.2f%%), miss=%v (%.2f%%)\n",
		prog.flHit, float64(prog.flHit)/total*100,
		prog.flMiss, float64(prog.flMiss)/total*100)

	var keys []types.Size
	for k := range prog.freeWires {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	for _, k := range keys {
		fmt.Printf(" %d:\t%d\n", k, len(prog.freeWires[types.Size(k)]))
	}
	fmt.Println()
}
