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

func idx(l0, l1 ot.Label) int {
	if l1.Undefined() {
		if l0.S() {
			return 1
		} else {
			return 0
		}
	}

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

	if !b.Undefined() {
		b.Mul4()
		a.Xor(b)
	}
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
	Wires ot.Inputs
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

	// Assign labels to wires.
	wires := make(ot.Inputs)

	for i := 0; i < len(c.Gates); i++ {
		gate := &c.Gates[i]
		data, err := gate.Garble(wires, enc, r, uint32(i))
		if err != nil {
			return nil, err
		}
		garbled[i] = data
	}

	// Assign unset wires. Wire can be unset if one of the inputs is
	// not used in the computation
	for i := 0; i < c.NumWires; i++ {
		_, ok := wires[i]
		if !ok {
			w, err := makeLabels(r)
			if err != nil {
				return nil, err
			}
			wires[i] = w
		}
	}

	return &Garbled{
		R:     r,
		Wires: wires,
		Gates: garbled,
	}, nil
}

func (g *Gate) Garble(wires ot.Inputs, enc Enc, r ot.Label, id uint32) (
	[]ot.Label, error) {

	var in []ot.Wire
	var out []ot.Wire
	var err error

	for _, i := range g.Inputs() {
		w, ok := wires[i.ID()]
		if !ok {
			w, err = makeLabels(r)
			if err != nil {
				return nil, err
			}
			wires[i.ID()] = w
		}
		in = append(in, w)
	}

	// Output
	w, ok := wires[g.Output.ID()]
	if ok {
		return nil, fmt.Errorf("gate output already set %d", g.Output)
	}
	switch g.Op {
	case XOR:
		l0 := in[0].L0
		l0.Xor(in[1].L0)

		l1 := l0
		l1.Xor(r)
		w = ot.Wire{
			L0: l0,
			L1: l1,
		}

	case XNOR:
		l0 := in[0].L0
		l0.Xor(in[1].L0)

		l1 := l0
		l1.Xor(r)
		w = ot.Wire{
			L0: l1,
			L1: l0,
		}

	default:
		w, err = makeLabels(r)
		if err != nil {
			return nil, err
		}
	}
	wires[g.Output.ID()] = w
	out = append(out, w)

	var table []TableEntry

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
		a := in[0]
		b := in[1]
		c := out[0]
		table = append(table, entry(enc, a.L0, b.L0, c.L0, id))
		table = append(table, entry(enc, a.L0, b.L1, c.L0, id))
		table = append(table, entry(enc, a.L1, b.L0, c.L0, id))
		table = append(table, entry(enc, a.L1, b.L1, c.L1, id))

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
		table = append(table, entry(enc, a.L0, b.L0, c.L0, id))
		table = append(table, entry(enc, a.L0, b.L1, c.L1, id))
		table = append(table, entry(enc, a.L1, b.L0, c.L1, id))
		table = append(table, entry(enc, a.L1, b.L1, c.L1, id))

	case INV:
		// a b c
		// -----
		// 0   1
		// 1   0
		a := in[0]
		c := out[0]
		table = append(table, entry(enc, a.L0, ot.Label{}, c.L1, id))
		table = append(table, entry(enc, a.L1, ot.Label{}, c.L0, id))

	default:
		return nil, fmt.Errorf("Invalid operand %s", g.Op)
	}

	sort.Sort(ByIndex(table))

	var result []ot.Label
	for _, entry := range table {
		result = append(result, entry.Data)
	}

	return result, nil
}
