//
// arithmetic_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"bufio"
	"io"
	"math/big"
	"testing"

	"github.com/markkurossi/mpc/circuit"
)

var add = `
package main
func main(a, b int2) int3 {
    return a + b
}
`

func TestAdd(t *testing.T) {
	circ, err := Compile(add)
	if err != nil {
		t.Fatalf("Failed to compile test: %s", err)
	}

	var key [32]byte

	for g := 0; g < 4; g++ {
		for e := 0; e < 4; e++ {
			gr, ew := io.Pipe()
			er, gw := io.Pipe()

			gio := bufio.NewReadWriter(bufio.NewReader(gr), bufio.NewWriter(gw))
			eio := bufio.NewReadWriter(bufio.NewReader(er), bufio.NewWriter(ew))

			gInput := big.NewInt(int64(g))
			eInput := big.NewInt(int64(e))

			go func() {
				_, err := circuit.Garbler(gio, circ, gInput, key[:], false)
				if err != nil {
					t.Fatalf("Garbler failed: %s\n", err)
				}
			}()

			result, err := circuit.Evaluator(eio, circ, eInput, key[:])
			if err != nil {
				t.Fatalf("Evaluator failed: %s\n", err)
			}

			expected := big.NewInt(0)
			expected.Add(gInput, eInput)

			if expected.Cmp(result) != 0 {
				t.Errorf("Addition failed: %s + %s = %s\n",
					gInput, eInput, result)
			}
		}
	}
}
