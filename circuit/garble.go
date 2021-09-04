//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"github.com/markkurossi/mpc/ot"
)

var (
	verbose = false
)

func idxUnary(l0 ot.Label) int {
	if l0.S() {
		return 1
	}
	return 0
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

func encrypt(alg cipher.Block, a, b, c ot.Label, t uint32,
	data *ot.LabelData) ot.Label {

	k := makeK(a, b, t)

	k.GetData(data)

	alg.Encrypt(data[:], data[:])

	var pi ot.Label
	pi.SetData(data)

	pi.Xor(k)
	pi.Xor(c)

	return pi
}

func decrypt(alg cipher.Block, a, b ot.Label, t uint32, c ot.Label,
	data *ot.LabelData) ot.Label {

	k := makeK(a, b, t)

	k.GetData(data)

	alg.Encrypt(data[:], data[:])

	var crypted ot.Label
	crypted.SetData(data)

	c.Xor(crypted)
	c.Xor(k)

	return c
}

func makeK(a, b ot.Label, t uint32) ot.Label {
	a.Mul2()

	b.Mul4()
	a.Xor(b)

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

// Garbled contains garbled circuit information.
type Garbled struct {
	R     ot.Label
	Wires []ot.Wire
	Gates [][]ot.Label
}

// Lambda returns the lambda value of the wire.
func (g *Garbled) Lambda(wire Wire) uint {
	if g.Wires[int(wire)].L0.S() {
		return 1
	}
	return 0
}

// SetLambda sets the lambda value of the wire.
func (g *Garbled) SetLambda(wire Wire, val uint) {
	w := g.Wires[int(wire)]
	if val == 0 {
		w.L0.SetS(false)
	} else {
		w.L0.SetS(true)
	}
	g.Wires[int(wire)] = w
}

// Garble garbles the circuit.
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

	// Wire labels.
	wires := make([]ot.Wire, c.NumWires)

	// Assing all input wires.
	for i := 0; i < c.Inputs.Size(); i++ {
		w, err := makeLabels(r)
		if err != nil {
			return nil, err
		}
		wires[i] = w
	}

	// Garble gates.
	var data ot.LabelData
	for i := 0; i < len(c.Gates); i++ {
		gate := &c.Gates[i]
		data, err := gate.Garble(wires, alg, r, uint32(i), &data)
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

// Garble garbles the gate and returns it labels.
func (g *Gate) Garble(wires []ot.Wire, enc cipher.Block, r ot.Label,
	id uint32, data *ot.LabelData) ([]ot.Label, error) {

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

	var table [4]ot.Label
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
		table[idx(a.L0, b.L0)] = encrypt(enc, a.L0, b.L0, c.L0, id, data)
		table[idx(a.L0, b.L1)] = encrypt(enc, a.L0, b.L1, c.L0, id, data)
		table[idx(a.L1, b.L0)] = encrypt(enc, a.L1, b.L0, c.L0, id, data)
		table[idx(a.L1, b.L1)] = encrypt(enc, a.L1, b.L1, c.L1, id, data)
		count = 4

	case OR:
		// a b c
		// -----
		// 0 0 0
		// 0 1 1
		// 1 0 1
		// 1 1 1
		table[idx(a.L0, b.L0)] = encrypt(enc, a.L0, b.L0, c.L0, id, data)
		table[idx(a.L0, b.L1)] = encrypt(enc, a.L0, b.L1, c.L1, id, data)
		table[idx(a.L1, b.L0)] = encrypt(enc, a.L1, b.L0, c.L1, id, data)
		table[idx(a.L1, b.L1)] = encrypt(enc, a.L1, b.L1, c.L1, id, data)
		count = 4

	case INV:
		// a b c
		// -----
		// 0   1
		// 1   0
		table[idxUnary(a.L0)] = encrypt(enc, a.L0, ot.Label{}, c.L1, id, data)
		table[idxUnary(a.L1)] = encrypt(enc, a.L1, ot.Label{}, c.L0, id, data)
		count = 2

	default:
		return nil, fmt.Errorf("invalid operand %s", g.Op)
	}

	return table[:count], nil
}
