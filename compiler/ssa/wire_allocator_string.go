//
// Copyright (c) 2020-2023 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"sort"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/types"
)

// WAllocString implements WireAllocator using Value.String to map
// values to wires.
type WAllocString struct {
	calloc     *circuits.Allocator
	freeWires  map[types.Size][][]*circuits.Wire
	wires      map[string]*wireAlloc
	nextWireID circuit.Wire
	flHit      int
	flMiss     int
}

// NewWAllocString creates a new WAllocString.
func NewWAllocString(calloc *circuits.Allocator) WireAllocator {
	return &WAllocString{
		calloc:    calloc,
		wires:     make(map[string]*wireAlloc),
		freeWires: make(map[types.Size][][]*circuits.Wire),
	}
}

var (
	vConst   int
	vTypeRef int
	vPtr     int
	vDefault int
	hash     [1024]int
)

func addValueStats(v Value) {
	if v.Const {
		vConst++
	} else if v.TypeRef {
		vTypeRef++
	} else if v.Type.Type == types.TPtr {
		vPtr++
	} else {
		vDefault++
	}

	hash[v.HashCode()%len(hash)]++
}

// Allocated implements WireAllocator.Allocated.
func (walloc *WAllocString) Allocated(v Value) bool {
	key := v.String()
	_, ok := walloc.wires[key]
	return ok
}

// NextWireID implements WireAllocator.NextWireID.
func (walloc *WAllocString) NextWireID() circuit.Wire {
	ret := walloc.nextWireID
	walloc.nextWireID++
	return ret
}

// Wires implements WireAllocator.Wires.
func (walloc *WAllocString) Wires(v Value, bits types.Size) (
	[]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	addValueStats(v)
	key := v.String()
	alloc, ok := walloc.wires[key]
	if !ok {
		alloc = walloc.allocWires(bits)
		walloc.wires[key] = alloc
	}
	return alloc.Wires, nil
}

// AssignedWires implements WireAllocator.AssignedWires.
func (walloc *WAllocString) AssignedWires(v Value, bits types.Size) (
	[]circuit.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	addValueStats(v)
	key := v.String()
	alloc, ok := walloc.wires[key]
	if !ok {
		alloc = walloc.allocWires(bits)
		walloc.wires[key] = alloc

		// Assign wire IDs.
		if alloc.Base == circuits.UnassignedID {
			alloc.Base = walloc.nextWireID
			for i := 0; i < int(bits); i++ {
				alloc.Wires[i].SetID(walloc.nextWireID + circuit.Wire(i))
			}
			walloc.nextWireID += circuit.Wire(bits)
		}
	}

	return alloc.IDs, nil
}

func (walloc *WAllocString) AssignedWiresAndIDs(v Value, bits types.Size) (
	[]*circuits.Wire, error) {
	return nil, fmt.Errorf("not implemented")
}

type wireAlloc struct {
	Base  circuit.Wire
	Wires []*circuits.Wire
	IDs   []circuit.Wire
}

func (walloc *WAllocString) allocWires(bits types.Size) *wireAlloc {
	result := &wireAlloc{
		Base: circuits.UnassignedID,
	}

	fl, ok := walloc.freeWires[bits]
	if ok && len(fl) > 0 {
		result.Wires = fl[len(fl)-1]
		result.Base = result.Wires[0].ID()
		walloc.freeWires[bits] = fl[:len(fl)-1]
		walloc.flHit++
	} else {
		result.Wires = walloc.calloc.Wires(bits)
		walloc.flMiss++
	}

	return result
}

// SetWires implements WireAllocator.SetWires.
func (walloc *WAllocString) SetWires(v Value, w []*circuits.Wire) {
	addValueStats(v)
	key := v.String()
	_, ok := walloc.wires[key]
	if ok {
		panic(fmt.Sprintf("wires already set for %v", key))
	}
	alloc := &wireAlloc{
		Wires: w,
	}
	if len(w) == 0 {
		alloc.Base = circuits.UnassignedID
	} else {
		alloc.Base = w[0].ID()
	}

	walloc.wires[key] = alloc
}

// GCWires implements WireAllocator.GCWires.
func (walloc *WAllocString) GCWires(v Value) {
	key := v.String()
	alloc, ok := walloc.wires[key]
	if !ok {
		panic(fmt.Sprintf("GC: %s not known", key))
	}
	delete(walloc.wires, key)

	if alloc.Base == circuits.UnassignedID {
		alloc.Base = alloc.Wires[0].ID()
	}
	// Clear wires and reassign their IDs.
	bits := types.Size(len(alloc.Wires))
	for i := 0; i < int(bits); i++ {
		alloc.Wires[i].Reset(alloc.Base + circuit.Wire(i))
	}

	fl := walloc.freeWires[bits]
	fl = append(fl, alloc.Wires)
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
func (walloc *WAllocString) Debug() {
	total := float64(walloc.flHit + walloc.flMiss)
	fmt.Printf("Wire freelist: hit=%v (%.2f%%), miss=%v (%.2f%%)\n",
		walloc.flHit, float64(walloc.flHit)/total*100,
		walloc.flMiss, float64(walloc.flMiss)/total*100)

	var keys []types.Size
	for k := range walloc.freeWires {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	for _, k := range keys {
		fmt.Printf(" %d:\t%d\n", k, len(walloc.freeWires[types.Size(k)]))
	}
	fmt.Println()

	fmt.Println("Value Stats:")

	sum := float64(vConst + vTypeRef + vPtr + vDefault)

	fmt.Printf(" - vConst:\t%v\t%f%%\n", vConst, float64(vConst)/sum*100)
	fmt.Printf(" - vTypeRef:\t%v\t%f%%\n", vTypeRef, float64(vTypeRef)/sum*100)
	fmt.Printf(" - vPtr:\t%v\t%f%%\n", vPtr, float64(vPtr)/sum*100)
	fmt.Printf(" - vDefault:\t%v\t%f%%\n", vDefault, float64(vDefault)/sum*100)

	if false {
		var zeroes int
		for idx, count := range hash {
			if count == 0 {
				zeroes++
			} else {
				fmt.Printf("%v:\t%v\n", idx, count)
			}
		}
		fmt.Printf("%v zero buckets\n", zeroes)
	}
}
