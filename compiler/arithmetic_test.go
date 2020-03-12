//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"bufio"
	"fmt"
	"io"
	"math/big"
	"testing"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
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
	Test{
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
	Test{
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
	Test{
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
	Test{
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
	for _, test := range tests {
		if testing.Short() && test.Heavy {
			fmt.Printf("Skipping %s\n", test.Name)
			continue
		}
		circ, _, err := NewCompiler(&utils.Params{}).Compile(test.Code)
		if err != nil {
			t.Fatalf("Failed to compile test %s: %s", test.Name, err)
		}

		limit := 1 << test.Bits

		for g := 0; g < limit; g++ {
			for e := 0; e < limit; e++ {
				gr, ew := io.Pipe()
				er, gw := io.Pipe()

				gio := bufio.NewReadWriter(
					bufio.NewReader(gr),
					bufio.NewWriter(gw))
				eio := bufio.NewReadWriter(
					bufio.NewReader(er),
					bufio.NewWriter(ew))

				gInput := []*big.Int{big.NewInt(int64(g))}
				eInput := []*big.Int{big.NewInt(int64(e))}

				go func() {
					_, err := circuit.Garbler(circuit.NewConn(gio), circ,
						gInput, false)
					if err != nil {
						t.Fatalf("Garbler failed: %s\n", err)
					}
				}()

				result, err := circuit.Evaluator(circuit.NewConn(eio), circ,
					eInput, false)
				if err != nil {
					t.Fatalf("Evaluator failed: %s\n", err)
				}

				expected := test.Eval(gInput[0], eInput[0])

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
	circ, _, err := NewCompiler(&utils.Params{}).Compile(mult512)
	if err != nil {
		b.Fatalf("failed to compile test: %s", err)
	}

	gr, ew := io.Pipe()
	er, gw := io.Pipe()

	gio := bufio.NewReadWriter(bufio.NewReader(gr), bufio.NewWriter(gw))
	eio := bufio.NewReadWriter(bufio.NewReader(er), bufio.NewWriter(ew))

	gInput := []*big.Int{big.NewInt(int64(11))}
	eInput := []*big.Int{big.NewInt(int64(13))}

	go func() {
		_, err := circuit.Garbler(circuit.NewConn(gio), circ, gInput, false)
		if err != nil {
			b.Fatalf("Garbler failed: %s\n", err)
		}
	}()

	_, err = circuit.Evaluator(circuit.NewConn(eio), circ, eInput, false)
	if err != nil {
		b.Fatalf("Evaluator failed: %s\n", err)
	}
}
