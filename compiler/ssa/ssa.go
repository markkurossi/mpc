//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"io"

	"github.com/markkurossi/mpc/circuit"
)

type SSA struct {
	Inputs    circuit.IO
	Outputs   circuit.IO
	Program   *Program
	Generator *Generator
}

type Program struct {
	Steps []Step
}

type Step struct {
	Label string
	Instr Instr
	Live  []string
}

func (prog *Program) PP(out io.Writer) {
	for _, step := range prog.Steps {
		if len(step.Label) > 0 {
			fmt.Fprintf(out, "# %s:\n", step.Label)
		}
		step.Instr.PP(out)
	}
}
