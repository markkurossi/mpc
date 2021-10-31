//
// Copyright (c) 2021 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"
)

// NewIndex creates a new array element selection (index) circuit.
func NewIndex(compiler *Compiler, size int, array, index, out []*Wire) error {
	if len(array)%size != 0 {
		return fmt.Errorf("array width %d must be multiple of element size %d",
			len(array), size)
	}
	if len(out) < size {
		return fmt.Errorf("out %d too small for element size %d",
			len(out), size)
	}
	n := len(array) / size
	if n == 0 {
		for i := 0; i < len(out); i++ {
			out[i] = compiler.ZeroWire()
		}
		return nil
	}
	selected := make([]*Wire, size)
	for i := 0; i < size; i++ {
		selected[i] = compiler.ZeroWire()
	}

	// Number of bits needed for indices.
	var bits int
	for bits = 1; bits < size; bits *= 2 {
	}
	fmt.Printf("selecting with %d bits\n", bits)

	if len(index) > bits {
		index = index[0:bits]
	}

	for n--; n >= 0; n-- {
		// Compare index to n, result to cond
		nWires := make([]*Wire, bits)
		for i := 0; i < bits; i++ {
			if n&(1<<i) == 0 {
				nWires[i] = compiler.ZeroWire()
			} else {
				nWires[i] = compiler.OneWire()
			}
		}
		nWires, index = compiler.ZeroPad(nWires, index)

		cond := make([]*Wire, 1)
		cond[0] = NewWire()
		err := NewEqComparator(compiler, nWires, index, cond)
		if err != nil {
			return err
		}

		// MUX cond, array[n*size:(n+1)*size], selected

		var result []*Wire
		if n == 0 {
			result = out
		} else {
			result = make([]*Wire, size)
			for i := 0; i < size; i++ {
				result[i] = NewWire()
			}
		}
		err = NewMUX(compiler, cond, array[n*size:(n+1)*size], selected, result)
		if err != nil {
			return err
		}

		selected = result
	}

	return nil
}
