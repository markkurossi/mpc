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

// Wires allocates unassigned wires for the argument value.
func (prog *Program) Wires(v Value, bits types.Size) (
	[]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	addValueStats(v)
	key := v.String()
	alloc, ok := prog.wires[key]
	if !ok {
		alloc = prog.allocWires(bits)
		prog.wires[key] = alloc
	}
	return alloc.Wires, nil
}

// AssignedWires allocates assigned wires for the argument value.
func (prog *Program) AssignedWires(v Value, bits types.Size) (
	[]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	addValueStats(v)
	key := v.String()
	alloc, ok := prog.wires[key]
	if !ok {
		alloc = prog.allocWires(bits)
		prog.wires[key] = alloc

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

// GCWires recycles the wires of the argument value. The wires must
// have been previously allocated with Wires, AssignedWires, or
// SetWires; the function panics if the wires have not been allocated.
func (prog *Program) GCWires(v Value) {
	key := v.String()
	alloc, ok := prog.wires[key]
	if !ok {
		panic(fmt.Sprintf("GC: %s not known", key))
	}
	delete(prog.wires, key)

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
func (prog *Program) SetWires(v Value, w []*circuits.Wire) error {
	addValueStats(v)
	key := v.String()
	_, ok := prog.wires[key]
	if ok {
		return fmt.Errorf("wires already set for %v", key)
	}
	alloc := &wireAlloc{
		Wires: w,
	}
	if len(w) == 0 {
		alloc.Base = circuits.UnassignedID
	} else {
		alloc.Base = w[0].ID()
	}

	prog.wires[key] = alloc

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
