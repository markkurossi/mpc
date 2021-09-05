//
// Copyright (c) 2020-2021 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"github.com/markkurossi/mpc/circuit"
)

// NewBinaryAND creates a new binary AND circuit implementing r=x&y
func NewBinaryAND(compiler *Compiler, x, y, r []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(r) < len(x) {
		x = x[0:len(r)]
		y = y[0:len(r)]
	}
	for i := 0; i < len(x); i++ {
		compiler.AddGate(NewBinary(circuit.AND, x[i], y[i], r[i]))
	}
	return nil
}

// NewBinaryClear creates a new binary clear circuit implementing r=x&^y.
func NewBinaryClear(compiler *Compiler, x, y, r []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(r) < len(x) {
		x = x[0:len(r)]
		y = y[0:len(r)]
	}
	for i := 0; i < len(x); i++ {
		w := NewWire()
		compiler.INV(y[i], w)
		compiler.AddGate(NewBinary(circuit.AND, x[i], w, r[i]))
	}
	return nil
}

// NewBinaryOR creates a new binary OR circuit implementing r=x|y.
func NewBinaryOR(compiler *Compiler, x, y, r []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(r) < len(x) {
		x = x[0:len(r)]
		y = y[0:len(r)]
	}
	for i := 0; i < len(x); i++ {
		compiler.AddGate(NewBinary(circuit.OR, x[i], y[i], r[i]))
	}
	return nil
}

// NewBinaryXOR creates a new binary XOR circuit implementing r=x^y.
func NewBinaryXOR(compiler *Compiler, x, y, r []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(r) < len(x) {
		x = x[0:len(r)]
		y = y[0:len(r)]
	}
	for i := 0; i < len(x); i++ {
		compiler.AddGate(NewBinary(circuit.XOR, x[i], y[i], r[i]))
	}
	return nil
}
