//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"testing"
)

type CompilerTest struct {
	Name    string
	Heavy   bool
	Operand string
	Bits    int
	Eval    func(a uint64, b uint64) uint64
	Code    string
}

var compilerTests = []CompilerTest{
	CompilerTest{
		Name:    "Add",
		Heavy:   true,
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
	CompilerTest{
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
	CompilerTest{
		Name:    "Multiply 2-bits",
		Heavy:   true,
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
	CompilerTest{
		Name:    "Multiply n-bits",
		Heavy:   true,
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

func TestCompiler(t *testing.T) {
	for _, test := range compilerTests {
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
