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
