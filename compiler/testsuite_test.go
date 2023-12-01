//
// Copyright (c) 2019-2023 Markku Rossi
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

	"github.com/markkurossi/mpc/circuit"
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

	params := utils.NewParams()
	params.MPCLCErrorLoc = true
	compiler := New(params)

loop:
	for _, file := range files {
		if !strings.HasSuffix(file, ".mpcl") {
			continue
		}
		name := path.Join(testsuite, file)
		pkg, err := compiler.ParseFile(name)
		if err != nil {
			t.Errorf("failed to parse '%s': %s", file, err)
			continue
		}
		main, ok := pkg.Functions["main"]
		if !ok {
			t.Errorf("%s: no main function", file)
			continue
		}

		var cpuprof bool
		var prof *os.File
		var lsb bool
		base := 10

		for _, annotation := range main.Annotations {
			ann := strings.TrimSpace(annotation)
			if strings.HasPrefix(ann, "@heavy") && testing.Short() {
				fmt.Printf("Skipping heavy test %s\n", file)
				continue loop
			}
			if strings.HasPrefix(ann, "@pprof") {
				cpuprof = true
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
				continue
			}
			if strings.HasPrefix(ann, "@Hex") {
				base = 16
				continue
			}
			if strings.HasPrefix(ann, "@LSB") {
				lsb = true
				continue
			}
			if !strings.HasPrefix(ann, "@Test ") {
				continue
			}
			parts := reWhitespace.Split(ann, -1)

			var inputValues []string
			var inputs []*big.Int
			var outputs []*big.Int
			var sep bool

			for i := 1; i < len(parts); i++ {
				part := parts[i]
				if part == "=" {
					sep = true
					continue
				}
				v := new(big.Int)
				if base == 16 && lsb {
					part = reverse(part)
				}

				_, ok := v.SetString(part, 0)
				if !ok {
					t.Errorf("%s: invalid argument '%s'", file, part)
					continue loop
				}
				if sep {
					outputs = append(outputs, v)
				} else {
					inputValues = append(inputValues, part)
					inputs = append(inputs, v)
				}
			}
			var inputSizes [][]int
			for _, iv := range inputValues {
				sizes, err := circuit.InputSizes([]string{iv})
				if err != nil {
					t.Errorf("%s: invalid inputs: %s", file, err)
					continue loop
				}
				inputSizes = append(inputSizes, sizes)
			}
			circ, _, err := compiler.CompileFile(name, inputSizes)
			if err != nil {
				t.Errorf("failed to compile '%s': %s", file, err)
				continue loop
			}

			results, err := circ.Compute(inputs)
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
		if cpuprof {
			pprof.StopCPUProfile()
			prof.Close()
		}
	}
}

func reverse(val string) string {
	var prefix string
	if strings.HasPrefix(val, "0x") {
		val = val[2:]
		prefix = "0x"
	}
	var result string

	for i := len(val) - 2; i >= 0; i -= 2 {
		result += val[i : i+2]
	}
	if len(val)%2 == 1 {
		result += val[0:1]
	}
	return prefix + result
}
