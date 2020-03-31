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

func (c *Circuit) Compute(inputs []*big.Int) ([]*big.Int, error) {
	if len(inputs) != len(c.Inputs) {
		return nil, fmt.Errorf("invalid inputs: got %d, expected %d",
			len(inputs), len(c.Inputs))
	}

	// Flatten inputs and arguments.
	wires := make([]byte, c.NumWires)

	var w int
	for idx, io := range c.Inputs {
		for bit := 0; bit < io.Size; bit++ {
			wires[w] = byte(inputs[idx].Bit(bit))
			w++
		}
	}

	// Evaluate circuit.
	for _, gate := range c.Gates {
		var result byte

		switch gate.Op {
		case XOR:
			result = wires[gate.Input0] ^ wires[gate.Input1]

		case XNOR:
			if wires[gate.Input0]^wires[gate.Input1] == 0 {
				result = 1
			} else {
				result = 0
			}

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
	w = c.NumWires - c.Outputs.Size()
	var result []*big.Int
	for _, io := range c.Outputs {
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
