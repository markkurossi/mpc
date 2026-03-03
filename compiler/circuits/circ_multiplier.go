//
// circ_multiplier.go
//
// Copyright (c) 2019-2023, 2026 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/types"
)

// NewMultiplier creates a multiplier circuit implementing x*y=z.
func NewMultiplier(c *Compiler, arrayTreshold int, x, y, z []*Wire) error {
	if c.Params.Target == utils.TargetGMW {
		return NewWallaceMultiplier(c, x, y, z)
	}
	if false {
		return NewArrayMultiplier(c, x, y, z)
	}
	if arrayTreshold < 8 {
		var ok bool

		arrayTreshold, ok = multiplierArrayTresholds[len(x)]
		if !ok {
			arrayTreshold = 21
		}
	}
	return NewKaratsubaMultiplier(c, arrayTreshold, x, y, z)
}

// NewArrayMultiplier creates a multiplier circuit implementing
// x*y=z. This function implements Array Multiplier Circuit.
func NewArrayMultiplier(cc *Compiler, x, y, z []*Wire) error {
	x, y = cc.ZeroPad(x, y)
	if len(x) > len(z) {
		x = x[0:len(z)]
		y = y[0:len(z)]
	}

	// One bit multiplication is AND.
	if len(x) == 1 {
		cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, x[0], y[0], z[0]))
		if len(z) > 1 {
			z[1] = cc.ZeroWire()
		}
		return nil
	}

	var sums []*Wire

	// Construct Y0 sums
	for i, xn := range x {
		var s *Wire
		if i == 0 {
			s = z[0]
		} else {
			s = cc.Calloc.Wire()
			sums = append(sums, s)
		}
		cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, xn, y[0], s))
	}

	// Construct len(y)-2 intermediate layers
	var j int
	for j = 1; j+1 < len(y); j++ {
		// ANDs for y(j)
		var ands []*Wire
		for _, xn := range x {
			wire := cc.Calloc.Wire()
			cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, xn, y[j], wire))
			ands = append(ands, wire)
		}

		// Compute next sums.
		var nsums []*Wire
		var c *Wire
		for i := 0; i < len(ands); i++ {
			cout := cc.Calloc.Wire()

			var s *Wire
			if i == 0 {
				s = z[j]
			} else {
				s = cc.Calloc.Wire()
				nsums = append(nsums, s)
			}

			if i == 0 {
				NewHalfAdder(cc, ands[i], sums[i], s, cout)
			} else if i >= len(sums) {
				NewHalfAdder(cc, ands[i], c, s, cout)
			} else {
				NewFullAdder(cc, ands[i], sums[i], c, s, cout)
			}
			c = cout
		}
		// New sums with carry as the highest bit.
		sums = append(nsums, c)
	}

	// Construct final layer.
	var c *Wire
	for i, xn := range x {
		and := cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, xn, y[j], and))

		var cout *Wire
		if i+1 >= len(x) && j+i+1 < len(z) {
			cout = z[j+i+1]
		} else {
			cout = cc.Calloc.Wire()
		}

		if j+i < len(z) {
			if i == 0 {
				NewHalfAdder(cc, and, sums[i], z[j+i], cout)
			} else if i >= len(sums) {
				NewHalfAdder(cc, and, c, z[j+i], cout)
			} else {
				NewFullAdder(cc, and, sums[i], c, z[j+i], cout)
			}
		}
		c = cout
	}
	for i := j + len(x) + 1; i < len(z); i++ {
		z[1] = cc.ZeroWire()
	}

	return nil
}

