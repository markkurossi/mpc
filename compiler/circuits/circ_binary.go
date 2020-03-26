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

func NewBinaryAND(compiler *Compiler, x, y, r []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(r) != len(x) {
		return fmt.Errorf("invalid binary and arguments: r=%d", len(r))
	}
	for i := 0; i < len(x); i++ {
		compiler.AddGate(NewBinary(circuit.AND, x[i], y[i], r[i]))
	}
	return nil
}

func NewBinaryClear(compiler *Compiler, x, y, r []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(r) != len(x) {
		return fmt.Errorf("invalid binary clear arguments: r=%d", len(r))
	}
	for i := 0; i < len(x); i++ {
		w := NewWire()
		compiler.INV(y[i], w)
		compiler.AddGate(NewBinary(circuit.AND, x[i], w, r[i]))
	}
	return nil
}

func NewBinaryOR(compiler *Compiler, x, y, r []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(r) != len(x) {
		return fmt.Errorf("invalid binary or arguments: r=%d", len(r))
	}
	for i := 0; i < len(x); i++ {
		compiler.AddGate(NewBinary(circuit.OR, x[i], y[i], r[i]))
	}
	return nil
}

func NewBinaryXOR(compiler *Compiler, x, y, r []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(r) != len(x) {
		return fmt.Errorf("invalid binary xor arguments: r=%d", len(r))
	}
	for i := 0; i < len(x); i++ {
		compiler.AddGate(NewBinary(circuit.XOR, x[i], y[i], r[i]))
	}
	return nil
}
