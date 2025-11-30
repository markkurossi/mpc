//
// Copyright (c) 2019-2025 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/env"
	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

type Test struct {
	Name    string
	Heavy   bool
	Operand string
	Bits    int
	Eval    func(a *big.Int, b *big.Int) *big.Int
	Code    string
}

var tests = []Test{
	{
		Name:    "Add",
		Heavy:   true,
		Operand: "+",
		Bits:    2,
		Eval: func(a *big.Int, b *big.Int) *big.Int {
			result := big.NewInt(0)
			result.Add(a, b)
			return result
		},
		Code: `
package main
func main(a, b int3) int3 {
    return a + b
}
`,
	},
	// 1-bit, 2-bit, and n-bit multipliers have a bit different wiring.
	{
		Name:    "Multiply 1-bit",
		Heavy:   false,
		Operand: "*",
		Bits:    1,
		Eval: func(a *big.Int, b *big.Int) *big.Int {
			result := big.NewInt(0)
			result.Mul(a, b)
			return result
		},
		Code: `
package main
func main(a, b int1) int1 {
    return a * b
}
`,
	},
	{
		Name:    "Multiply 2-bits",
		Heavy:   true,
		Operand: "*",
		Bits:    2,
		Eval: func(a *big.Int, b *big.Int) *big.Int {
			result := big.NewInt(0)
			result.Mul(a, b)
			return result
		},
		Code: `
package main
func main(a, b int4) int4 {
    return a * b
}
`,
	},
	{
		Name:    "Multiply n-bits",
		Heavy:   true,
		Operand: "*",
		Bits:    2,
		Eval: func(a *big.Int, b *big.Int) *big.Int {
			result := big.NewInt(0)
			result.Mul(a, b)
			return result
		},
		Code: `
package main
func main(a, b int6) int6 {
    return a * b
}
`,
	},
}

func TestArithmetics(t *testing.T) {
	cfg := &env.Config{}
	rand := cfg.GetRandom()

	for _, test := range tests {
		if testing.Short() && test.Heavy {
			fmt.Printf("Skipping %s\n", test.Name)
			continue
		}
		circ, _, err := New(utils.NewParams()).Compile(test.Code, nil)
		if err != nil {
			t.Fatalf("Failed to compile test %s: %s", test.Name, err)
		}

		limit := 1 << test.Bits

		for g := 0; g < limit; g++ {
			for e := 0; e < limit; e++ {

				gConn, eConn := p2p.Pipe()

				gInput := big.NewInt(int64(g))
				eInput := big.NewInt(int64(e))

				gerr := make(chan error)

				go func() {
					_, err := circuit.Garbler(cfg, gConn, ot.NewCO(rand), circ,
						gInput, false)
					gerr <- err
				}()

				result, err := circuit.Evaluator(eConn, ot.NewCO(rand), circ,
					eInput, false)
				if err != nil {
					t.Fatalf("Evaluator failed: %s\n", err)
				}

				err = <-gerr
				if err != nil {
					t.Fatalf("Garbler failed: %s\n", err)
				}

				expected := test.Eval(gInput, eInput)

				if expected.Cmp(result[0]) != 0 {
					t.Errorf("%s failed: %s %s %s = %s, expected %s\n",
						test.Name, gInput, test.Operand, eInput, result,
						expected)
				}
			}
		}
	}
}

var mult512 = `
package main
func main(a, b int512) int512 {
    return a * b
}
`

func BenchmarkMult(b *testing.B) {
	cfg := &env.Config{}
	rand := cfg.GetRandom()

	circ, _, err := New(utils.NewParams()).Compile(mult512, nil)
	if err != nil {
		b.Fatalf("failed to compile test: %s", err)
	}

	gConn, eConn := p2p.Pipe()

	gInput := big.NewInt(int64(11))
	eInput := big.NewInt(int64(13))

	gerr := make(chan error)

	go func() {
		_, err := circuit.Garbler(cfg, gConn, ot.NewCO(rand), circ, gInput,
			false)
		gerr <- err
	}()

	_, err = circuit.Evaluator(eConn, ot.NewCO(rand), circ, eInput, false)
	if err != nil {
		b.Fatalf("Evaluator failed: %s\n", err)
	}

	err = <-gerr
	if err != nil {
		b.Fatalf("Garbler failed: %s\n", err)
	}
}