// NewKaratsubaMultiplier creates a multiplier circuit implementing
// the Karatsuba algorithm
// (https://en.wikipedia.org/wiki/Karatsuba_algorithm). The Karatsuba
// algorithm is should produce faster circuits on inputs of about 128
// bits (the number of non-XOR gates is smaller). On input sizes of
// 256 bits also the overall circuits are smaller than with the array
// multiplier algorithm.
//
//	Bits  Array     a-xor     a-and    Karatsu   K-xor    K-and
//	8     301       172       129      993       724      269
//	16    1365      852       513      3573      2660     913
//	32    5797      3748      2049     11937     8980     2957
//	64    23877     15684     8193     38277     28964    9313
//	128   96901     64132     32769    119793    90964    28829
//	256   390405    259332    131073   369333    281060   88273
//	512   1567237   1042948   524289   1127937   859540   268397
//	1024  6280197   4183044   2097153  3423717   2611364  812353
//	2048  25143301  16754692  8388609  10350993  7899604  2451389
func NewKaratsubaMultiplier(cc *Compiler, limit int, a, b, r []*Wire) error {

	a, b = cc.ZeroPad(a, b)
	if len(a) > len(r) {
		a = a[0:len(r)]
		b = b[0:len(r)]
	}

	// Compute smaller multiplications with array multiplier.
	if len(a) <= limit {
		return NewArrayMultiplier(cc, a, b, r)
	}

	mid := len(a) / 2

	aLow := a[:mid]
	aHigh := a[mid:]

	bLow := b[:mid]
	bHigh := b[mid:]

	z0 := cc.Calloc.Wires(types.Size(min(max(len(aLow), len(bLow))*2, len(r))))
	if err := NewKaratsubaMultiplier(cc, limit, aLow, bLow, z0); err != nil {
		return err
	}
	aSumLen := max(len(aLow), len(aHigh)) + 1
	aSum := cc.Calloc.Wires(types.Size(aSumLen))
	if err := NewAdder(cc, aLow, aHigh, aSum); err != nil {
		return err
	}
	bSumLen := max(len(bLow), len(bHigh)) + 1
	bSum := cc.Calloc.Wires(types.Size(bSumLen))
	if err := NewAdder(cc, bLow, bHigh, bSum); err != nil {
		return err
	}
	z1 := cc.Calloc.Wires(types.Size(min(max(aSumLen, bSumLen)*2, len(r))))
	if err := NewKaratsubaMultiplier(cc, limit, aSum, bSum, z1); err != nil {
		return err
	}
	z2 := cc.Calloc.Wires(types.Size(min(max(len(aHigh), len(bHigh))*2, len(r))))
	if err := NewKaratsubaMultiplier(cc, limit, aHigh, bHigh, z2); err != nil {
		return err
	}

	sub1 := cc.Calloc.Wires(types.Size(len(r)))
	if err := NewSubtractor(cc, z1, z2, sub1); err != nil {
		return err
	}
	sub2 := cc.Calloc.Wires(types.Size(len(r)))
	if err := NewSubtractor(cc, sub1, z0, sub2); err != nil {
		return err
	}

	shift1 := cc.ShiftLeft(z2, len(r), mid*2)
	shift2 := cc.ShiftLeft(sub2, len(r), mid)

	add1 := cc.Calloc.Wires(types.Size(len(r)))
	if err := NewAdder(cc, shift1, shift2, add1); err != nil {
		return err
	}

	return NewAdder(cc, add1, z0, r)
}

// NewWallaceMultiplier implements multiplication using a Wallace
// tree reduction. It reduces AND-depth at the cost of a larger
// gate count and increased circuit width.
func NewWallaceMultiplier(cc *Compiler, a, b, r []*Wire) error {
	n := len(r)

	a = cc.Pad(a, n)
	b = cc.Pad(b, n)

	if len(a) > len(r) {
		a = a[0:len(r)]
		b = b[0:len(r)]
		n = len(b)
	}
	columns := make([][]*Wire, 2*n)

	// 1. Partial Product Generation (Depth 1).
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			and := cc.Calloc.Wire()
			cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, a[i], b[j], and))
			columns[i+j] = append(columns[i+j], and)
		}
	}

	// 2. Wallace Tree Reduction (Depth log_1.5(n))
	for {
		maxH := 0
		for _, col := range columns {
			if len(col) > maxH {
				maxH = len(col)
			}
		}
		if maxH <= 2 {
			break
		}

		nextColumns := make([][]*Wire, 2*n)
		for i := 0; i < len(columns); i++ {
			j := 0
			for j+2 < len(columns[i]) {
				s, c := cc.Calloc.Wire(), cc.Calloc.Wire()
				NewFullAdder(cc,
					columns[i][j], columns[i][j+1], columns[i][j+2], s, c)

				nextColumns[i] = append(nextColumns[i], s)
				if i+1 < len(nextColumns) {
					nextColumns[i+1] = append(nextColumns[i+1], c)
				}
				j += 3
			}
			if j+1 < len(columns[i]) {
				s, c := cc.Calloc.Wire(), cc.Calloc.Wire()
				NewHalfAdder(cc, columns[i][j], columns[i][j+1], s, c)

				nextColumns[i] = append(nextColumns[i], s)
				if i+1 < len(nextColumns) {
					nextColumns[i+1] = append(nextColumns[i+1], c)
				}
				j += 2
			}
			if j < len(columns[i]) {
				nextColumns[i] = append(nextColumns[i], columns[i][j])
			}
		}
		columns = nextColumns
	}

	// 3. Prepare rows for final addition
	row1 := make([]*Wire, n)
	row2 := make([]*Wire, n)
	for i := 0; i < n; i++ {
		if len(columns[i]) > 0 {
			row1[i] = columns[i][0]
		} else {
			row1[i] = cc.ZeroWire()
		}
		if len(columns[i]) > 1 {
			row2[i] = columns[i][1]
		} else {
			row2[i] = cc.ZeroWire()
		}
	}

	// 4. Final Addition using Kogge-Stone (Depth log2(n)).
	return NewKoggeStoneAdder(cc, row1, row2, r)

}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
