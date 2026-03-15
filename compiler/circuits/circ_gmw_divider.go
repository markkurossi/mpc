//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"math/bits"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/types"
)

// m: ROM address+output width. 8 gives 7 bits initial precision.
const romBits = 8

// NewUDividerGoldschmidtFast creates a Goldschmidt divider minimizing
// the circuit AND depth and size.
func NewUDividerGoldschmidtFast(cc *Compiler, a, b, qFinal, rFinal []*Wire) error {
	n := len(a)

	useROM := n >= 4
	m := romBits
	if m >= n {
		m = n - 1
	}

	// 1. Detect MSB of b to determine shift amount.
	msb := DetectMSB(cc, b)

	// 2. Generate Logarithmic Shift Signals.
	// We calculate the binary shift amount 's' needed to move b's MSB to n-1.
	shiftBits := bits.Len(uint(n - 1))
	s := make([]*Wire, shiftBits)
	for bIdx := 0; bIdx < shiftBits; bIdx++ {
		acc := cc.ZeroWire()
		for i := 0; i < n; i++ {
			shiftDist := (n - 1) - i
			if (shiftDist>>bIdx)&1 == 1 {
				tmp := cc.Calloc.Wire()
				cc.OR(acc, msb[i], tmp)
				acc = tmp
			}
		}
		s[bIdx] = acc
	}

	// 3. Normalize both a and b using the SAME shift signals.
	// This preserves the ratio a/b while aligning b for the ROM.
	bNorm := ApplyLogShifter(cc, b, s, n)

	aPadded := make([]*Wire, 2*n)
	for i := 0; i < 2*n; i++ {
		if i < n {
			aPadded[i] = a[i]
		} else {
			aPadded[i] = cc.ZeroWire()
		}
	}
	aNorm2n := ApplyLogShifter(cc, aPadded, s, 2*n)

	W := n + 1
	twoConst := make([]*Wire, W)
	for i := range twoConst {
		twoConst[i] = cc.ZeroWire()
	}
	twoConst[n] = cc.OneWire()

	var bCurr []*Wire
	var qCurr []*Wire
	qWidth := 2 * n
	var iters int

	if useROM {
		recip := NewReciprocalROM(cc, bNorm, m)

		bNormW := cc.Pad(bNorm, W)
		bSeedProd := cc.Calloc.Wires(types.Size(W + m))
		NewWallaceMultiplier(cc, bNormW, recip, bSeedProd)
		bCurr = bSeedProd[m-1 : m-1+W]

		qSeedProd := cc.Calloc.Wires(types.Size(qWidth + m))
		NewWallaceMultiplier(cc, aNorm2n, recip, qSeedProd)
		qCurr = qSeedProd[m-1 : m-1+qWidth]

		iters = iterationsForWidthWithSeed(n, m)
	} else {
		bCurr = cc.Pad(bNorm, W)
		qCurr = aNorm2n
		iters = iterationsForWidth(n)
	}

	// Goldschmidt iterations.
	for i := 0; i < iters; i++ {
		f := cc.Calloc.Wires(types.Size(W))
		NewKoggeStoneSubtractor(cc, twoConst, bCurr, f)
		fN := f[:n]

		bProd := cc.Calloc.Wires(types.Size(2 * W))
		NewWallaceMultiplier(cc, bCurr, fN, bProd)
		bCurr = bProd[n-1 : n-1+W]

		qProd := cc.Calloc.Wires(types.Size(qWidth + n))
		NewWallaceMultiplier(cc, qCurr, fN, qProd)
		qCurr = qProd[n-1 : n-1+qWidth]
	}

	// 4. Parallelized Correction Logic.
	q := qCurr[n-1 : 2*n-1]
	qbLong := cc.Calloc.Wires(types.Size(2 * n))
	NewWallaceMultiplier(cc, q, b, qbLong)
	qb := qbLong[:n]

	// r = a - qb (using n+1 bits to detect underflow via sign bit)
	r := cc.Calloc.Wires(types.Size(n + 1))
	NewKoggeStoneSubtractor(cc, a, qb, r)

	qMinus1 := SubConstOne(cc, q)
	qPlus1 := AddConstOne(cc, q)

	rPlusB := cc.Calloc.Wires(types.Size(n))
	NewKoggeStoneAdder(cc, r[:n], b, rPlusB)

	rMinusB := cc.Calloc.Wires(types.Size(n + 1))
	NewKoggeStoneSubtractor(cc, r[:n], b, rMinusB)

	isNeg := r[n]
	isGe := cc.Calloc.Wire()
	cc.INV(rMinusB[n], isGe)

	// Selection MUXes: Parallelize the correction choice
	qHigh := cc.Calloc.Wires(types.Size(n))
	rHigh := cc.Calloc.Wires(types.Size(n))
	NewMUX(cc, []*Wire{isGe}, qPlus1, q, qHigh)
	NewMUX(cc, []*Wire{isGe}, rMinusB[:n], r[:n], rHigh)

	NewMUX(cc, []*Wire{isNeg}, qMinus1, qHigh, qFinal)
	NewMUX(cc, []*Wire{isNeg}, rPlusB, rHigh, rFinal)

	return nil
}

