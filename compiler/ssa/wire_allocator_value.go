//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"math"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/types"
)

// WAllocValue implements WireAllocator using Value.HashCode to map
// values to wires.
type WAllocValue struct {
	calloc      *circuits.Allocator
	freeHdrs    []*allocByValue
	freeWires   map[types.Size][][]*circuits.Wire
	freeIDs     map[types.Size][][]circuit.Wire
	hash        [10240]*allocByValue
	nextWireID  circuit.Wire
	flHit       int
	flMiss      int
	lookupCount int
	lookupFound int
}

type allocByValue struct {
	next  *allocByValue
	key   Value
	base  circuit.Wire
	wires []*circuits.Wire
	ids   []circuit.Wire
}

func (alloc *allocByValue) String() string {
	return fmt.Sprintf("%v[%v]: base=%v, len(wires)=%v",
		alloc.key.String(), alloc.key.Type,
		alloc.base, len(alloc.wires))
}

// NewWAllocValue creates a new WAllocValue.
func NewWAllocValue(calloc *circuits.Allocator) WireAllocator {
	return &WAllocValue{
		calloc:    calloc,
		freeWires: make(map[types.Size][][]*circuits.Wire),
		freeIDs:   make(map[types.Size][][]circuit.Wire),
	}
}

func (walloc *WAllocValue) hashCode(v Value) int {
	return v.HashCode() % len(walloc.hash)
}

func (walloc *WAllocValue) newHeader(v Value) (ret *allocByValue) {
	if len(walloc.freeHdrs) == 0 {
		ret = new(allocByValue)
	} else {
		ret = walloc.freeHdrs[len(walloc.freeHdrs)-1]
		walloc.freeHdrs = walloc.freeHdrs[:len(walloc.freeHdrs)-1]
	}
	ret.key = v
	ret.base = circuits.UnassignedID
	return ret
}

func (walloc *WAllocValue) newWires(bits types.Size) (result []*circuits.Wire) {
	fl, ok := walloc.freeWires[bits]
	if ok && len(fl) > 0 {
		result = fl[len(fl)-1]
		walloc.freeWires[bits] = fl[:len(fl)-1]
		walloc.flHit++
	} else {
		result = walloc.calloc.Wires(bits)
		walloc.flMiss++
	}
	return result
}

func (walloc *WAllocValue) newIDs(bits types.Size) (result []circuit.Wire) {
	fl, ok := walloc.freeIDs[bits]
	if ok && len(fl) > 0 {
		result = fl[len(fl)-1]
		walloc.freeIDs[bits] = fl[:len(fl)-1]
		walloc.flHit++
	} else {
		result = make([]circuit.Wire, bits)
		for i := 0; i < int(bits); i++ {
			result[i] = circuits.UnassignedID
		}
		walloc.flMiss++
	}
	return result
}

// Allocated implements WireAllocator.Allocated.
func (walloc *WAllocValue) Allocated(v Value) bool {
	hash := walloc.hashCode(v)
	alloc := walloc.lookup(hash, v)
	return alloc != nil
}

// NextWireID implements WireAllocator.NextWireID.
func (walloc *WAllocValue) NextWireID() circuit.Wire {
	ret := walloc.nextWireID
	walloc.nextWireID++
	return ret
}

func (walloc *WAllocValue) lookup(hash int, v Value) *allocByValue {
	var count int
	for ptr := &walloc.hash[hash]; *ptr != nil; ptr = &(*ptr).next {
		count++
		if (*ptr).key.Equal(&v) {
			alloc := *ptr

			if count > 2 {
				// MRU in the hash bucket.
				*ptr = alloc.next
				alloc.next = walloc.hash[hash]
				walloc.hash[hash] = alloc
			}

			walloc.lookupCount++
			walloc.lookupFound += count
			return alloc
		}
	}
	return nil
}

func (walloc *WAllocValue) remove(hash int, v Value) *allocByValue {
	for ptr := &walloc.hash[hash]; *ptr != nil; ptr = &(*ptr).next {
		if (*ptr).key.Equal(&v) {
			ret := *ptr
			*ptr = (*ptr).next
			return ret
		}
	}
	return nil
}

func (walloc *WAllocValue) alloc(bits types.Size, v Value,
	wires, ids bool) *allocByValue {

	result := walloc.newHeader(v)

	if wires && ids {
		result.wires = walloc.newWires(bits)
		result.ids = walloc.newIDs(bits)
		result.base = result.wires[0].ID()

		for i := 0; i < int(bits); i++ {
			result.ids[i] = result.wires[i].ID()
		}
	} else if wires {
		result.wires = walloc.newWires(bits)
		result.base = result.wires[0].ID()
	} else {
		result.ids = walloc.newIDs(bits)
		result.base = result.ids[0]
	}
	return result
}

// Wires implements WireAllocator.Wires.
func (walloc *WAllocValue) Wires(v Value, bits types.Size) (
	[]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	hash := walloc.hashCode(v)
	alloc := walloc.lookup(hash, v)
	if alloc == nil {
		alloc = walloc.alloc(bits, v, true, false)
		alloc.next = walloc.hash[hash]
		walloc.hash[hash] = alloc
	}
	return alloc.wires, nil
}

