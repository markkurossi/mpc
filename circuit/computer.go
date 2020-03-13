//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
	"math/big"
)

func (c *Circuit) Compute(n1, n2 []*big.Int) ([]*big.Int, error) {
	if len(n1) != len(c.N1) {
		return nil, fmt.Errorf("invalid n1 arguments: got %d, expected %d\n",
			len(n1), len(c.N1))
	}
	if len(n2) != len(c.N2) {
		return nil, fmt.Errorf("invalid n2 arguments: got %d, expected %d\n",
			len(n2), len(c.N2))
	}

	wires := make([]byte, c.NumWires)
	var ios IO
	ios = append(ios, c.N1...)
	ios = append(ios, c.N2...)

	var args []*big.Int
	args = append(args, n1...)
	args = append(args, n2...)

	var w int
	for idx, io := range ios {
		for bit := 0; bit < io.Size; bit++ {
			wires[w] = byte(args[idx].Bit(bit))
			w++
		}
	}

	// Evaluate circuit.
	for _, gate := range c.Gates {
		var result byte

		switch gate.Op {
		case XOR:
			result = wires[gate.Input0] ^ wires[gate.Input1]

		case AND:
			result = wires[gate.Input0] & wires[gate.Input1]

		case OR:
			result = wires[gate.Input0] | wires[gate.Input1]

		case INV:
			if wires[gate.Input0] == 0 {
				result = 1
			} else {
				result = 0
			}

		default:
			return nil, fmt.Errorf("invalid gate %s", gate.Op)
		}

		wires[gate.Output] = result
	}

	// Construct outputs
	w = c.NumWires - c.N3.Size()
	var result []*big.Int
	for _, io := range c.N3 {
		r := new(big.Int)
		for bit := 0; bit < io.Size; bit++ {
			if wires[w] != 0 {
				r.SetBit(r, bit, 1)
			}
			w++
		}
		result = append(result, r)
	}

	return result, nil
}
