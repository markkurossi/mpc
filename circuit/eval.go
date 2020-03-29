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

func (c *Circuit) Eval(key []byte, wires []ot.Label,
	garbled [][]ot.Label) error {

	alg, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	dec := func(a, b ot.Label, t uint32, data ot.Label) (ot.Label, error) {
		return decrypt(alg, a, b, t, data)
	}

	for i := 0; i < len(c.Gates); i++ {
		gate := &c.Gates[i]

		var a ot.Label
		var b ot.Label

		switch gate.Op {
		case XOR, XNOR, AND, OR:
			a = wires[gate.Input0]
			b = wires[gate.Input1]

		case INV:
			a = wires[gate.Input0]

		default:
			return fmt.Errorf("invalid operation %s", gate.Op)
		}

		var output ot.Label

		switch gate.Op {
		case XOR, XNOR:
			a.Xor(b)
			output = a

		default:
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
		wires[gate.Output] = output
	}

	return nil
}
