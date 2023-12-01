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

type intTest struct {
	a int64
	b int64
	r int64
}

var addTests = []intTest{
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
	for idx, test := range addTests {
		a := NewInt(test.a)
		b := NewInt(test.b)
		r := NewInt(0).Add(a, b)
		if r.Int64() != test.r {
			t.Errorf("add%v: %v+%v=%v, expected %v\n",
				idx, test.a, test.b, r.Int64(), test.r)
		}
	}
}

var andTests = []intTest{
	{
		a: 0x0000ffff,
		b: 0x00001111,
		r: 0x00001111,
	},
	{
		a: 0x0000ffff,
		b: 0x00000000,
		r: 0x00000000,
	},
}

func TestIntAnd(t *testing.T) {
	for _, test := range andTests {
		a := NewInt(test.a)
		b := NewInt(test.b)
		r := NewInt(0).And(a, b)
		if r.Int64() != test.r {
			t.Errorf("%v&%v=%v, expected %v\n",
				test.a, test.b, r.Int64(), test.r)
		}
	}
}

func TestIntCmp(t *testing.T) {
	a := NewInt(1)
	b := NewInt(1)
	a.Sub(a, b)
	a.Sub(a, b)

	c := NewInt(0)
	cmp := a.Cmp(c)
	if cmp >= 0 {
		t.Errorf("%v.Cmp(%v)=%v\n", a.Int64(), c.Int64(), cmp)
	}
}

var divTests = []intTest{
	{
		a: 0x0000ffff,
		b: 0x00001111,
		r: 0x0000000f,
	},
	{
		a: 0x0000ffff,
		b: 0x00000000,
		r: -1,
	},
	{
		a: 10,
		b: 2,
		r: 5,
	},
}

func TestIntDiv(t *testing.T) {
	for _, test := range divTests {
		a := NewInt(test.a)
		b := NewInt(test.b)
		r := NewInt(0).Div(a, b)
		if r.Int64() != test.r {
			t.Errorf("%v/%v=%v, expected %v\n",
				test.a, test.b, r.Int64(), test.r)
		}
	}
}

var lshTests = []intTest{
	{
		a: 0x0000ffff,
		b: 1,
		r: 0x0001fffe,
	},
	{
		a: 0x0000ffff,
		b: 0x00000000,
		r: 0x0000ffff,
	},
	{
		a: 10,
		b: 2,
		r: 40,
	},
	{
		a: 1,
		b: 63,
		r: -9223372036854775808,
	},
}

func TestIntLsh(t *testing.T) {
	for _, test := range lshTests {
		a := NewInt(test.a)
		r := NewInt(0).Lsh(a, uint(test.b))
		if r.Int64() != test.r {
			t.Errorf("%v<<%v=%v(%x), expected %v\n",
				test.a, test.b, r.Int64(), r.Int64(), test.r)
		}
	}
}

var modTests = []intTest{
	{
		a: 0x0000ffff,
		b: 0x00001111,
		r: 0,
	},
	{
		a: 0x0000ffff,
		b: 0x00000000,
		r: 0x0000ffff,
	},
	{
		a: 10,
		b: 2,
		r: 0,
	},
}

func TestIntMod(t *testing.T) {
	for _, test := range modTests {
		a := NewInt(test.a)
		b := NewInt(test.b)
		r := NewInt(0).Mod(a, b)
		if r.Int64() != test.r {
			t.Errorf("%v%%%v=%v, expected %v\n",
				test.a, test.b, r.Int64(), test.r)
		}
	}
}

var mulTests = []intTest{
	{
		a: 0x0000ffff,
		b: 0x00001111,
		r: 0x1110eeef,
	},
	{
		a: 0x0000ffff,
		b: 0x00000000,
		r: 0x00000000,
	},
	{
		a: 10,
		b: 2,
		r: 20,
	},
	{
		a: 0x7fffffffffffffff,
		b: 2,
		r: -2,
	},
	{
		a: 0x7fffffffffffffff,
		b: 0x00000000ffffffff,
		r: 0x7fffffff00000001,
	},
}

func TestIntMul(t *testing.T) {
	for _, test := range mulTests {
		a := NewInt(test.a)
		b := NewInt(test.b)
		r := NewInt(0).Mul(a, b)
		if r.Int64() != test.r {
			t.Errorf("%v*%v=%v, expected %v\n",
				test.a, test.b, r.Int64(), test.r)
		}
	}
}

