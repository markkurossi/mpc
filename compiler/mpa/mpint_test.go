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

type int64Test struct {
	a int64
	b int64
	r int64
}

var addTests = []int64Test{
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

func TestInt64Add(t *testing.T) {
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

var andTests = []int64Test{
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

func TestInt64And(t *testing.T) {
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

func TestInt64Cmp(t *testing.T) {
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

var divTests = []int64Test{
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

var lshTests = []int64Test{
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

var modTests = []int64Test{
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

var mulTests = []int64Test{
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

var orTests = []int64Test{
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

var rshTests = []int64Test{
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

var subTests = []int64Test{
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

var xorTests = []int64Test{
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

type int32Test struct {
	a int32
	b int32
	r int64
}

var add32Tests = []int32Test{
	{
		a: math.MaxInt32,
		b: 0x00000001,
		r: math.MinInt32,
	},
	{
		a: 0x7fffffff,
		b: 0x7fffffff,
		r: -2,
	},
	{
		a: 0,
		b: -1,
		r: -1,
	},
}

func TestInt32Add(t *testing.T) {
	for idx, test := range add32Tests {
		a := NewInt(int64(test.a))
		a.SetTypeSize(32)
		b := NewInt(int64(test.b))
		b.SetTypeSize(32)
		r := NewInt(0).Add(a, b)
		if r.Int64() != test.r {
			t.Errorf("TestInt32Add-%v: %v+%v=%v, expected %v\n",
				idx, test.a, test.b, r.Int64(), test.r)
		}
	}
}

var div32Tests = []int32Test{
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

func TestInt32Div(t *testing.T) {
	for _, test := range div32Tests {
		a := NewInt(int64(test.a))
		a.SetTypeSize(32)
		b := NewInt(int64(test.b))
		b.SetTypeSize(32)
		r := NewInt(0).Div(a, b)
		if r.Int64() != test.r {
			t.Errorf("%v/%v=%v, expected %v\n",
				test.a, test.b, r.Int64(), test.r)
		}
	}
}

var lsh32Tests = []int32Test{
	{
		a: 0x0000ffff,
		b: 1,
		r: 0x0001fffe,
	},
	{
		a: 0x7fffffff,
		b: 1,
		r: -2,
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
		b: 31,
		r: -2147483648,
	},
}

func TestInt32Lsh(t *testing.T) {
	for idx, test := range lsh32Tests {
		a := NewInt(int64(test.a))
		a.SetTypeSize(32)
		r := NewInt(0).Lsh(a, uint(test.b))
		if r.Int64() != test.r {
			t.Errorf("TestInt32Lsh-%v: %v<<%v=%v(%x), expected %v\n",
				idx, test.a, test.b, r.Int64(), r.Int64(), test.r)
		}
	}
}

var mul32Tests = []int32Test{
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
		a: 0x7fffffff,
		b: 2,
		r: -2,
	},
	{
		a: 0x7fffffff,
		b: 0x0000ffff,
		r: 0x7fff0001,
	},
}

func TestInt32Mul(t *testing.T) {
	for idx, test := range mul32Tests {
		a := NewInt(int64(test.a))
		a.SetTypeSize(32)
		b := NewInt(int64(test.b))
		b.SetTypeSize(32)
		r := NewInt(0).Mul(a, b)
		if r.Int64() != test.r {
			t.Errorf("TestInt32Mul-%v: %v*%v=%v, expected %v\n",
				idx, test.a, test.b, r.Int64(), test.r)
		}
	}
}

var rsh32Tests = []int32Test{
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
		b: 31,
		r: 0,
	},
}

func TestInt32Rsh(t *testing.T) {
	for _, test := range rsh32Tests {
		a := NewInt(int64(test.a))
		a.SetTypeSize(32)
		r := NewInt(0).Rsh(a, uint(test.b))
		if r.Int64() != test.r {
			t.Errorf("%v>>%v=%v(%x), expected %v\n",
				test.a, test.b, r.Int64(), r.Int64(), test.r)
		}
	}
}

var sub32Tests = []int32Test{
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
		a: math.MaxInt32,
		b: -1,
		r: math.MinInt32,
	},
	{
		a: math.MinInt32,
		b: 1,
		r: math.MaxInt32,
	},
	{
		a: math.MaxInt32,
		b: 1,
		r: math.MaxInt32 - 1,
	},
	{
		a: 0,
		b: 5,
		r: -5,
	},
}

func TestInt32Sub(t *testing.T) {
	for idx, test := range sub32Tests {
		a := NewInt(int64(test.a))
		a.SetTypeSize(32)
		b := NewInt(int64(test.b))
		b.SetTypeSize(32)
		r := NewInt(0).Sub(a, b)
		if r.Int64() != test.r {
			t.Errorf("TestInt32Sub-%v: %v-%v=%v, expected %v\n",
				idx, test.a, test.b, r.Int64(), test.r)
		}
	}
}

func TestInt32SubMaxInt32(t *testing.T) {
	a := NewInt(int64(math.MaxInt32))
	a.SetTypeSize(32)
	b := NewInt(1)
	b.SetTypeSize(32)
	r := NewInt(0).Sub(a, b)
	if r.Int64() != int64(math.MaxInt32-1) {
		t.Errorf("%v-%v=%v, expected %v\n", a, b, r, math.MaxInt32-1)
	}
}
