//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"

	"github.com/markkurossi/mpc/circuit"
)

// NewHalfLtComparator creates a circuit that tests if argument a is
// smaller than argument b.
func NewHalfLtComparator(compiler *Compiler, a, b, bout *Wire) {
	w1 := NewWire()

	compiler.AddGate(NewINV(a, w1))
	compiler.AddGate(NewBinary(circuit.AND, w1, b, bout))
}

// NewFullLtComparator creates a circuit that tests if argument a is
// smaller than argument b with the borrow bit bin.
func NewFullLtComparator(compiler *Compiler, a, b, bin, bout *Wire) {
	w3 := NewWire()
	w4 := NewWire()
	w5 := NewWire()
	w6 := NewWire()
	w7 := NewWire()

	compiler.AddGate(NewBinary(circuit.XOR, a, b, w3))
	compiler.AddGate(NewINV(a, w4))
	compiler.AddGate(NewBinary(circuit.AND, b, w4, w5))
	compiler.AddGate(NewINV(w3, w6))
	compiler.AddGate(NewBinary(circuit.AND, bin, w6, w7))
	compiler.AddGate(NewBinary(circuit.OR, w5, w7, bout))
}

func NewLtComparator(compiler *Compiler, x, y, r []*Wire) error {
	if len(x) != len(y) || len(r) != 1 {
		return fmt.Errorf("Invalid lt comparator arguments: x=%d, y=%d, z=%d",
			len(x), len(y), len(r))
	}
	if len(x) == 1 {
		NewHalfLtComparator(compiler, x[0], y[0], r[0])
	} else {
		bin := NewWire()
		NewHalfLtComparator(compiler, x[0], y[0], bin)

		for i := 1; i < len(x); i++ {
			var bout *Wire
			if i+1 >= len(x) {
				bout = r[0]
			} else {
				bout = NewWire()
			}

			NewFullLtComparator(compiler, x[i], y[i], bin, bout)

			bin = bout
		}
	}
	return nil
}