var orTests = []intTest{
	{
		a: 0x0000ffff,
		b: 0x00001111,
		r: 0x0000ffff,
	},
	{
		a: 0x0000ffff,
		b: 0x00000000,
		r: 0x0000ffff,
	},
	{
		a: 10,
		b: 2,
		r: 10,
	},
	{
		a: 0x7fffffffffffffff,
		b: 2,
		r: 0x7fffffffffffffff,
	},
	{
		a: 0x0555555555555555,
		b: 0x0aaaaaaaaaaaaaaa,
		r: 0x0fffffffffffffff,
	},
}

func TestIntOr(t *testing.T) {
	for _, test := range orTests {
		a := NewInt(test.a)
		b := NewInt(test.b)
		r := NewInt(0).Or(a, b)
		if r.Int64() != test.r {
			t.Errorf("%v|%v=%v, expected %x\n",
				test.a, test.b, r.Int64(), test.r)
		}
	}
}

var rshTests = []intTest{
	{
		a: 0x0000ffff,
		b: 1,
		r: 0x00007fff,
	},
	{
		a: 0x0000ffff,
		b: 0x00000000,
		r: 0x0000ffff,
	},
	{
		a: 10,
		b: 2,
		r: 2,
	},
	{
		a: 1,
		b: 63,
		r: 0,
	},
}

func TestIntRsh(t *testing.T) {
	for _, test := range rshTests {
		a := NewInt(test.a)
		r := NewInt(0).Rsh(a, uint(test.b))
		if r.Int64() != test.r {
			t.Errorf("%v>>%v=%v(%x), expected %v\n",
				test.a, test.b, r.Int64(), r.Int64(), test.r)
		}
	}
}

func TestIntSetString(t *testing.T) {
	i, ok := NewInt(0).SetString("0xdeadbeef", 0)
	if !ok {
		t.Fatalf("SetString failed")
	}
	_ = i
}

var subTests = []intTest{
	{
		a: 0x00010000,
		b: 0x00000001,
		r: 0x0000ffff,
	},
	{
		a: 0x0000ffff,
		b: -1,
		r: 0x00010000,
	},
	{
		a: math.MaxInt64,
		b: -1,
		r: math.MinInt64,
	},
	{
		a: math.MinInt64,
		b: 1,
		r: math.MaxInt64,
	},
	{
		a: math.MaxInt64,
		b: 1,
		r: math.MaxInt64 - 1,
	},
	{
		a: 0,
		b: 5,
		r: -5,
	},
}

func TestIntSub(t *testing.T) {
	for idx, test := range subTests {
		a := NewInt(test.a)
		b := NewInt(test.b)
		r := NewInt(0).Sub(a, b)
		if r.Int64() != test.r {
			t.Errorf("test-%v: %v-%v=%v, expected %v\n",
				idx, test.a, test.b, r.Int64(), test.r)
		}
	}
}

func TestIntSubNegative(t *testing.T) {
	val := NewInt(5)
	r := NewInt(0)
	r.Sub(r, val)

	add := NewInt(10)
	r.Add(r, add)
	result := r.Int64()
	if result != 5 {
		t.Errorf("TestIntSubNegative: +10 failed, got %v", result)
	}
}

var xorTests = []intTest{
	{
		a: 0x0000ffff,
		b: 0x00001111,
		r: 0x0000eeee,
	},
	{
		a: 0x0000ffff,
		b: 0x00000000,
		r: 0x0000ffff,
	},
	{
		a: 10,
		b: 2,
		r: 8,
	},
	{
		a: 0x7fffffffffffffff,
		b: 2,
		r: 0x7ffffffffffffffd,
	},
	{
		a: 0x0555555555555555,
		b: 0x0aaaaaaaaaaaaaaa,
		r: 0x0fffffffffffffff,
	},
}

func TestIntXor(t *testing.T) {
	for _, test := range xorTests {
		a := NewInt(test.a)
		b := NewInt(test.b)
		r := NewInt(0).Xor(a, b)
		if r.Int64() != test.r {
			t.Errorf("%v^%v=%v, expected %x\n",
				test.a, test.b, r.Int64(), test.r)
		}
	}
}
