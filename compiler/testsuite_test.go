//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"fmt"
	"math/big"
	"os"
	"path"
	"regexp"
	"runtime/pprof"
	"strings"
	"testing"

	"github.com/markkurossi/mpc/compiler/utils"
)

const (
	testsuite = "tests"
)

var (
	reWhitespace = regexp.MustCompilePOSIX(`[[:space:]]+`)
)

func TestSuite(t *testing.T) {
	dir, err := os.Open(testsuite)
	if err != nil {
		t.Fatalf("Failed to open tests directory: %s", err)
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		t.Fatalf("Failed to list tests directory: %s", err)
	}

	compiler := NewCompiler(&utils.Params{})

loop:
	for _, file := range files {
		if !strings.HasSuffix(file, ".mpcl") {
			continue
		}
		name := path.Join(testsuite, file)
		circ, annotations, err := compiler.CompileFile(name)
		if err != nil {
			t.Errorf("failed to compiler '%s': %s", file, err)
			continue
		}

		var cpuprof bool
		base := 10

		for _, ann := range annotations {
			if strings.HasPrefix(ann, "@heavy") && testing.Short() {
				fmt.Printf("Skipping heavy test %s\n", file)
				continue loop
			}
			if strings.HasPrefix(ann, "@pprof") {
				cpuprof = true
				continue
			}
			if strings.HasPrefix(ann, "@Hex") {
				base = 16
				continue
			}
			if !strings.HasPrefix(ann, "@Test ") {
				continue
			}
			parts := reWhitespace.Split(ann, -1)

			var inputs []*big.Int
			var outputs []*big.Int
			var sep bool

			split := -1

			for i := 1; i < len(parts); i++ {
				part := parts[i]
				if part == "=" {
					sep = true
					continue
				}
				v := new(big.Int)
				_, ok := v.SetString(parts[i], 0)
				if !ok {
					t.Errorf("%s: invalid argument '%s'", file, parts[i])
					continue loop
				}
				if sep {
					outputs = append(outputs, v)
				} else {
					inputs = append(inputs, v)
				}
			}

			if split < 0 {
				split = len(inputs) / 2
			}
			var n1, n2 []*big.Int

			n1 = append(n1, inputs[:split]...)
			n2 = append(n2, inputs[split:]...)

			var prof *os.File

			if cpuprof {
				prof, err = os.Create(fmt.Sprintf("%s.cpu.prof", file))
				if err != nil {
					t.Errorf("%s: failed to create cpu.prof: %s", file, err)
					continue loop
				}
				err = pprof.StartCPUProfile(prof)
				if err != nil {
					t.Errorf("%s: failed to start CPU profile: %s", file, err)
					continue loop
				}
			}

			results, err := circ.Compute(n1, n2)

			if cpuprof {
				pprof.StopCPUProfile()
				prof.Close()
			}

			if err != nil {
				t.Errorf("%s: compute failed: %s", file, err)
				continue loop
			}

			if len(results) != len(outputs) {
				t.Errorf("%s: unexpected return values: got %v, expected %v",
					file, results, outputs)
				continue loop
			}
			for idx, result := range results {
				if result.Cmp(outputs[idx]) != 0 {
					t.Errorf("%s: result %d mismatch: got %v, expected %v",
						file, idx, result.Text(base), outputs[idx].Text(base))
				}
			}

			_ = results
		}
	}
}