// ApplyLogShifter implements an O(n log n) barrel shifter.
func ApplyLogShifter(cc *Compiler, in []*Wire, s []*Wire, width int) []*Wire {
	current := in
	for b := 0; b < len(s); b++ {
		shiftAmount := 1 << b
		next := make([]*Wire, width)
		for i := 0; i < width; i++ {
			var shiftedVal *Wire
			if i-shiftAmount >= 0 {
				shiftedVal = current[i-shiftAmount]
			} else {
				shiftedVal = cc.ZeroWire()
			}
			out := cc.Calloc.Wire()
			NewMUX(cc, []*Wire{s[b]}, []*Wire{shiftedVal}, []*Wire{current[i]},
				[]*Wire{out})
			next[i] = out
		}
		current = next
	}
	return current
}

// iterationsForWidth returns iterations needed with no seed (plain
// Goldschmidt).  Worst-case ε₀=0.5; after k iterations error =
// 2^(-2^k) < 2^(-n) requires 2^k ≥ n, so k = ceil(log2(n)), plus 1
// for truncation bias.
func iterationsForWidth(n int) int {
	if n <= 1 {
		return 1
	}
	return bits.Len(uint(n-1)) + 1
}

// DetectMSB detects the most significant bit of a.
func DetectMSB(cc *Compiler, a []*Wire) []*Wire {
	n := len(a)
	msb := make([]*Wire, n)
	seen := cc.ZeroWire()
	for i := n - 1; i >= 0; i-- {
		notSeen := cc.Calloc.Wire()
		cc.INV(seen, notSeen)
		msb[i] = cc.Calloc.Wire()
		cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, a[i], notSeen, msb[i]))
		tmp := cc.Calloc.Wire()
		cc.OR(seen, a[i], tmp)
		seen = tmp
	}
	return msb
}

// AddConstOne adds 1 to the argument a.
func AddConstOne(cc *Compiler, a []*Wire) []*Wire {
	one := []*Wire{cc.OneWire()}
	out := cc.Calloc.Wires(types.Size(len(a)))
	NewKoggeStoneAdder(cc, a, one, out)
	return out
}

// SubConstOne subtracts 1 from the argument a.
func SubConstOne(cc *Compiler, a []*Wire) []*Wire {
	one := []*Wire{cc.OneWire()}
	out := cc.Calloc.Wires(types.Size(len(a)))
	NewKoggeStoneSubtractor(cc, a, one, out)
	return out
}

