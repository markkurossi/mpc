//
// circ_multiplier.go
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

func NewMultiplier(compiler *Compiler, x, y, z []*Wire) error {
	if true {
		return NewArrayMultiplier(compiler, x, y, z)
	} else {
		return NewKaratsubaMultiplier(compiler, x, y, z)
	}
}

// NewArrayMultiplier creates a multiplier circuit implementing
// x*y=z. This function implements Array Multiplier Circuit.
func NewArrayMultiplier(compiler *Compiler, x, y, z []*Wire) error {
	x, y = compiler.ZeroPad(x, y)
	if len(x) > len(z) || len(x)+len(y) < len(z) {
		return fmt.Errorf("Invalid multiplier arguments: x=%d, y=%d, z=%d",
			len(x), len(y), len(z))
	}

	// One bit multiplication is AND.
	if len(x) == 1 {
		compiler.AddGate(NewBinary(circuit.AND, x[0], y[0], z[0]))
		if len(z) > 1 {
			compiler.Zero(z[1])
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
			s = NewWire()
			sums = append(sums, s)
		}
		compiler.AddGate(NewBinary(circuit.AND, xn, y[0], s))
	}

	// Construct len(y)-2 intermediate layers
	var j int
	for j = 1; j+1 < len(y); j++ {
		// ANDs for y(j)
		var ands []*Wire
		for _, xn := range x {
			wire := NewWire()
			compiler.AddGate(NewBinary(circuit.AND, xn, y[j], wire))
			ands = append(ands, wire)
		}

		// Compute next sums.
		var nsums []*Wire
		var c *Wire
		for i := 0; i < len(ands); i++ {
			cout := NewWire()

			var s *Wire
			if i == 0 {
				s = z[j]
			} else {
				s = NewWire()
				nsums = append(nsums, s)
			}

			if i == 0 {
				NewHalfAdder(compiler, ands[i], sums[i], s, cout)
			} else if i >= len(sums) {
				NewHalfAdder(compiler, ands[i], c, s, cout)
			} else {
				NewFullAdder(compiler, ands[i], sums[i], c, s, cout)
			}
			c = cout
		}
		// New sums with carry as the highest bit.
		sums = append(nsums, c)
	}

	// Construct final layer.
	var c *Wire
	for i, xn := range x {
		and := NewWire()
		compiler.AddGate(NewBinary(circuit.AND, xn, y[j], and))

		var cout *Wire
		if i+1 >= len(x) {
			cout = z[len(z)-1]
		} else {
			cout = NewWire()
		}

		if j+i < len(z) {
			if i == 0 {
				NewHalfAdder(compiler, and, sums[i], z[j+i], cout)
			} else if i >= len(sums) {
				NewHalfAdder(compiler, and, c, z[j+i], cout)
			} else {
				NewFullAdder(compiler, and, sums[i], c, z[j+i], cout)
			}
		}
		c = cout
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
//   Bits  Array     a-xor     a-and    Karatsu   K-xor    K-and
//   8     301       172       129      993       724      269
//   16    1365      852       513      3573      2660     913
//   32    5797      3748      2049     11937     8980     2957
//   64    23877     15684     8193     38277     28964    9313
//   128   96901     64132     32769    119793    90964    28829
//   256   390405    259332    131073   369333    281060   88273
//   512   1567237   1042948   524289   1127937   859540   268397
//   1024  6280197   4183044   2097153  3423717   2611364  812353
//   2048  25143301  16754692  8388609  10350993  7899604  2451389
//
func NewKaratsubaMultiplier(compiler *Compiler, a, b, r []*Wire) error {
	a, b = compiler.ZeroPad(a, b)
	if len(a) > len(r) || len(a)+len(b) < len(r) {
		return fmt.Errorf("Invalid multiplier arguments: a=%d, b=%d, r=%d",
			len(a), len(b), len(r))
	}

	// One bit multiplication is AND.
	if len(a) == 1 {
		compiler.AddGate(NewBinary(circuit.AND, a[0], b[0], r[0]))
		if len(r) > 1 {
			compiler.Zero(r[1])
		}
		return nil
	}

	mid := len(a) / 2

	aLow := a[:mid]
	aHigh := a[mid:]

	bLow := b[:mid]
	bHigh := b[mid:]

	z0 := MakeWires(len(a))
	if err := NewKaratsubaMultiplier(compiler, aLow, bLow, z0); err != nil {
		return err
	}
	aSum := MakeWires(max(len(aLow), len(aHigh)))
	if err := NewAdder(compiler, aLow, aHigh, aSum); err != nil {
		return err
	}
	bSum := MakeWires(max(len(bLow), len(bHigh)))
	if err := NewAdder(compiler, bLow, bHigh, bSum); err != nil {
		return err
	}
	z1 := MakeWires(len(a))
	if err := NewKaratsubaMultiplier(compiler, aSum, bSum, z1); err != nil {
		return err
	}
	z2 := MakeWires(len(a))
	if err := NewKaratsubaMultiplier(compiler, aHigh, bHigh, z2); err != nil {
		return err
	}

	sub1 := MakeWires(len(a))
	if err := NewSubtractor(compiler, z1, z2, sub1); err != nil {
		return err
	}
	sub2 := MakeWires(len(a))
	if err := NewSubtractor(compiler, sub1, z0, sub2); err != nil {
		return err
	}

	shift1 := compiler.ShiftLeft(z2, len(r), mid*2)
	shift2 := compiler.ShiftLeft(sub2, len(r), mid)

	add1 := MakeWires(len(r))
	if err := NewAdder(compiler, shift1, shift2, add1); err != nil {
		return err
	}

	return NewAdder(compiler, add1, z0, r)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
