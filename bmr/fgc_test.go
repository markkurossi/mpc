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
	var a, b uint

	for a = 0; a < 2; a++ {
		for b = 0; b < 2; b++ {
			c, d, err := Fx(a, b)
			if err != nil {
				t.Fatalf("Fx(%v, %v) failed: %v", a, b, err)
			}

			fmt.Printf("Fx(%v,%v)=%v,%v: XOR(%v,%v)=%v, %v*%v=%v => %v\n",
				a, b, c, d, c, d, c^d, a, b, a*b,
				c^d == a*b)
		}
	}
}
