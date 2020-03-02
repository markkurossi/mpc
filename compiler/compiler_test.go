//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"testing"
)

type IteratorTest struct {
	Name    string
	Operand string
	Bits    int
	Eval    func(a uint64, b uint64) uint64
	Code    string
}

var iteratorTests = []IteratorTest{
	IteratorTest{
		Name:    "Add",
		Operand: "+",
		Bits:    2,
		Eval: func(a uint64, b uint64) uint64 {
			return a + b
		},
		Code: `
package main
func main(a, b int3) int3 {
    return a + b
}
`,
	},
	// 1-bit, 2-bit, and n-bit multipliers have a bit different wiring.
	IteratorTest{
		Name:    "Multiply 1-bit",
		Operand: "*",
		Bits:    1,
		Eval: func(a uint64, b uint64) uint64 {
			return a * b
		},
		Code: `
package main
func main(a, b int1) int1 {
    return a * b
}
`,
	},
	IteratorTest{
		Name:    "Multiply 2-bits",
		Operand: "*",
		Bits:    2,
		Eval: func(a uint64, b uint64) uint64 {
			return a * b
		},
		Code: `
package main
func main(a, b int4) int4 {
    return a * b
}
`,
	},
	IteratorTest{
		Name:    "Multiply n-bits",
		Operand: "*",
		Bits:    2,
		Eval: func(a uint64, b uint64) uint64 {
			return a * b
		},
		Code: `
package main
func main(a, b int6) int6 {
    return a * b
}
`,
	},
}

func TestIterator(t *testing.T) {
	for _, test := range iteratorTests {
		circ, err := Compile(test.Code, &Params{})
		if err != nil {
			t.Fatalf("Failed to compile test %s: %s", test.Name, err)
		}

		limit := uint64(1 << test.Bits)

		var g, e uint64

		for g = 0; g < limit; g++ {
			for e = 0; e < limit; e++ {

				results, err := circ.Compute([]uint64{g}, []uint64{e})
				if err != nil {
					t.Fatalf("compute failed: %s\n", err)
				}

				expected := test.Eval(g, e)

				if expected != results[0] {
					t.Errorf("%s failed: %d %s %d = %d, expected %d\n",
						test.Name, g, test.Operand, e, results[0],
						expected)
				}
			}
		}
	}
}

type FixedTest struct {
	N1   uint64
	N2   uint64
	N3   uint64
	Zero bool
	Code string
}

var fixedTests = []FixedTest{
	FixedTest{
		N1: 5,
		N2: 3,
		N3: 5,
		Code: `
package main
func main(a, b int) int {
    if a > b {
        return a
    }
    return b
}
`,
	},
	FixedTest{
		N1:   5,
		N2:   3,
		N3:   6,
		Zero: true,
		Code: `
package main
func main(a, b int) int {
    if a > b {
        return add1(a)
    }
    return add1(b)
}
func add1(val int) int {
    return val + 1
}
`,
	},
	FixedTest{
		N1:   5,
		N2:   3,
		N3:   7,
		Zero: true,
		Code: `
package main
func main(a, b int) int {
    if a > b {
        return add2(a)
    }
    return add2(b)
}
func add1(val int) int {
    return val + 1
}
func add2(val int) int {
    return add1(add1(val))
}
`,
	},
}

func TestFixed(t *testing.T) {
	for _, test := range fixedTests {
		circ, err := Compile(test.Code, &Params{})
		if err != nil {
			t.Fatalf("Failed to compile test: %s", err)
		}
		var n1 = []uint64{test.N1}
		if test.Zero {
			n1 = append(n1, 0)
		}
		results, err := circ.Compute(n1, []uint64{test.N2})
		if err != nil {
			t.Fatalf("compute failed: %s", err)
		}
		if results[0] != test.N3 {
			t.Errorf("test failed: got %d, expected %d", results[0], test.N3)
		}
	}
}
