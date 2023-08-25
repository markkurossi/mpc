//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"

	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/types"
)

// WAllocValue implements WireAllocator using Value.HashCode to map
// values to wires.
type WAllocValue struct {
	freeWires  map[types.Size][][]*circuits.Wire
	wires      [10240]*allocByValue
	nextWireID uint32
	flHit      int
	flMiss     int
}

type allocByValue struct {
	next  *allocByValue
	key   Value
	base  uint32
	wires []*circuits.Wire
}

func (alloc *allocByValue) String() string {
	return fmt.Sprintf("%v[%v]: base=%v, len(wires)=%v",
		alloc.key.String(), alloc.key.Type,
		alloc.base, len(alloc.wires))
}

// NewWAllocValue creates a new WAllocValue.
func NewWAllocValue() WireAllocator {
	return &WAllocValue{
		freeWires: make(map[types.Size][][]*circuits.Wire),
	}
}

// Allocated implements WireAllocator.Allocated.
func (walloc *WAllocValue) Allocated(v Value) bool {
	hash := v.HashCode() % len(walloc.wires)
	alloc := walloc.lookup(hash, v)
	return alloc != nil
}

// NextWireID implements WireAllocator.NextWireID.
func (walloc *WAllocValue) NextWireID() uint32 {
	ret := walloc.nextWireID
	walloc.nextWireID++
	return ret
}

func (walloc *WAllocValue) lookup(hash int, v Value) *allocByValue {
	for a := walloc.wires[hash]; a != nil; a = a.next {
		if a.key.Equal(&v) {
			return a
		}
	}
	return nil
}

func (walloc *WAllocValue) remove(hash int, v Value) *allocByValue {
	for ptr := &walloc.wires[hash]; *ptr != nil; ptr = &(*ptr).next {
		if (*ptr).key.Equal(&v) {
			ret := *ptr
			*ptr = (*ptr).next
			return ret
		}
	}
	return nil
}

func (walloc *WAllocValue) alloc(bits types.Size, v Value) *allocByValue {
	result := &allocByValue{
		key:  v,
		base: circuits.UnassignedID,
	}

	fl, ok := walloc.freeWires[bits]
	if ok && len(fl) > 0 {
		result.wires = fl[len(fl)-1]
		result.base = result.wires[0].ID()
		walloc.freeWires[bits] = fl[:len(fl)-1]
		walloc.flHit++
	} else {
		result.wires = circuits.MakeWires(bits)
		walloc.flMiss++
	}

	return result
}

// Wires implements WireAllocator.Wires.
func (walloc *WAllocValue) Wires(v Value, bits types.Size) (
	[]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	hash := v.HashCode() % len(walloc.wires)
	alloc := walloc.lookup(hash, v)
	if alloc == nil {
		alloc = walloc.alloc(bits, v)
		alloc.next = walloc.wires[hash]
		walloc.wires[hash] = alloc
	}
	return alloc.wires, nil
}

// AssignedWires implements WireAllocator.AssignedWires.
func (walloc *WAllocValue) AssignedWires(v Value, bits types.Size) (
	[]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	hash := v.HashCode() % len(walloc.wires)
	alloc := walloc.lookup(hash, v)
	if alloc == nil {
		alloc = walloc.alloc(bits, v)
		alloc.next = walloc.wires[hash]
		walloc.wires[hash] = alloc

		// Assign wire IDs.
		if alloc.base == circuits.UnassignedID {
			alloc.base = walloc.nextWireID
			for i := 0; i < int(bits); i++ {
				alloc.wires[i].SetID(walloc.nextWireID + uint32(i))
			}
			walloc.nextWireID += uint32(bits)
		}
	}
	return alloc.wires, nil
}

// SetWires implements WireAllocator.SetWires.
func (walloc *WAllocValue) SetWires(v Value, w []*circuits.Wire) {
	hash := v.HashCode() % len(walloc.wires)
	alloc := walloc.lookup(hash, v)
	if alloc != nil {
		panic(fmt.Sprintf("wires already set for %v", v))
	}
	alloc = &allocByValue{
		key:   v,
		wires: w,
	}
	if len(w) == 0 {
		alloc.base = circuits.UnassignedID
	} else {
		alloc.base = w[0].ID()
	}

	alloc.next = walloc.wires[hash]
	walloc.wires[hash] = alloc
}

// GCWires implements WireAllocator.GCWires.
func (walloc *WAllocValue) GCWires(v Value) {
	hash := v.HashCode() % len(walloc.wires)
	alloc := walloc.remove(hash, v)
	if alloc == nil {
		panic(fmt.Sprintf("GC: %s not known", v))
	}

	if alloc.base == circuits.UnassignedID {
		alloc.base = alloc.wires[0].ID()
	}
	// Clear wires and reassign their IDs.
	bits := types.Size(len(alloc.wires))
	for i := 0; i < int(bits); i++ {
		alloc.wires[i].Reset(alloc.base + uint32(i))
	}

	fl := walloc.freeWires[bits]
	fl = append(fl, alloc.wires)
	walloc.freeWires[bits] = fl
	if false {
		fmt.Printf("FL: %d: ", bits)
		for k, v := range walloc.freeWires {
			fmt.Printf(" %d:%d", k, len(v))
		}
		fmt.Println()
	}
}

// Debug implements WireAllocator.Debug.
func (walloc *WAllocValue) Debug() {
}
