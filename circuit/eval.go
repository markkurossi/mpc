//
// eval.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/aes"
	"fmt"

	"github.com/markkurossi/mpc/ot"
)

func (c *Circuit) Eval(key []byte, wires map[Wire]*ot.Label,
	garbled map[int][][]byte) error {

	alg, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	dec := func(a, b *ot.Label, t uint32, data []byte) ([]byte, error) {
		return decrypt(alg, a, b, t, data)
	}

	for id := 0; id < c.NumGates; id++ {
		gate := c.Gates[id]

		output, err := gate.Eval(wires, dec, garbled[id])
		if err != nil {
			return err
		}
		wires[gate.Outputs[0]] = ot.LabelFromData(output)
	}

	return nil
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
