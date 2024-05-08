//
// Copyright (c) 2024 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"
	"time"

	"github.com/markkurossi/mpc/circuit"
)

// Rewrite applies rewrite patterns to the circuit.
func (cc *Compiler) Rewrite() {
	var stats circuit.Stats

	start := time.Now()

	for _, g := range cc.Gates {
		switch g.Op {
		case circuit.AND:
			// AND(A,A) = A
			if g.A == g.B && g.O.NumOutputs() > 0 {
				stats[g.Op]++
				g.ShortCircuit(g.A)
			}
		case circuit.OR:
			// OR(A,A) = A
			if g.A == g.B && g.O.NumOutputs() > 0 {
				stats[g.Op]++
				g.ShortCircuit(g.A)
			}
		case circuit.XOR:
			// XOR(A,A) = 0
			if g.A == g.B && g.O.NumOutputs() > 0 {
				stats[g.Op]++
				g.O.SetValue(Zero)
			}
		}
	}

	elapsed := time.Since(start)

	if cc.Params.Diagnostics {
		fmt.Printf(" - Rewrite:             %12s: %d/%d (%.2f%%)\n",
			elapsed, stats.Count(), len(cc.Gates),
			float64(stats.Count())/float64(len(cc.Gates))*100)
	}
}
