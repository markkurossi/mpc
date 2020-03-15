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
	x, y = compiler.ZeroPad(x, y)
	if len(r) != 1 {
		return fmt.Errorf("invalid lt comparator arguments: r=%d", len(r))
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

// NewLeComparator creates comparator circuit computing `r :=
// x<=y'. The circuit is implemented by checking that `y-x' does not
// overflow i.e. `x<=y == !(y<x)'.
func NewLeComparator(compiler *Compiler, x, y, r []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(r) != 1 {
		return fmt.Errorf("invalid le comparator arguments: r=%d", len(r))
	}

	// w = y < x
	w := NewWire()
	err := NewLtComparator(compiler, y, x, []*Wire{w})
	if err != nil {
		return err
	}

	// r = !w
	compiler.AddGate(NewINV(w, r[0]))
	return nil
}

func NewNeqComparator(compiler *Compiler, x, y, r []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(r) != 1 {
		return fmt.Errorf("invalid neq comparator arguments: r=%d", len(r))
	}

	if len(x) == 1 {
		compiler.AddGate(NewBinary(circuit.XOR, x[0], y[0], r[0]))
		return nil
	}

	c := NewWire()
	compiler.AddGate(NewBinary(circuit.XOR, x[0], y[0], c))

	for i := 1; i < len(x); i++ {
		xor := NewWire()
		compiler.AddGate(NewBinary(circuit.XOR, x[i], y[i], xor))

		var out *Wire
		if i+1 >= len(x) {
			out = r[0]
		} else {
			out = NewWire()
		}
		compiler.AddGate(NewBinary(circuit.OR, c, xor, out))
		c = out
	}
	return nil
}

func NewEqComparator(compiler *Compiler, x, y, r []*Wire) error {
	if len(r) != 1 {
		return fmt.Errorf("invalid eq comparator arguments: r=%d", len(r))
	}

	// w = x == y
	w := NewWire()
	err := NewNeqComparator(compiler, x, y, []*Wire{w})
	if err != nil {
		return err
	}
	// r = !w
	compiler.AddGate(NewINV(w, r[0]))
	return nil
}

func NewLogicalAND(compiler *Compiler, x, y, r []*Wire) error {
	if len(x) != 1 || len(y) != 1 || len(r) != 1 {
		return fmt.Errorf("invalid logical and arguments: x=%d, y=%d, r=%d",
			len(x), len(y), len(r))
	}
	compiler.AddGate(NewBinary(circuit.AND, x[0], y[0], r[0]))
	return nil
}

func NewLogicalOR(compiler *Compiler, x, y, r []*Wire) error {
	if len(x) != 1 || len(y) != 1 || len(r) != 1 {
		return fmt.Errorf("invalid logical or arguments: x=%d, y=%d, r=%d",
			len(x), len(y), len(r))
	}
	compiler.AddGate(NewBinary(circuit.OR, x[0], y[0], r[0]))
	return nil
}

func NewBitSetTest(compiler *Compiler, x []*Wire, index int, r []*Wire) error {
	if len(r) != 1 {
		return fmt.Errorf("invalid bit set test arguments: x=%d, r=%d",
			len(x), len(r))
	}
	if index < len(x) {
		w := NewWire()
		compiler.Zero(w)
		compiler.AddGate(NewBinary(circuit.XOR, x[index], w, r[0]))
	} else {
		compiler.Zero(r[0])
	}
	return nil
}
