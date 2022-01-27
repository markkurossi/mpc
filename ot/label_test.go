//
// label_test.go
//
// Copyright (c) 2019-2022 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"testing"
)

func BenchmarkLabelMul2(b *testing.B) {
	var l Label

	for i := 0; i < b.N; i++ {
		l.Mul2()
	}
}

func BenchmarkLabelMul4(b *testing.B) {
	var l Label

	for i := 0; i < b.N; i++ {
		l.Mul4()
	}
}

func BenchmarkLabelXor(b *testing.B) {
	var l0, l1 Label

	for i := 0; i < b.N; i++ {
		l0.Xor(l1)
	}
}

func TestLabel(t *testing.T) {
	label := &Label{
		d0: 0xffffffffffffffff,
		d1: 0xffffffffffffffff,
	}

	label.SetS(true)
	if label.d0 != 0xffffffffffffffff {
		t.Fatal("Failed to set S-bit")
	}

	label.SetS(false)
	if label.d0 != 0x7fffffffffffffff {
		t.Fatalf("Failed to clear S-bit: %x", label.d0)
	}

	label = &Label{
		d1: 0xffffffffffffffff,
	}
	label.Mul2()
	if label.d0 != 0x1 {
		t.Fatalf("Mul2 d0 failed")
	}
	if label.d1 != 0xfffffffffffffffe {
		t.Fatalf("Mul2 d1 failed: %x", label.d1)
	}

	label = &Label{
		d1: 0xffffffffffffffff,
	}
	label.Mul4()
	if label.d0 != 0x3 {
		t.Fatalf("Mul4 d0 failed")
	}
	if label.d1 != 0xfffffffffffffffc {
		t.Fatalf("Mul4 d1 failed")
	}
}
