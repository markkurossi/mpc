//
// Copyright (c) 2019-2026 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/rand"
	"fmt"
	"strings"
	"testing"
)

// buildANDChain returns a circuit of n chained AND gates over two input bits:
// gate i computes AND(wire i, wire i+1) -> wire i+2. It exercises the per-gate
// garbled-table allocation path without depending on the compiler.
func buildANDChain(n int) *Circuit {
	var b strings.Builder
	fmt.Fprintf(&b, "%d %d\n", n, n+2)
	b.WriteString("2 1 1\n")
	b.WriteString("1 1\n\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "2 1 %d %d %d AND\n", i, i+1, i+2)
	}
	c, err := ParseBristol(strings.NewReader(b.String()))
	if err != nil {
		panic(err)
	}
	return c
}

var benchKey = []byte("0123456789abcdef")

// BenchmarkGarble measures a single Garble with no reuse. Runnable on any
// version, so it is the apples-to-apples baseline for the slab change.
func BenchmarkGarble(b *testing.B) {
	c := buildANDChain(10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g, err := c.Garble(rand.Reader, benchKey)
		if err != nil {
			b.Fatal(err)
		}
		_ = g
	}
}

// BenchmarkGarbleReuse measures Garble with Release, the intended steady-state
// usage where scratch is recycled across calls.
func BenchmarkGarbleReuse(b *testing.B) {
	c := buildANDChain(10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g, err := c.Garble(rand.Reader, benchKey)
		if err != nil {
			b.Fatal(err)
		}
		g.Release()
	}
}
