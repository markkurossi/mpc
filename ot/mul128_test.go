//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"math/rand"
	"testing"
)

func TestMul128Basic(t *testing.T) {
	zero := Label{0, 0}
	one := Label{1, 0}

	// 0 * x = 0
	lo, hi := mul128(zero, Label{0xdeadbeef, 0x12345678})
	if lo != zero || hi != zero {
		t.Fatal("0*x != 0")
	}

	// 1 * x = x
	x := Label{0xabcdef, 0x1234}
	lo, hi = mul128(one, x)
	if lo != x || hi != zero {
		t.Fatal("1*x != x")
	}

	// x * x = x^2
	a := Label{2, 0} // polynomial x
	lo, hi = mul128(a, a)
	if lo.D0 != 4 || lo.D1 != 0 || hi != zero {
		t.Fatal("x*x != x^2")
	}
}

func TestMul128Cross(t *testing.T) {
	// x^63 * x^63 = x^126
	a := Label{D0: 1 << 63, D1: 0}

	lo, hi := mul128(a, a)

	expLo := Label{D1: 1 << 62} // 126 = 64 + 62
	expHi := Label{}

	if lo != expLo || hi != expHi {
		t.Fatalf("got lo=%v hi=%v, expected lo=%v hi=%v", lo, hi, expLo, expHi)
	}
}

func TestMul128AllOnes(t *testing.T) {
	a := Label{^uint64(0), ^uint64(0)}
	b := a

	lo1, hi1 := mul128(a, b)
	lo2, hi2 := mul128Ref(a, b)

	if lo1 != lo2 || hi1 != hi2 {
		t.Fatal("all-ones mismatch")
	}
}

func TestMul128Random(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 1000; i++ {
		a := Label{rng.Uint64(), rng.Uint64()}
		b := Label{rng.Uint64(), rng.Uint64()}

		lo1, hi1 := mul128(a, b)
		lo2, hi2 := mul128Ref(a, b)

		if lo1 != lo2 || hi1 != hi2 {
			t.Fatalf("mismatch on %v * %v", a, b)
		}
	}
}

func BenchmarkMul128(b *testing.B) {
	var b0, b1 Label

	for b.Loop() {
		lo, hi := mul128Generic(b0, b1)
		_ = lo
		_ = hi
	}
}
