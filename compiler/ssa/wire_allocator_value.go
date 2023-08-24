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

type WAllocValue struct {
	freeWires  map[types.Size][][]*circuits.Wire
	wires      [10240]*allocByValue
	debug      map[string]*allocByValue
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

func NewWAllocValue() WireAllocator {
	return &WAllocValue{
		freeWires: make(map[types.Size][][]*circuits.Wire),
		debug:     make(map[string]*allocByValue),
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

func (walloc *WAllocValue) verify(v Value, expected *allocByValue) {
	dbg, ok := walloc.debug[v.String()]
	if expected == nil {
		if ok {
			panic(fmt.Sprintf("debug has key %v", v.String()))
		}
		if walloc.Allocated(v) {
			panic(fmt.Sprintf("hash has key %v", v.String()))
		}
		return
	}
	if !ok {
		panic(fmt.Sprintf("debug has not key %v", v.String()))
	}
	if !walloc.Allocated(v) {
		panic(fmt.Sprintf("hash has not key %v", v.String()))
	}
	if dbg != expected {
		fmt.Printf("debug %v != expected %v\n", dbg, expected)
		fmt.Printf(" - debug.key:    %v\n", dbg.key)
		fmt.Printf(" - expected.key: %v\n", expected.key)
		if dbg.key.Equal(&expected.key) {
			fmt.Printf(" - keys Equal\n")
		} else {
			fmt.Printf(" - keys not Equal\n")
			fmt.Printf(" - dbg: %v, expected: %v\n",
				dbg.key.HashCode()%len(walloc.wires),
				expected.key.HashCode()%len(walloc.wires))

		}
		panic("done")
	}
	if dbg.key.String() != v.String() {
		panic("dbg.String() mismatch")
	}

	hash := v.HashCode() % len(walloc.wires)
	for alloc := walloc.wires[hash]; alloc != nil; alloc = alloc.next {
		if alloc.key.Equal(&v) {
			if alloc != expected {
				panic("wires 1")
			}
			if alloc != dbg {
				panic("wires 2")
			}
			if alloc.key.String() != v.String() {
				panic("alloc.String() mismatch")
			}
			return
		}
	}
	panic("wires not found")
}

func (walloc *WAllocValue) verifyAdd(v Value, alloc *allocByValue) {
	walloc.debug[v.String()] = alloc
	walloc.verify(v, alloc)
}

func (walloc *WAllocValue) lookup(hash int, v Value) *allocByValue {
	const key = "g{1,0}struct1024"
	var keyMatch bool
	if v.String() == key {
		fmt.Printf("*** lookup %v: Const=%v, Bits=%v, Scope=%v, Version=%v, hash=%v\n",
			key, v.Const, v.Type.Bits, v.Scope, v.Version, hash)
		keyMatch = true
	}
	for a := walloc.wires[hash]; a != nil; a = a.next {
		if a.key.Equal(&v) {
			if keyMatch {
				fmt.Printf("  - found!\n")
			}
			return a
		}
	}
	if keyMatch {
		fmt.Printf("  - not found!\n")
	}
	return nil
}

func (walloc *WAllocValue) remove(hash int, v Value) *allocByValue {
	for ptr := &walloc.wires[hash]; *ptr != nil; ptr = &(*ptr).next {
		if (*ptr).key.Equal(&v) {
			ret := *ptr
			delete(walloc.debug, v.String())
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
		_, ok := walloc.debug[v.String()]
		if ok {
			panic(fmt.Sprintf("Wires: dbg has %v, TypeRef=%v",
				v.String(), v.TypeRef))
		}

		alloc = walloc.alloc(bits, v)
		alloc.next = walloc.wires[hash]
		walloc.wires[hash] = alloc
		walloc.verifyAdd(v, alloc)
	}
	walloc.verify(v, alloc)
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
		_, ok := walloc.debug[v.String()]
		if ok {
			panic(fmt.Sprintf("AssignedWires: dbg has %v: Const=%v, Bits=%v, Scope=%v, Version=%v",
				v.String(), v.Const, v.Type.Bits, v.Scope, v.Version))
		}
		alloc = walloc.alloc(bits, v)
		alloc.next = walloc.wires[hash]
		walloc.wires[hash] = alloc
		walloc.verifyAdd(v, alloc)

		// Assign wire IDs.
		if alloc.base == circuits.UnassignedID {
			alloc.base = walloc.nextWireID
			for i := 0; i < int(bits); i++ {
				alloc.wires[i].SetID(walloc.nextWireID + uint32(i))
			}
			walloc.nextWireID += uint32(bits)
		}
	}
	walloc.verify(v, alloc)
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
	walloc.verifyAdd(v, alloc)
	walloc.verify(v, alloc)
}

// GCWires implements WireAllocator.GCWires.
func (walloc *WAllocValue) GCWires(v Value) {
	hash := v.HashCode() % len(walloc.wires)
	alloc := walloc.remove(hash, v)
	if alloc == nil {
		panic(fmt.Sprintf("GC: %s not known", v))
	}
	walloc.verify(v, nil)

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
