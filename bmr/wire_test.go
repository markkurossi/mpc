//
// Copyright (c) 2022 Markku Rossi
//
// All rights reserved.
//

package bmr

import (
	"testing"
)

func TestLabelXor(t *testing.T) {
	a, err := NewLabel()
	if err != nil {
		t.Fatalf("NewLabel failed: %v", err)
	}
	b, err := NewLabel()
	if err != nil {
		t.Fatalf("NewLabel failed: %v", err)
	}
	c := a.Copy()
	for i := 0; i < len(c); i++ {
		c[i] ^= b[i]
	}
	a.Xor(b)
	if !a.Equal(c) {
		t.Fatalf("Label.Xor failed")
	}
}

func TestLabelBitXor(t *testing.T) {
	a, err := NewLabel()
	if err != nil {
		t.Fatalf("NewLabel failed: %v", err)
	}

	c := a.Copy()

	// BitXor(0) leaves a unchanged.
	a.BitXor(0)
	if !a.Equal(c) {
		t.Fatalf("Label.BitXor(0) failed")
	}

	// BitXor(1) swaps bits.
	a.BitXor(1)
	for i := 0; i < len(c); i++ {
		c[i] ^= 0xff
	}
	if !a.Equal(c) {
		t.Fatalf("Label.BitXor(1) failed")
	}
}

func TestLabelMult(t *testing.T) {
	a, err := NewLabel()
	if err != nil {
		t.Fatalf("NewLabel failed: %v", err)
	}

	c := a.Copy()

	// Mult(1) leaves a unchanged.
	a.Mult(1)
	if !a.Equal(c) {
		t.Fatalf("Label.Mult(1) failed")
	}

	// Mult(0) zeroes a.
	a.Mult(0)
	for i := 0; i < len(a); i++ {
		if a[i] != 0 {
			t.Fatalf("Label.Mult(0) failed")
		}
	}
}
