//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"sort"

	"github.com/markkurossi/mpc/ot"
)

type StreamWires struct {
	Wires    []ot.Wire
	TmpWires []ot.Wire
}

func NewStreamWires(normal, tmp int) *StreamWires {
	return &StreamWires{
		Wires:    make([]ot.Wire, normal),
		TmpWires: make([]ot.Wire, tmp),
	}
}

func (wires *StreamWires) Get(w Wire) ot.Wire {
	if w >= TmpWireID {
		return wires.TmpWires[w-TmpWireID]
	} else {
		return wires.Wires[w]
	}
}

func (wires *StreamWires) Set(w Wire, val ot.Wire) {
	if w >= TmpWireID {
		wires.TmpWires[w-TmpWireID] = val
	} else {
		wires.Wires[w] = val
	}
}

func (c *Circuit) GarbleStream(key []byte, r ot.Label) error {

	alg, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	var numWires int
	var numTmpWires int

	max := func(w Wire) {
		if w >= TmpWireID {
			l := w - TmpWireID + 1
			if int(l) > numTmpWires {
				numTmpWires = int(l)
			}
		} else if int(w) >= numWires {
			numWires = int(w) + 1
		}
	}

	for i := 0; i < len(c.Gates); i++ {
		g := &c.Gates[i]
		max(g.Input0)
		max(g.Input1)
		max(g.Output)
	}

	// Wire labels.
	wires := NewStreamWires(numWires, numTmpWires)

	// XXX
	if false {
		// Assing all input wires.
		for i := 0; i < c.Inputs.Size(); i++ {
			w, err := makeLabels(r)
			if err != nil {
				return err
			}
			wires.Set(Wire(i), w)
		}
	}

	// Garble gates.
	for i := 0; i < len(c.Gates); i++ {
		gate := &c.Gates[i]
		data, err := gate.GarbleStream(wires, alg, r, uint32(i))
		if err != nil {
			return err
		}
		// XXX stream data
		_ = data
	}
	return nil
}

func (g *Gate) GarbleStream(wires *StreamWires, enc cipher.Block,
	r ot.Label, id uint32) ([]ot.Label, error) {

	var a, b, c ot.Wire
	var err error

	// Inputs.
	switch g.Op {
	case XOR, XNOR, AND, OR:
		b = wires.Get(Wire(g.Input1.ID()))
		fallthrough

	case INV:
		a = wires.Get(Wire(g.Input0.ID()))

	default:
		return nil, fmt.Errorf("invalid gate type %s", g.Op)
	}

	// Output.
	switch g.Op {
	case XOR:
		l0 := a.L0
		l0.Xor(b.L0)

		l1 := l0
		l1.Xor(r)
		c = ot.Wire{
			L0: l0,
			L1: l1,
		}

	case XNOR:
		l0 := a.L0
		l0.Xor(b.L0)

		l1 := l0
		l1.Xor(r)
		c = ot.Wire{
			L0: l1,
			L1: l0,
		}

	default:
		c, err = makeLabels(r)
		if err != nil {
			return nil, err
		}
	}
	wires.Set(Wire(g.Output.ID()), c)

	var table [4]TableEntry
	var count int

	switch g.Op {
	case XOR, XNOR:
		// Free XOR.

	case AND:
		// a b c
		// -----
		// 0 0 0
		// 0 1 0
		// 1 0 0
		// 1 1 1
		table[0] = entry(enc, a.L0, b.L0, c.L0, id)
		table[1] = entry(enc, a.L0, b.L1, c.L0, id)
		table[2] = entry(enc, a.L1, b.L0, c.L0, id)
		table[3] = entry(enc, a.L1, b.L1, c.L1, id)
		count = 4

	case OR:
		// a b c
		// -----
		// 0 0 0
		// 0 1 1
		// 1 0 1
		// 1 1 1
		table[0] = entry(enc, a.L0, b.L0, c.L0, id)
		table[1] = entry(enc, a.L0, b.L1, c.L1, id)
		table[2] = entry(enc, a.L1, b.L0, c.L1, id)
		table[3] = entry(enc, a.L1, b.L1, c.L1, id)
		count = 4

	case INV:
		// a b c
		// -----
		// 0   1
		// 1   0
		table[0] = entryUnary(enc, a.L0, c.L1, id)
		table[1] = entryUnary(enc, a.L1, c.L0, id)
		count = 2

	default:
		return nil, fmt.Errorf("Invalid operand %s", g.Op)
	}

	sort.Sort(ByIndex(table[:count]))

	result := make([]ot.Label, count)
	for idx, entry := range table[:count] {
		result[idx] = entry.Data
	}

	return result, nil
}
