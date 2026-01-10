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

func TestCLMULBasis(t *testing.T) {
	a := Block{Lo: 1, Hi: 0}
	b := Block{Lo: 2, Hi: 0}

	lo, hi := mul128CLMUL(a, b)

	if lo.Lo != 2 || lo.Hi != 0 || hi != (Block{}) {
		t.Fatalf("bad: lo=%v hi=%v", lo, hi)
	}
}

func TestMul128Basic(t *testing.T) {
	zero := Block{0, 0}
	one := Block{1, 0}

	// 0 * x = 0
	lo, hi := mul128(zero, Block{0xdeadbeef, 0x12345678})
	if lo != zero || hi != zero {
		t.Fatal("0*x != 0")
	}

	// 1 * x = x
	x := Block{0xabcdef, 0x1234}
	lo, hi = mul128(one, x)
	if lo != x || hi != zero {
		t.Fatal("1*x != x")
	}

	// x * x = x^2
	a := Block{2, 0} // polynomial x
	lo, hi = mul128(a, a)
	if lo.Lo != 4 || lo.Hi != 0 || hi != zero {
		t.Fatal("x*x != x^2")
	}
}

func TestMul128Cross(t *testing.T) {
	// x^63 * x^63 = x^126
	a := Block{Lo: 1 << 63, Hi: 0}

	lo, hi := mul128(a, a)

	expLo := Block{Hi: 1 << 62} // 126 = 64 + 62
	expHi := Block{}

	if lo != expLo || hi != expHi {
		t.Fatalf("got lo=%v hi=%v, expected lo=%v hi=%v", lo, hi, expLo, expHi)
	}
}

func TestMul128AllOnes(t *testing.T) {
	a := Block{^uint64(0), ^uint64(0)}
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
		a := Block{rng.Uint64(), rng.Uint64()}
		b := Block{rng.Uint64(), rng.Uint64()}

		lo1, hi1 := mul128(a, b)
		lo2, hi2 := mul128Ref(a, b)

		if lo1 != lo2 || hi1 != hi2 {
			t.Fatalf("mismatch on %v * %v", a, b)
		}
	}
}

func TestMul128CLMULMatchesGeneric(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	for i := 0; i < 10000; i++ {
		a := Block{rng.Uint64(), rng.Uint64()}
		b := Block{rng.Uint64(), rng.Uint64()}

		lo1, hi1 := mul128CLMUL(a, b)
		lo2, hi2 := mul128Generic(a, b)

		if lo1 != lo2 || hi1 != hi2 {
			t.Fatalf("a=%v b=%v\nCLMUL lo=%v hi=%v\nGEN   lo=%v hi=%v",
				a, b, lo1, hi1, lo2, hi2)
		}
	}
}

// func TestMul128CLMULMatchesGeneric(t *testing.T) {
// 	rng := rand.New(rand.NewSource(2))
// 	for i := 0; i < 1000; i++ {
// 		a := Block{rng.Uint64(), rng.Uint64()}
// 		b := Block{rng.Uint64(), rng.Uint64()}
//
// 		lo1, hi1 := mul128(a, b)
// 		lo2, hi2 := mul128Generic(a, b)
//
// 		if lo1 != lo2 || hi1 != hi2 {
// 			t.Fatal("CLMUL != generic")
// 		}
// 	}
// }

func BenchmarkMul128(b *testing.B) {

	var b0, b1 Block

	for b.Loop() {
		lo, hi := mul128Generic(b0, b1)
		_ = lo
		_ = hi
	}
}
