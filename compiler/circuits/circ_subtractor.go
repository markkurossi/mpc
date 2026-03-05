//
// circ_subtractor.go
//
// Copyright (c) 2019-2026 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
)

// NewFullSubtractor creates a full subtractor circuit.
func NewFullSubtractor(cc *Compiler, x, y, cin, d, cout *Wire) {
	w1 := cc.Calloc.Wire()
	cc.AddGate(cc.Calloc.BinaryGate(circuit.XNOR, y, cin, w1))
	cc.AddGate(cc.Calloc.BinaryGate(circuit.XNOR, x, w1, d))

	if cout != nil {
		w2 := cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, x, cin, w2))

		w3 := cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, w1, w2, w3))

		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, w3, cin, cout))
	}
}

// NewSubtractor creates a new subtractor circuit implementing z=x-y.
func NewSubtractor(cc *Compiler, x, y, z []*Wire) error {
	if cc.Params.Target == utils.TargetGMW {
		return NewKoggeStoneSubtractor(cc, x, y, z)
	}

	x, y = cc.ZeroPad(x, y)
	if len(x) > len(z) {
		x = x[0:len(z)]
		y = y[0:len(z)]
	}
	cin := cc.ZeroWire()

	for i := 0; i < len(x); i++ {
		var cout *Wire
		if i+1 >= len(x) {
			if i+1 >= len(z) {
				// N-N=N, overflow, drop carry bit.
				cout = nil
			} else {
				cout = z[i+1]
			}
		} else {
			cout = cc.Calloc.Wire()
		}

		// Note y-x here.
		NewFullSubtractor(cc, y[i], x[i], cin, z[i], cout)

		cin = cout
	}
	for i := len(x) + 1; i < len(z); i++ {
		z[i] = cc.ZeroWire()
	}
	return nil
}

// NewKoggeStoneSubtractor creates a Kogge-Stone parallel prefix
// subtractor.  It achieves O(log n) AND-depth at the cost of a larger
// gate count.
func NewKoggeStoneSubtractor(cc *Compiler, x, y, z []*Wire) error {
	n := len(x)
	if n < len(y) {
		n = len(y)
	}
	if len(z) > n {
		n++
	}
	x = cc.Pad(x, n)
	y = cc.Pad(y, n)

	if len(x) > len(z) {
		x = x[0:len(z)]
		y = y[0:len(z)]
		n = len(z)
	}
	pInit := make([]*Wire, n)
	p := make([]*Wire, n)
	g := make([]*Wire, n)

	// Bitwise preparation & 2's complement.
	for i := 0; i < n; i++ {
		bInv := cc.Calloc.Wire()
		cc.INV(y[i], bInv)

		p[i] = cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, x[i], bInv, p[i]))
		pInit[i] = p[i]

		g[i] = cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, x[i], bInv, g[i]))
	}

	// Incorporate +1 for 2's complement (Cin = 1).  Carry out of bit
	// 0 is normally: G_0 OR (P_0 AND Cin) Since Cin=1, it becomes G_0
	// OR P_0.  Because G_0 and P_0 are mutually exclusive, OR becomes
	// XOR.
	w := cc.Calloc.Wire()
	cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, g[0], p[0], w))
	g[0] = w

	// Prefix network (logarithmic depth).
	for step := 1; step < n; step *= 2 {
		nextG := make([]*Wire, n)
		nextP := make([]*Wire, n)

		// Wires that don't reach far back enough just carry their
		// values forward.
		for i := 0; i < step; i++ {
			nextG[i] = g[i]
			nextP[i] = p[i]
		}

		// Parallel computation for the current tree level.
		for i := step; i < n; i++ {
			// Calculate combined Generate: G_i = G_i XOR (P_i AND G_{i-step})
			// (Again, replacing OR with XOR due to mutual exclusivity)
			pg := cc.Calloc.Wire()
			cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, p[i], g[i-step], pg))

			w := cc.Calloc.Wire()
			cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, g[i], pg, w))
			nextG[i] = w

			// Calculate combined Propagate: P_i = P_i AND P_{i-step}
			w = cc.Calloc.Wire()
			cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, p[i], p[i-step], w))
			nextP[i] = w
		}

		g = nextG
		p = nextP
	}

	// Calculate sum.

	// Bit 0 sum includes our theoretical Cin=1.
	cc.INV(pInit[0], z[0])

	// Remaining bits use the calculated carry signals (which are now in G).
	for i := 1; i < n; i++ {
		cc.AddGate(cc.Calloc.BinaryGate(circuit.XOR, pInit[i], g[i-1], z[i]))
	}

	// Set all leftover bits to zero.
	for i := n; i < len(z); i++ {
		z[i] = cc.ZeroWire()
	}

	return nil
}
