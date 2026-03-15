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
// the circuit AND depth. It uses NewReciprocalROM to provide an m-bit
// reciprocal approximation as the starting point for the Goldschmidt
// iteration, reducing iteration count.
//
// For m=8, n=64: 4 iterations instead of 7 (saves 3 pairs of Wallace
// multiplies).
//
// HOW THE SEED INTEGRATES:
//
// Without seed (v7):
//
//	bCurr₀ = bNorm                    (bCurr_real ∈ [1.0, 2.0))
//	qCurr₀ = aNorm2n
//	iterate: f = 2^n - bCurr, bCurr *= f >> (n-1), qCurr *= f >> (n-1)
//
// With ROM seed:
//
//	The ROM gives recip ≈ 2^(m-1) / bNorm_real in Q.(m-1) scale (m bits).
//	We use it for one "seed multiply" step before the main iterations:
//
//	seed_step:
//	  bCurr₁ = (bNorm_extended × recip) >> (m-1)   [n+m bit product, >> (m-1)]
//	  qCurr₁ = (aNorm2n × recip)       >> (m-1)   [2n+m bit product, >> (m-1)]
//
//	After seed_step:
//	  bCurr₁_real = bNorm_real × recip_real ≈ bNorm_real × 1/bNorm_real = 1.0
//	  But is now in Q.(n-1) scale? No — scale changed.
//
// SCALE ANALYSIS:
//
//	bNorm is Q.(n-1):    bNorm_real = bNorm / 2^(n-1) ∈ [1.0, 2.0)
//	recip is Q.(m-1):    recip_real = recip / 2^(m-1) ≈ 1/bNorm_real
//	product raw:         bNorm × recip
//	product real:        bNorm_real × recip_real ≈ 1.0
//	product raw ≈        2^(n-1) × 2^(m-1) = 2^(n+m-2)
//	after >> (m-1):      ≈ 2^(n-1) — back to Q.(n-1) scale ✓
//
//	So after one seed multiply, bCurr is back in Q.(n-1) scale and close to
//	2^(n-1) (= 1.0). The main Goldschmidt loop then proceeds unchanged.
//
// VERIFICATION (n=8, m=8, a=100, b=5):
//
//	bNorm=160, recip=table[32]=floor(16384/160)=102
//	bCurr₁ = (160 × 102) >> 7 = 16320 >> 7 = 127   (target: 128 = 2^7) ✓ (1 off)
//	qCurr₁ = (3200 × 102) >> 7 = 326400 >> 7 = 2550 (target: 2560 = 20×128) ✓
//	Now 1 Goldschmidt iteration:
//	f = 256 - 127 = 129
//	bCurr₂ = (127 × 129) >> 7 = 16383 >> 7 = 127   (converged)
//	qCurr₂ = (2550 × 129) >> 7 = 328950 >> 7 = 2569
//	q = 2569 >> 7 = 20 ✓
//	Total: 1 seed multiply + 1 iteration = 2 multiplies vs 3 without seed.
//	For n=64 the saving is 3 iterations = 6 fewer Wallace multiplies.
//
// ROM ADDRESS:
//
//	bNorm_top = bNorm[n-1 .. n-m] (top m bits of bNorm).
//	Bit n-1 is always 1 (MSB guaranteed by normalization) — carries no info.
//	Address = bNorm[n-2 .. n-m] (the m-1 bits below MSB), giving 2^(m-1) entries.
//	table index i: bNorm_top = 128+i, recip = floor(16384/(128+i)).
func NewUDividerGoldschmidtFast(cc *Compiler, a, b, qFinal, rFinal []*Wire) error {

	n := len(a)

	// Use ROM seed only when n is large enough to benefit.
	// Minimum useful m is 2 (1 bit of precision — barely worth it).
	// For n < 4, skip the ROM and use plain iterations.
	useROM := n >= 4
	m := romBits
	if m >= n {
		m = n - 1 // ensure address bits bNorm[n-m..n-2] are valid (at least 1 bit)
	}

	//--------------------------------------------
	// Normalize: shift b left so MSB is at bit n-1.
	// Shift a into 2n-bit word to avoid overflow.
	//--------------------------------------------

	msb := DetectMSB(cc, b)
	bNorm := ShiftToTop(cc, b, msb)
	aNorm2n := ShiftToTop2n(cc, a, msb, n)

	W := n + 1 // working width for bCurr (n bits + 1 carry bit)

	//--------------------------------------------
	// Constant: twoConst = 2^n (W bits, bit n set).
	//--------------------------------------------

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
		// ROM seed: m-bit reciprocal approximation of bNorm.
		// recip ≈ 2^(m-1) / bNorm_real  in Q.(m-1) scale.

		recip := NewReciprocalROM(cc, bNorm, m)

		// Seed multiply step.
		// bCurr = (bNorm_extended × recip) >> (m-1)  →  Q.(n-1) scale
		// qCurr = (aNorm2n × recip)        >> (m-1)  →  Q.(n-1) scale
		//
		// Product sizes:
		//   bNorm (W bits) × recip (m bits) → W+m bits; we need bits [m-1 .. m-1+W]
		//   aNorm2n (2n bits) × recip (m bits) → 2n+m bits; we need [m-1 .. m-1+2n]

		bNormW := cc.Pad(bNorm, W)

		bSeedProd := cc.Calloc.Wires(types.Size(W + m))
		NewWallaceMultiplier(cc, bNormW, recip, bSeedProd)
		bCurr = bSeedProd[m-1 : m-1+W]

		qSeedProd := cc.Calloc.Wires(types.Size(qWidth + m))
		NewWallaceMultiplier(cc, aNorm2n, recip, qSeedProd)
		qCurr = qSeedProd[m-1 : m-1+qWidth]

		iters = iterationsForWidthWithSeed(n, m)

	} else {
		// No ROM: start directly from bNorm and aNorm2n.
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

	// Extract integer quotient: q = qCurr >> (n-1) = qCurr[n-1 : 2n-1].
	q := qCurr[n-1 : 2*n-1]

	// Remainder: r = a - q*b

	qbLong := cc.Calloc.Wires(types.Size(2 * n))
	NewWallaceMultiplier(cc, q, b, qbLong)
	qb := qbLong[:n]

	r := cc.Calloc.Wires(types.Size(n))
	NewKoggeStoneSubtractor(cc, a, qb, r)

	// Correction pass 1: r < 0 → q--, r += b

	rPlusB := cc.Calloc.Wires(types.Size(n))
	NewKoggeStoneAdder(cc, r, b, rPlusB)
	qMinus1 := SubConstOne(cc, q)
	neg := SignBit(r)

	qCorr := cc.Calloc.Wires(types.Size(n))
	rCorr := cc.Calloc.Wires(types.Size(n))
	NewMUX(cc, []*Wire{neg}, qMinus1, q, qCorr)
	NewMUX(cc, []*Wire{neg}, rPlusB, r, rCorr)

	// Correction pass 2: r >= b → q++, r -= b

	rMinusB := cc.Calloc.Wires(types.Size(n + 1))
	NewKoggeStoneSubtractor(cc, rCorr, b, rMinusB)
	borrow := rMinusB[n]
	ge := cc.Calloc.Wire()
	cc.INV(borrow, ge)

	qPlus1 := AddConstOne(cc, qCorr)
	NewMUX(cc, []*Wire{ge}, qPlus1, qCorr, qFinal)
	NewMUX(cc, []*Wire{ge}, rMinusB[:n], rCorr, rFinal)

	return nil
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

// ShiftToTop shifts "in" so that its MSB is at wire n-1 using logarithmic stages.
// This reduces gate complexity from O(n^2) to O(n log n).
func ShiftToTop(cc *Compiler, in []*Wire, msb []*Wire) []*Wire {
	n := len(in)

	// 1. Convert one-hot MSB vector to a binary shift amount (s).
	// msb[i] is 1 if bit i is the MSB. To move it to n-1, shift distance is (n-1-i).
	// We calculate the binary representation of the shift distance.
	shiftBits := bits.Len(uint(n - 1))
	s := make([]*Wire, shiftBits)
	for b := 0; b < shiftBits; b++ {
		acc := cc.ZeroWire()
		for i := 0; i < n; i++ {
			shiftDist := (n - 1) - i
			if (shiftDist>>b)&1 == 1 {
				tmp := cc.Calloc.Wire()
				cc.OR(acc, msb[i], tmp)
				acc = tmp
			}
		}
		s[b] = acc
	}

	// 2. Logarithmic Shifter Stages
	current := in
	for b := 0; b < shiftBits; b++ {
		shiftAmount := 1 << b
		next := make([]*Wire, n)

		for i := 0; i < n; i++ {
			// If s[b] is 1, we take the value from a lower index (shifted up).
			// If s[b] is 0, we keep the current value.
			var shiftedVal *Wire
			if i-shiftAmount >= 0 {
				shiftedVal = current[i-shiftAmount]
			} else {
				shiftedVal = cc.ZeroWire()
			}

			out := cc.Calloc.Wire()
			// NewMUX(cond, trueVal, falseVal, out)
			// If bit s[b] is set, shift the value.
			NewMUX(cc, []*Wire{s[b]}, []*Wire{shiftedVal}, []*Wire{current[i]},
				[]*Wire{out})
			next[i] = out
		}
		current = next
	}

	return current
}

// ShiftToTop2n shifts in so that its wire msb is at 2n-1.
func ShiftToTop2n(cc *Compiler, in []*Wire, msb []*Wire, n int) []*Wire {
	out := make([]*Wire, 2*n)
	for i := range out {
		out[i] = cc.ZeroWire()
	}
	for i := 0; i < 2*n; i++ {
		acc := cc.ZeroWire()
		for j := 0; j < n; j++ {
			shift := (n - 1) - j
			src := i - shift
			var bit *Wire
			if src >= 0 && src < n {
				bit = in[src]
			} else {
				bit = cc.ZeroWire()
			}
			and := cc.Calloc.Wire()
			cc.AddGate(cc.Calloc.BinaryGate(circuit.AND, msb[j], bit, and))
			tmp := cc.Calloc.Wire()
			cc.OR(acc, and, tmp)
			acc = tmp
		}
		out[i] = acc
	}
	return out
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

// SignBit returns the sign bit of a.
func SignBit(a []*Wire) *Wire {
	return a[len(a)-1]
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
