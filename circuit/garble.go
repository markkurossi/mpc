//
// garble.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"sort"

	"github.com/markkurossi/mpc/ot"
)

var (
	verbose = false
)

type Enc func(a, b, c ot.Label, t uint32) ot.Label

type Dec func(a, b ot.Label, t uint32, data ot.Label) (ot.Label, error)

type TableEntry struct {
	Index int
	Data  ot.Label
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

func entry(enc Enc, a, b, c ot.Label, tweak uint32) TableEntry {
	return TableEntry{
		Index: idx(a, b),
		Data:  enc(a, b, c, tweak),
	}
}

func entryUnary(enc Enc, a, c ot.Label, tweak uint32) TableEntry {
	return TableEntry{
		Index: idxUnary(a),
		Data:  enc(a, ot.Label{}, c, tweak),
	}
}

func idxUnary(l0 ot.Label) int {
	if l0.S() {
		return 1
	} else {
		return 0
	}
}

func idx(l0, l1 ot.Label) int {
	var ret int

	if l0.S() {
		ret |= 0x2
	}
	if l1.S() {
		ret |= 0x1
	}

	return ret
}

func encrypt(alg cipher.Block, a, b, c ot.Label, t uint32) ot.Label {
	k := makeK(a, b, t)
	kData := k.Data()

	var crypted ot.LabelData
	alg.Encrypt(crypted[:], kData[:])

	pi := ot.LabelFromData(crypted)
	pi.Xor(k)
	pi.Xor(c)

	return pi
}

func decrypt(alg cipher.Block, a, b ot.Label, t uint32, encrypted ot.Label) (
	ot.Label, error) {

	k := makeK(a, b, t)
	kData := k.Data()

	var crypted ot.LabelData
	alg.Encrypt(crypted[:], kData[:])

	c := encrypted
	c.Xor(ot.LabelFromData(crypted))
	c.Xor(k)

	return c, nil
}

func makeK(a, b ot.Label, t uint32) ot.Label {
	a.Mul2()
	a.Xor(ot.NewTweak(t))

	return a
}

func makeLabels(r ot.Label) (ot.Wire, error) {
	l0, err := ot.NewLabel(rand.Reader)
	if err != nil {
		return ot.Wire{}, err
	}
	l1 := l0
	l1.Xor(r)

	return ot.Wire{
		L0: l0,
		L1: l1,
	}, nil
}

type Garbled struct {
	R     ot.Label
	Wires []ot.Wire
	Gates [][]ot.Label
}

func (g *Garbled) Lambda(wire Wire) uint {
	if g.Wires[int(wire)].L0.S() {
		return 1
	}
	return 0
}

func (g *Garbled) SetLambda(wire Wire, val uint) {
	w := g.Wires[int(wire)]
	if val == 0 {
		w.L0.SetS(false)
	} else {
		w.L0.SetS(true)
	}
	g.Wires[int(wire)] = w
}

func (c *Circuit) Garble(key []byte) (*Garbled, error) {
	// Create R.
	r, err := ot.NewLabel(rand.Reader)
	if err != nil {
		return nil, err
	}
	r.SetS(true)

	garbled := make([][]ot.Label, c.NumGates)

	alg, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	enc := func(a, b, c ot.Label, t uint32) ot.Label {
		return encrypt(alg, a, b, c, t)
	}

	// Wire labels.
	wires := make([]ot.Wire, c.NumWires)

	// Assing all input wires.
	for i := 0; i < c.N1.Size()+c.N2.Size(); i++ {
		w, err := makeLabels(r)
		if err != nil {
			return nil, err
		}
		wires[i] = w
	}

	// Garble gates.
	for i := 0; i < len(c.Gates); i++ {
		gate := &c.Gates[i]
		data, err := gate.Garble(wires, enc, r, uint32(i))
		if err != nil {
			return nil, err
		}
		garbled[i] = data
	}

	return &Garbled{
		R:     r,
		Wires: wires,
		Gates: garbled,
	}, nil
}

func (g *Gate) Garble(wires []ot.Wire, enc Enc, r ot.Label, id uint32) (
	[]ot.Label, error) {

	var a, b, c ot.Wire
	var err error

	// Inputs.
	switch g.Op {
	case XOR, XNOR, AND, OR:
		b = wires[g.Input1.ID()]
		fallthrough

	case INV:
		a = wires[g.Input0.ID()]

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
	wires[g.Output.ID()] = c

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
