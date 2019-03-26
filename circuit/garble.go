//
// garble.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
	"sort"

	"github.com/markkurossi/mpc/ot"
)

type Enc func(a, b, c *ot.Label, t uint32) []byte

type Dec func(a, b *ot.Label, t uint32, data []byte) ([]byte, error)

type TableEntry struct {
	Index int
	Data  []byte
}

type ByIndex []TableEntry

func (a ByIndex) Len() int {
	return len(a)
}

func (a ByIndex) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByIndex) Less(i, j int) bool {
	return a[i].Index < a[j].Index
}

func entry(enc Enc, a, b, c *ot.Label, tweak uint32) TableEntry {
	return TableEntry{
		Index: idx(a, b),
		Data:  enc(a, b, c, tweak),
	}
}

func idx(l0, l1 *ot.Label) int {
	if l0 == nil {
		if l1 == nil {
			return 0
		}
		if l1.S() {
			return 1
		} else {
			return 0
		}
	} else if l1 == nil {
		if l0.S() {
			return 1
		} else {
			return 0
		}
	} else {
		var ret int
		if l0.S() {
			ret |= 0x2
		}
		if l1.S() {
			ret |= 0x1
		}
		return ret
	}
}

func (g *Gate) Garble(wires ot.Inputs, enc Enc) ([][]byte, error) {
	var in []ot.Wire
	var out []ot.Wire

	for _, i := range g.Inputs {
		w, ok := wires[i.ID()]
		if !ok {
			return nil, fmt.Errorf("Unknown input wire %d", i)
		}
		in = append(in, w)
	}

	for _, o := range g.Outputs {
		w, ok := wires[o.ID()]
		if !ok {
			return nil, fmt.Errorf("Unknown output wire %d", o)
		}
		out = append(out, w)
	}

	var table []TableEntry

	switch g.Op {
	case XOR:
		// a b c
		// -----
		// 0 0 0
		// 0 1 1
		// 1 0 1
		// 1 1 0
		a := in[0]
		b := in[1]
		c := out[0]
		table = append(table, entry(enc, a.Label0, b.Label0, c.Label0, g.ID))
		table = append(table, entry(enc, a.Label0, b.Label1, c.Label1, g.ID))
		table = append(table, entry(enc, a.Label1, b.Label0, c.Label1, g.ID))
		table = append(table, entry(enc, a.Label1, b.Label1, c.Label0, g.ID))

	case AND:
		// a b c
		// -----
		// 0 0 0
		// 0 1 0
		// 1 0 0
		// 1 1 1
		a := in[0]
		b := in[1]
		c := out[0]
		table = append(table, entry(enc, a.Label0, b.Label0, c.Label0, g.ID))
		table = append(table, entry(enc, a.Label0, b.Label1, c.Label0, g.ID))
		table = append(table, entry(enc, a.Label1, b.Label0, c.Label0, g.ID))
		table = append(table, entry(enc, a.Label1, b.Label1, c.Label1, g.ID))

	case OR:
		// a b c
		// -----
		// 0 0 0
		// 0 1 1
		// 1 0 1
		// 1 1 1
		a := in[0]
		b := in[1]
		c := out[0]
		table = append(table, entry(enc, a.Label0, b.Label0, c.Label0, g.ID))
		table = append(table, entry(enc, a.Label0, b.Label1, c.Label1, g.ID))
		table = append(table, entry(enc, a.Label1, b.Label0, c.Label1, g.ID))
		table = append(table, entry(enc, a.Label1, b.Label1, c.Label1, g.ID))

	case INV:
		// a b c
		// -----
		// 0   1
		// 1   0
		a := in[0]
		c := out[0]
		table = append(table, entry(enc, a.Label0, nil, c.Label1, g.ID))
		table = append(table, entry(enc, a.Label1, nil, c.Label0, g.ID))

	default:
		return nil, fmt.Errorf("Invalid operand %s", g.Op)
	}

	sort.Sort(ByIndex(table))

	var result [][]byte
	for _, entry := range table {
		result = append(result, entry.Data)
	}

	return result, nil
}

func (g *Gate) Eval(wires map[Wire]*ot.Label, dec Dec, garbled [][]byte) (
	[]byte, error) {

	var a *ot.Label
	var aOK bool
	var b *ot.Label
	var bOK bool

	switch g.Op {
	case XOR, AND, OR:
		a, aOK = wires[g.Inputs[0]]
		b, bOK = wires[g.Inputs[1]]

	case INV:
		a, aOK = wires[g.Inputs[0]]
		b = nil
		bOK = true

	default:
		return nil, fmt.Errorf("Invalid operation %s", g.Op)
	}

	if !aOK {
		return nil, fmt.Errorf("No input for wire a found")
	}
	if !bOK {
		return nil, fmt.Errorf("No input for wire b found")
	}

	return dec(a, b, g.ID, garbled[idx(a, b)])
}
