//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"fmt"
	"io"
	"math/big"
	"testing"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
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
	for _, test := range tests {
		if testing.Short() && test.Heavy {
			fmt.Printf("Skipping %s\n", test.Name)
			continue
		}
		circ, _, err := New(utils.NewParams()).Compile(test.Code)
		if err != nil {
			t.Fatalf("Failed to compile test %s: %s", test.Name, err)
		}

		limit := 1 << test.Bits

		for g := 0; g < limit; g++ {
			for e := 0; e < limit; e++ {
				gr, ew := io.Pipe()
				er, gw := io.Pipe()

				gio := newReadWriter(gr, gw)
				eio := newReadWriter(er, ew)

				gInput := big.NewInt(int64(g))
				eInput := big.NewInt(int64(e))

				gerr := make(chan error)

				go func() {
					_, err := circuit.Garbler(p2p.NewConn(gio), ot.NewCO(),
						circ, gInput, false)
					gerr <- err
				}()

				result, err := circuit.Evaluator(p2p.NewConn(eio),
					ot.NewCO(), circ, eInput, false)
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
	circ, _, err := New(utils.NewParams()).Compile(mult512)
	if err != nil {
		b.Fatalf("failed to compile test: %s", err)
	}

	gr, ew := io.Pipe()
	er, gw := io.Pipe()

	gio := newReadWriter(gr, gw)
	eio := newReadWriter(er, ew)

	gInput := big.NewInt(int64(11))
	eInput := big.NewInt(int64(13))

	gerr := make(chan error)

	go func() {
		_, err := circuit.Garbler(p2p.NewConn(gio), ot.NewCO(), circ, gInput,
			false)
		gerr <- err
	}()

	_, err = circuit.Evaluator(p2p.NewConn(eio), ot.NewCO(), circ, eInput,
		false)
	if err != nil {
		b.Fatalf("Evaluator failed: %s\n", err)
	}

	err = <-gerr
	if err != nil {
		b.Fatalf("Garbler failed: %s\n", err)
	}
}

func newReadWriter(in io.Reader, out io.Writer) io.ReadWriter {
	return &wrap{
		in:  in,
		out: out,
	}
}

type wrap struct {
	in  io.Reader
	out io.Writer
}

func (w *wrap) Read(p []byte) (n int, err error) {
	return w.in.Read(p)
}

func (w *wrap) Write(p []byte) (n int, err error) {
	return w.out.Write(p)
}
