//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

//go:build amd64 && gc

package ot

import (
	"testing"
)

func BenchmarkMul128AMD64(b *testing.B) {

	var b0, b1 Label

	for b.Loop() {
		lo, hi := mul128(b0, b1)
		_ = lo
		_ = hi
	}
}
