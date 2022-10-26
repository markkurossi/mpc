//
// Copyright (c) 2022 Markku Rossi
//
// All rights reserved.
//

package bmr

import (
	"fmt"
	"testing"
)

func TestFx(t *testing.T) {
	f, err := NewF(2048)
	if err != nil {
		t.Fatalf("NewF failed: %v", err)
	}

	var a, b uint

	for a = 0; a < 2; a++ {
		for b = 0; b < 2; b++ {
			c, d, err := f.X(a, b)
			if err != nil {
				t.Fatalf("Fx(%v, %v) failed: %v", a, b, err)
			}

			fmt.Printf("Fx(%v,%v)=%v,%v: XOR(%v,%v)=%v, %v*%v=%v => %v\n",
				a, b, c, d, c, d, c^d, a, b, a*b,
				c^d == a*b)
		}
	}
}

func TestFxn(t *testing.T) {
	f, err := NewF(2048)
	if err != nil {
		t.Fatalf("NewF faield: %v", err)
	}

	s, err := NewLabel()
	if err != nil {
		t.Fatalf("NewLabel failed: %s", err)
	}

	testFxn(t, f, s, 0)
	testFxn(t, f, s, 1)
}

func testFxn(t *testing.T, f *F, s Label, bit uint) {

	cArr, dArr, err := f.XK(s[:], bit)
	if err != nil {
		t.Fatalf("Fxk(%v, %v) failed: %v", s, 0, err)
	}
	if len(cArr) != 1 {
		t.Fatalf("Postcondition failed: len(c) = %v != 1", len(cArr))
	}
	var c uint
	if cArr[0] != 0 {
		c = 1
	}

	d, err := NewLabelFromData(dArr)
	if err != nil {
		t.Fatalf("Postcondition failed: NewLabelFromData(%v): %v", dArr, err)
	}

	fmt.Printf("%v⊕%v=", d, c)
	d.BitXor(c)
	fmt.Printf("%v\n", d)

	fmt.Printf("%v⋅%v=", s, bit)
	s.Mult(bit)
	fmt.Printf("%v\n", s)

	if !d.Equal(s) {
		t.Errorf("c⊕d=%v != s⋅b=%v", d, s)
	}
}
