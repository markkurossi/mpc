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

// NewMUX creates a multiplexer circuit that selects the input t or f
// to output, based on the value of the condition cond.
func NewMUX(compiler *Compiler, cond, t, f, out []*Wire) error {
	t, f = compiler.ZeroPad(t, f)
	if len(cond) != 1 || len(t) != len(f) || len(t) != len(out) {
		return fmt.Errorf("invalid mux arguments: cond=%d, l=%d, r=%d, out=%d",
			len(cond), len(t), len(f), len(out))
	}

	for i := 0; i < len(t); i++ {
		w1 := NewWire()
		w2 := NewWire()

		// w1 = XOR(f[i], t[i])
		compiler.AddGate(NewBinary(circuit.XOR, f[i], t[i], w1))

		// w2 = AND(w1, cond)
		compiler.AddGate(NewBinary(circuit.AND, w1, cond[0], w2))

		// out[i] = XOR(w2, f[i])
		compiler.AddGate(NewBinary(circuit.XOR, w2, f[i], out[i]))
	}

	return nil
}
