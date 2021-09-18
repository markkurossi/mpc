//
// Copyright (c) 2020-2021 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/types"
)

// comparator tests if x>y if cin=0, and x>=y if cin=1.
func comparator(compiler *Compiler, cin *Wire, x, y, r []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(r) != 1 {
		return fmt.Errorf("invalid lt comparator arguments: r=%d", len(r))
	}

	for i := 0; i < len(x); i++ {
		w1 := NewWire()
		compiler.AddGate(NewBinary(circuit.XNOR, cin, y[i], w1))
		w2 := NewWire()
		compiler.AddGate(NewBinary(circuit.XOR, cin, x[i], w2))
		w3 := NewWire()
		compiler.AddGate(NewBinary(circuit.AND, w1, w2, w3))

		var cout *Wire
		if i+1 < len(x) {
			cout = NewWire()
		} else {
			cout = r[0]
		}
		compiler.AddGate(NewBinary(circuit.XOR, cin, w3, cout))
		cin = cout
	}
	return nil
}

// NewGtComparator tests if x>y.
func NewGtComparator(compiler *Compiler, x, y, r []*Wire) error {
	return comparator(compiler, compiler.ZeroWire(), x, y, r)
}

// NewGeComparator tests if x>=y.
func NewGeComparator(compiler *Compiler, x, y, r []*Wire) error {
	return comparator(compiler, compiler.OneWire(), x, y, r)
}

// NewLtComparator tests if x<y.
func NewLtComparator(compiler *Compiler, x, y, r []*Wire) error {
	return comparator(compiler, compiler.ZeroWire(), y, x, r)
}

// NewLeComparator tests if x<=y.
func NewLeComparator(compiler *Compiler, x, y, r []*Wire) error {
	return comparator(compiler, compiler.OneWire(), y, x, r)
}

// NewNeqComparator tewsts if x!=y.
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

// NewEqComparator tests if x==y.
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
	compiler.INV(w, r[0])
	return nil
}

// NewLogicalAND implements logical AND implementing r=x&y. The input
// and output wires must be 1 bit wide.
func NewLogicalAND(compiler *Compiler, x, y, r []*Wire) error {
	if len(x) != 1 || len(y) != 1 || len(r) != 1 {
		return fmt.Errorf("invalid logical and arguments: x=%d, y=%d, r=%d",
			len(x), len(y), len(r))
	}
	compiler.AddGate(NewBinary(circuit.AND, x[0], y[0], r[0]))
	return nil
}

// NewLogicalOR implements logical OR implementing r=x|y.  The input
// and output wires must be 1 bit wide.
func NewLogicalOR(compiler *Compiler, x, y, r []*Wire) error {
	if len(x) != 1 || len(y) != 1 || len(r) != 1 {
		return fmt.Errorf("invalid logical or arguments: x=%d, y=%d, r=%d",
			len(x), len(y), len(r))
	}
	compiler.AddGate(NewBinary(circuit.OR, x[0], y[0], r[0]))
	return nil
}

// NewBitSetTest tests if the index'th bit of x is set.
func NewBitSetTest(compiler *Compiler, x []*Wire, index types.Size,
	r []*Wire) error {

	if len(r) != 1 {
		return fmt.Errorf("invalid bit set test arguments: x=%d, r=%d",
			len(x), len(r))
	}
	if index < types.Size(len(x)) {
		w := compiler.ZeroWire()
		compiler.AddGate(NewBinary(circuit.XOR, x[index], w, r[0]))
	} else {
		r[0] = compiler.ZeroWire()
	}
	return nil
}

// NewBitClrTest tests if the index'th bit of x is unset.
func NewBitClrTest(compiler *Compiler, x []*Wire, index types.Size,
	r []*Wire) error {

	if len(r) != 1 {
		return fmt.Errorf("invalid bit clear test arguments: x=%d, r=%d",
			len(x), len(r))
	}
	if index < types.Size(len(x)) {
		w := compiler.OneWire()
		compiler.AddGate(NewBinary(circuit.XOR, x[index], w, r[0]))
	} else {
		r[0] = compiler.OneWire()
	}
	return nil
}