// AssignedWires implements WireAllocator.AssignedWires.
func (walloc *WAllocValue) AssignedWires(v Value, bits types.Size) (
	[]circuit.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	hash := walloc.hashCode(v)
	alloc := walloc.lookup(hash, v)
	if alloc == nil {
		alloc = walloc.alloc(bits, v, false, true)
		alloc.next = walloc.hash[hash]
		walloc.hash[hash] = alloc

		// Assign wire IDs.
		if alloc.base == circuits.UnassignedID {
			alloc.base = walloc.nextWireID
			for i := 0; i < int(bits); i++ {
				alloc.ids[i] = walloc.nextWireID + circuit.Wire(i)
			}
			walloc.nextWireID += circuit.Wire(bits)
		}
	}
	if alloc.ids == nil {
		alloc.ids = walloc.newIDs(bits)
		for i := 0; i < int(bits); i++ {
			alloc.ids[i] = alloc.wires[i].ID()
		}
	}
	return alloc.ids, nil
}

func (walloc *WAllocValue) AssignedWiresAndIDs(v Value, bits types.Size) (
	[]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	hash := walloc.hashCode(v)
	alloc := walloc.lookup(hash, v)
	if alloc == nil {
		alloc = walloc.alloc(bits, v, true, true)
		alloc.next = walloc.hash[hash]
		walloc.hash[hash] = alloc

		// Assign wire IDs.
		if alloc.base == circuits.UnassignedID {
			alloc.base = walloc.nextWireID
			for i := 0; i < int(bits); i++ {
				alloc.wires[i].SetID(walloc.nextWireID + circuit.Wire(i))
			}
			walloc.nextWireID += circuit.Wire(bits)
		}
	}
	if alloc.ids == nil {
		alloc.ids = walloc.newIDs(bits)
		for i := 0; i < int(bits); i++ {
			alloc.ids[i] = alloc.wires[i].ID()
		}
	}
	return alloc.wires, nil
}

// SetWires implements WireAllocator.SetWires.
func (walloc *WAllocValue) SetWires(v Value, w []*circuits.Wire) {
	hash := walloc.hashCode(v)
	alloc := walloc.lookup(hash, v)
	if alloc != nil {
		panic(fmt.Sprintf("wires already set for %v", v))
	}
	alloc = &allocByValue{
		key:   v,
		wires: w,
		ids:   make([]circuit.Wire, len(w)),
	}
	if len(w) == 0 {
		alloc.base = circuits.UnassignedID
	} else {
		alloc.base = w[0].ID()
		for i := 0; i < len(w); i++ {
			alloc.ids[i] = w[i].ID()
		}
	}

	alloc.next = walloc.hash[hash]
	walloc.hash[hash] = alloc
}

// GCWires implements WireAllocator.GCWires.
func (walloc *WAllocValue) GCWires(v Value) {
	hash := walloc.hashCode(v)
	alloc := walloc.remove(hash, v)
	if alloc == nil {
		panic(fmt.Sprintf("GC: %s not known", v))
	}

	if alloc.wires != nil {
		if alloc.base == circuits.UnassignedID {
			alloc.base = alloc.wires[0].ID()
		}
		// Clear wires and reassign their IDs.
		for i := 0; i < len(alloc.wires); i++ {
			alloc.wires[i].Reset(alloc.base + circuit.Wire(i))
		}
		bits := types.Size(len(alloc.wires))
		walloc.freeWires[bits] = append(walloc.freeWires[bits], alloc.wires)
	}
	if alloc.ids != nil {
		if alloc.base == circuits.UnassignedID {
			alloc.base = alloc.ids[0]
		}
		// Clear IDs.
		for i := 0; i < len(alloc.ids); i++ {
			alloc.ids[i] = alloc.base + circuit.Wire(i)
		}
		bits := types.Size(len(alloc.ids))
		walloc.freeIDs[bits] = append(walloc.freeIDs[bits], alloc.ids)
	}

	alloc.next = nil
	alloc.base = circuits.UnassignedID
	alloc.wires = nil
	alloc.ids = nil
	walloc.freeHdrs = append(walloc.freeHdrs, alloc)
}

// Debug implements WireAllocator.Debug.
func (walloc *WAllocValue) Debug() {
	total := float64(walloc.flHit + walloc.flMiss)
	fmt.Printf("Wire freelist: hit=%v (%.2f%%), miss=%v (%.2f%%)\n",
		walloc.flHit, float64(walloc.flHit)/total*100,
		walloc.flMiss, float64(walloc.flMiss)/total*100)

	var sum, max int
	min := math.MaxInt

	var maxIndex int

	for i := 0; i < len(walloc.hash); i++ {
		var count int
		for alloc := walloc.hash[i]; alloc != nil; alloc = alloc.next {
			count++
		}
		sum += count
		if count < min {
			min = count
		}
		if count > max {
			max = count
			maxIndex = i
		}
	}
	fmt.Printf("Hash: min=%v, max=%v, avg=%.4f, lookup=%v (avg=%.4f)\n",
		min, max, float64(sum)/float64(len(walloc.hash)),
		walloc.lookupCount,
		float64(walloc.lookupFound)/float64(walloc.lookupCount))

	if false {
		fmt.Printf("Max bucket:\n")
		for alloc := walloc.hash[maxIndex]; alloc != nil; alloc = alloc.next {
			fmt.Printf(" %v: %v\n", alloc.key.String(), len(alloc.wires))
		}
	}
}
