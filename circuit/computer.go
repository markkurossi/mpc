//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
)

func (c *Circuit) Compute(n1, n2 []uint64) ([]uint64, error) {
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

	var args []uint64
	args = append(args, n1...)
	args = append(args, n2...)

	var w int
	for idx, io := range ios {
		a := args[idx]
		for bit := 0; bit < io.Size; bit++ {
			if a&(1<<bit) != 0 {
				wires[w] = 1
			}
			w++
		}
	}

	// Evaluate circuit.
	for _, gate := range c.Gates {
		var result byte

		switch gate.Op {
		case XOR:
			result = wires[gate.Inputs[0]] ^ wires[gate.Inputs[1]]

		case AND:
			result = wires[gate.Inputs[0]] & wires[gate.Inputs[1]]

		case OR:
			result = wires[gate.Inputs[0]] | wires[gate.Inputs[1]]

		case INV:
			if wires[gate.Inputs[0]] == 0 {
				result = 1
			} else {
				result = 0
			}

		default:
			return nil, fmt.Errorf("invalid gate %s", gate.Op)
		}

		wires[gate.Outputs[0]] = result
	}

	// Construct outputs
	w = c.NumWires - c.N3.Size()
	var result []uint64
	for _, io := range c.N3 {
		var r uint64
		for bit := 0; bit < io.Size; bit++ {
			if wires[w] != 0 {
				r |= 1 << bit
			}
			w++
		}
		result = append(result, r)
	}

	return result, nil
}
