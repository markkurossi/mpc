//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.
//

package mpa

import (
	"math"
	"testing"
)

var addTests = []struct {
	a int64
	b int64
	r int64
}{
	{
		a: 0x0000ffff,
		b: 0x00000001,
		r: 0x00010000,
	},
	{
		a: 0x0000ffff,
		b: -1,
		r: 0x0000fffe,
	},
	{
		a: math.MaxInt64,
		b: 1,
		r: math.MinInt64,
	},
	{
		a: math.MinInt64,
		b: -1,
		r: math.MaxInt64,
	},
	{
		a: math.MinInt64,
		b: 1,
		r: math.MinInt64 + 1,
	},
}

func TestIntAdd(t *testing.T) {
	for _, test := range addTests {
		a := NewInt(test.a)
		b := NewInt(test.b)
		r := NewInt(0).Add(a, b)
		if r.Int64() != test.r {
			t.Errorf("%v+%v=%v, expected %v\n",
				test.a, test.b, r.Int64(), test.r)
		}
	}
}
