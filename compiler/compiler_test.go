//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"math/big"
	"testing"

	"github.com/markkurossi/mpc/compiler/utils"
)

type IteratorTest struct {
	Name    string
	Operand string
	Bits    int
	Eval    func(a int64, b int64) int64
	Code    string
}

var iteratorTests = []IteratorTest{
	IteratorTest{
		Name:    "Add",
		Operand: "+",
		Bits:    2,
		Eval: func(a int64, b int64) int64 {
			return a + b
		},
		Code: `
package main
func main(a, b uint3) uint3 {
    return a + b
}
`,
	},
	// 1-bit, 2-bit, and n-bit multipliers have a bit different wiring.
	IteratorTest{
		Name:    "Multiply 1-bit",
		Operand: "*",
		Bits:    1,
		Eval: func(a int64, b int64) int64 {
			return a * b
		},
		Code: `
package main
func main(a, b uint1) uint1 {
    return a * b
}
`,
	},
	IteratorTest{
		Name:    "Multiply 2-bits",
		Operand: "*",
		Bits:    2,
		Eval: func(a int64, b int64) int64 {
			return a * b
		},
		Code: `
package main
func main(a, b uint4) uint4 {
    return a * b
}
`,
	},
	IteratorTest{
		Name:    "Multiply n-bits",
		Operand: "*",
		Bits:    2,
		Eval: func(a int64, b int64) int64 {
			return a * b
		},
		Code: `
package main
func main(a, b uint6) uint6 {
    return a * b
}
`,
	},
}

func TestIterator(t *testing.T) {
	for _, test := range iteratorTests {
		circ, _, err := NewCompiler(&utils.Params{}).Compile(test.Code)
		if err != nil {
			t.Fatalf("Failed to compile test %s: %s", test.Name, err)
		}

		limit := int64(1 << test.Bits)

		var g, e int64

		for g = 0; g < limit; g++ {
			for e = 0; e < limit; e++ {

				results, err := circ.Compute([]*big.Int{big.NewInt(g)},
					[]*big.Int{big.NewInt(e)})
				if err != nil {
					t.Fatalf("compute failed: %s\n", err)
				}

				expected := test.Eval(g, e)

				if expected != results[0].Int64() {
					t.Errorf("%s failed: %d %s %d = %d, expected %d\n",
						test.Name, g, test.Operand, e, results[0],
						expected)
				}
			}
		}
	}
}

type FixedTest struct {
	N1   int64
	N2   int64
	N3   int64
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
	FixedTest{
		N1: 5,
		N2: 3,
		N3: 8,
		Code: `
package main
func main(a, b int) int {
    return Sum2(MinMax(a, b))
}
func Sum2(a, b int) int {
    return a + b
}
func MinMax(a, b int) (int, int) {
    if a > b {
        return b, a
    }
    return a, b
}
`,
	},
	// For raw sha256 without padding, the digest is as follow:
	// da5698be17b9b46962335799779fbeca8ce5d491c0d26243bafef9ea1837a9d8
	FixedTest{
		N1:   0,
		N2:   0,
		N3:   -4972262154746484264,
		Zero: true,
		Code: `
package main
import (
    "crypto/sha256"
)
func main(a, b uint512) uint256 {
    return sha256.Block(a, sha256.H0)
}
`,
	},
	// For raw sha512 without padding, the digest is as follow:
	// 0xcf7881d5774acbe8533362e0fbc780700267639d87460eda3086cb40e85931b0717dc95288a023a396bab2c14ce0b5e06fc4fe04eae33e0b91f4d80cbd668bee
	FixedTest{
		N1:   0,
		N2:   0,
		N3:   -7929475494663779346,
		Zero: true,
		Code: `
package main
import (
    "crypto/sha512"
)
func main(a, b uint1024) uint512 {
    return sha512.Block(a, sha512.H0)
}
`,
	},
}

func TestFixed(t *testing.T) {
	for idx, test := range fixedTests {
		circ, _, err := NewCompiler(&utils.Params{}).Compile(test.Code)
		if err != nil {
			t.Errorf("Failed to compile test %d: %s", idx, err)
			continue
		}
		var n1 = []*big.Int{big.NewInt(test.N1)}
		if test.Zero {
			n1 = append(n1, big.NewInt(0))
		}
		results, err := circ.Compute(n1, []*big.Int{big.NewInt(test.N2)})
		if err != nil {
			t.Errorf("compute failed: %s", err)
			continue
		}
		if results[0].Int64() != test.N3 {
			t.Errorf("test %d failed: got %d (%x), expected %d (%x)", idx,
				results[0].Int64(), results[0].Int64(), test.N3, test.N3)
		}
	}
}
