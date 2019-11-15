//
// circ_subtractor.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"

	"github.com/markkurossi/mpc/circuit"
)

func NewHalfSubtractor(compiler *Compiler, a, b, diff, bout *Wire) {
	w1 := NewWire()

	compiler.AddGate(NewBinary(circuit.XOR, a, b, diff))
	compiler.AddGate(NewINV(a, w1))
	compiler.AddGate(NewBinary(circuit.AND, w1, b, bout))
}

func NewFullSubtractor(compiler *Compiler, a, b, bin, diff, bout *Wire) {
	w3 := NewWire()
	w4 := NewWire()
	w5 := NewWire()
	w6 := NewWire()
	w7 := NewWire()

	compiler.AddGate(NewBinary(circuit.XOR, a, b, w3))
	compiler.AddGate(NewBinary(circuit.XOR, bin, w3, diff))
	compiler.AddGate(NewINV(a, w4))
	compiler.AddGate(NewBinary(circuit.AND, b, w4, w5))
	compiler.AddGate(NewINV(w3, w6))
	compiler.AddGate(NewBinary(circuit.AND, bin, w6, w7))
	compiler.AddGate(NewBinary(circuit.OR, w5, w7, bout))
}

func NewSubtractor(compiler *Compiler, x, y, z []*Wire) error {
	if len(x) != len(y) || len(x)+1 != len(z) {
		return fmt.Errorf("Invalid subtractor arguments: x=%d, y=%d, z=%d",
			len(x), len(y), len(z))
	}
	if len(x) == 1 {
		NewHalfSubtractor(compiler, x[0], y[0], z[0], z[1])
	} else {
		bin := NewWire()
		NewHalfSubtractor(compiler, x[0], y[0], z[0], bin)

		for i := 1; i < len(x); i++ {
			var bout *Wire
			if i+1 >= len(x) {
				bout = z[len(x)]
			} else {
				bout = NewWire()
			}

			NewFullSubtractor(compiler, x[i], y[i], bin, z[i], bout)

			bin = bout
		}
	}
	return nil
}
