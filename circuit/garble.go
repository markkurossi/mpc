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
	"time"

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

func encrypt(alg cipher.Block, a, b, c *ot.Label, t uint32) []byte {
	k := makeK(a, b, t)

	crypted := make([]byte, alg.BlockSize())
	alg.Encrypt(crypted, k.Bytes())

	pi := ot.LabelFromData(crypted)
	pi.Xor(k)
	pi.Xor(c)

	return pi.Bytes()
}

func decrypt(alg cipher.Block, a, b *ot.Label, t uint32, encrypted []byte) (
	[]byte, error) {

	k := makeK(a, b, t)

	crypted := make([]byte, alg.BlockSize())
	alg.Encrypt(crypted, k.Bytes())

	c := ot.LabelFromData(encrypted)
	c.Xor(ot.LabelFromData(crypted))
	c.Xor(k)

	return c.Bytes(), nil
}

func makeK(a, b *ot.Label, t uint32) *ot.Label {
	k := a.Copy()
	k.Mul2()

	if b != nil {
		tmp := b.Copy()
		tmp.Mul4()
		k.Xor(tmp)
	}
	k.Xor(ot.NewTweak(t))

	return k
}

type Garbled struct {
	Wires ot.Inputs
	Gates map[int][][]byte
}

func (c *Circuit) Garble(key []byte) (*Garbled, error) {
	// Assign labels to wires.
	start := time.Now()
	wires := make(ot.Inputs)
	for w := 0; w < c.NumWires; w++ {
		l0, err := ot.NewLabel(rand.Reader)
		if err != nil {
			return nil, err
		}
		l1, err := ot.NewLabel(rand.Reader)
		if err != nil {
			return nil, err
		}

		// Point-and-permutate

		var s [1]byte
		if _, err := rand.Read(s[:]); err != nil {
			return nil, err
		}

		ws := (s[0] & 0x80) != 0

		l0.SetS(ws)
		l1.SetS(!ws)

		wires[w] = ot.Wire{
			Label0: l0,
			Label1: l1,
		}
	}
	t := time.Now()
	fmt.Printf("Garble.Labels:\t%s\n", t.Sub(start))
	start = t

	garbled := make(map[int][][]byte)

	alg, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	enc := func(a, b, c *ot.Label, t uint32) []byte {
		return encrypt(alg, a, b, c, t)
	}

	for id, gate := range c.Gates {
		data, err := gate.Garble(wires, enc)
		if err != nil {
			return nil, err
		}
		garbled[id] = data
	}
	t = time.Now()
	fmt.Printf("Garble.Garble:\t%s\n", t.Sub(start))
	start = t

	return &Garbled{
		Wires: wires,
		Gates: garbled,
	}, nil
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
