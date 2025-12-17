//
// Copyright (c) 2025 Markku Rossi
//
// All rights reserved.

package otext

import (
	"testing"
)

func BenchmarkPRGAESCTR1K(b *testing.B) {
	benchmarkPRGAESCTR(b, 1000)
}

func BenchmarkPRGAESCTR10K(b *testing.B) {
	benchmarkPRGAESCTR(b, 10000)
}

func BenchmarkPRGAESCTR100K(b *testing.B) {
	benchmarkPRGAESCTR(b, 100000)
}

func benchmarkPRGAESCTR(b *testing.B, n int) {
	var key [16]byte
	out := make([]byte, n)

	for b.Loop() {
		prgAESCTR(key[:], out)
	}
}
