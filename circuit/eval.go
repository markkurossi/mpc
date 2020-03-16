//
// Copyright (c) 2019-2020 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/aes"
	"fmt"

	"github.com/markkurossi/mpc/ot"
)

func (c *Circuit) Eval(key []byte, wires []*ot.Label,
	garbled [][][]byte) error {

	alg, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	dec := func(a, b *ot.Label, t uint32, data []byte) ([]byte, error) {
		return decrypt(alg, a, b, t, data)
	}

	for i := 0; i < len(c.Gates); i++ {
		gate := &c.Gates[i]

		var a *ot.Label
		var b *ot.Label

		switch gate.Op {
		case XOR, AND, OR:
			a = wires[gate.Input0]
			b = wires[gate.Input1]

		case INV:
			a = wires[gate.Input0]
			b = nil

		default:
			return fmt.Errorf("invalid operation %s", gate.Op)
		}

		var output []byte

		if gate.Op == XOR {
			result := a.Copy()
			result.Xor(b)
			output = result.Bytes()
		} else {
			row := garbled[i]
			index := idx(a, b)
			if index >= len(row) {
				return fmt.Errorf("corrupted circuit: index %d >= row len %d",
					index, len(row))
			}
			output, err = dec(a, b, uint32(i), row[index])
			if err != nil {
				return err
			}
		}

		wires[gate.Output] = ot.LabelFromData(output)
	}

	return nil
}
