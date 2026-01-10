//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

//go:build amd64 && gc

package ot

import (
	"math/rand"
	"testing"
)

func TestCLMULBasis(t *testing.T) {
	a := Label{D0: 1, D1: 0}
	b := Label{D0: 2, D1: 0}

	var lo, hi Label
	mul128CLMUL(&a, &b, &lo, &hi)

	if lo.D0 != 2 || lo.D1 != 0 || hi != (Label{}) {
		t.Fatalf("bad: lo=%v hi=%v", lo, hi)
	}
}

func TestMul128CLMULMatchesGeneric(t *testing.T) {
	rng := rand.New(rand.NewSource(123))
	for i := 0; i < 10000; i++ {
		a := Label{rng.Uint64(), rng.Uint64()}
		b := Label{rng.Uint64(), rng.Uint64()}

		var lo1, hi1 Label
		mul128CLMUL(&a, &b, &lo1, &hi1)

		lo2, hi2 := mul128Generic(a, b)

		if lo1 != lo2 || hi1 != hi2 {
			t.Fatalf("a=%v b=%v\nCLMUL lo=%v hi=%v\nGEN   lo=%v hi=%v",
				a, b, lo1, hi1, lo2, hi2)
		}
	}
}

func BenchmarkMul128AMD64(b *testing.B) {

	var b0, b1 Label

	for b.Loop() {
		lo, hi := mul128(b0, b1)
		_ = lo
		_ = hi
	}
}