// NewReciprocalROM returns an m-bit approximation of 1/bNorm_real in
// Q.(m-1) scale, where bNorm is an n-bit value with its MSB always at
// position n-1 (i.e. bNorm ∈ [2^(n-1), 2^n), as produced by
// ShiftToTop).
//
// The result recip satisfies:
//
//	|recip/2^(m-1) - 1/bNorm_real| < 2^(-(m-1))
//
// i.e. the reciprocal is correct to m-1 fractional bits.
//
// Table entry for address i (i ∈ [0, 2^(m-1))):
//
//	table[i] = floor(2^(2m-2) / (2^(m-1) + i))
//
// Address bits: bNorm[n-m .. n-2]  (the m-1 bits just below the always-1 MSB).
// MUX tree depth: m-1 levels, (2^(m-1)-1) × m 1-bit MUXes total.
// Gate cost for m=8: 127 × 8 × 1 AND = 1016 AND gates.
//
// Verified for m=8:
//
//	i=0:   floor(16384/128) = 128 → 128/128 = 1.000 (recip of 1.000) ✓
//	i=64:  floor(16384/192) = 85  → 85/128  ≈ 0.664 (recip of 1.500 ≈ 0.667) ✓
//	i=127: floor(16384/255) = 64  → 64/128  = 0.500 (recip of ~1.992 ≈ 0.502) ✓
func NewReciprocalROM(cc *Compiler, bNorm []*Wire, m int) []*Wire {

	n := len(bNorm)

	if m < 2 {
		panic("NewReciprocalROM: m must be at least 2")
	}
	if m > n {
		panic("NewReciprocalROM: m cannot exceed n")
	}

	numEntries := 1 << (m - 1) // 2^(m-1)
	half := numEntries         // 2^(m-1), the implicit leading 1 of bNorm_top

	// Build table as constant wire slices.
	// table[i] = floor(2^(2m-2) / (half+i)), encoded as m-bit little-endian
	// ZeroWire/OneWire constants.

	table := make([][]*Wire, numEntries)
	for i := 0; i < numEntries; i++ {
		val := (half * half) / (half + i)
		table[i] = intToConstWires(cc, val, m)
	}

	// MUX tree.
	// Level k uses address bit bNorm[n-m+k] to select between pairs.
	// After m-1 levels exactly one m-bit vector remains.
	//
	// NewMUX(cc, cond, t, f, out): cond[0]=1 → t,  cond[0]=0 → f
	// So odd-indexed entry is t (selected when address bit=1),
	//    even-indexed entry is f (selected when address bit=0).

	current := table

	for level := 0; level < m-1; level++ {

		sel := []*Wire{bNorm[n-m+level]} // single-bit cond slice

		next := make([][]*Wire, len(current)/2)
		for pair := 0; pair < len(current)/2; pair++ {
			even := current[pair*2]  // selected when sel=0
			odd := current[pair*2+1] // selected when sel=1

			out := cc.Calloc.Wires(types.Size(m))
			NewMUX(cc, sel, odd, even, out) // cond=1→odd, cond=0→even
			next[pair] = out
		}
		current = next
	}

	return current[0]
}

// intToConstWires converts an integer value to an m-bit little-endian wire
// vector of ZeroWire/OneWire constants (bit 0 = LSB).
func intToConstWires(cc *Compiler, val, m int) []*Wire {
	wires := make([]*Wire, m)
	for i := 0; i < m; i++ {
		if (val>>i)&1 == 1 {
			wires[i] = cc.OneWire()
		} else {
			wires[i] = cc.ZeroWire()
		}
	}
	return wires
}

// iterationsForWidthWithSeed returns the number of Goldschmidt iterations
// needed after a ROM seed that provides m-1 correct bits.
// Each iteration doubles the number of correct bits.
// We need (m-1) × 2^k ≥ n, so k = ceil(log2(n/(m-1))).
// Add 1 for accumulated truncation bias from >> (n-1) rounding each step.
func iterationsForWidthWithSeed(n, m int) int {
	if n <= 1 {
		return 1
	}
	correctBits := m - 1
	if correctBits < 1 {
		correctBits = 1
	}
	// ceil(n / correctBits)
	needed := (n + correctBits - 1) / correctBits
	if needed <= 1 {
		return 2 // minimum 2: 1 may be too few given truncation bias
	}
	return bits.Len(uint(needed-1)) + 1
}
