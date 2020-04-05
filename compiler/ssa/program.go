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

type Program struct {
	Inputs    circuit.IO
	Outputs   circuit.IO
	Constants map[string]ConstantInst
	Steps     []Step
}

type Step struct {
	Label string
	Instr Instr
	Live  Set
}

func (prog *Program) liveness() {
	live := NewSet()

	for i := len(prog.Steps) - 1; i >= 0; i-- {
		step := &prog.Steps[i]
		for _, in := range step.Instr.In {
			if in.Const {
				continue
			}
			live[in.String()] = true
		}
		if step.Instr.Out != nil {
			delete(live, step.Instr.Out.String())
		}
		step.Live = NewSet()
		for k, _ := range live {
			step.Live.Add(k)
		}
	}
}

func (prog *Program) GC() {
	steps := make([]Step, 0, len(prog.Steps))
	last := NewSet()
	for _, step := range prog.Steps {
		// GC dead variables.
		deleted := last.Copy()
		deleted.Subtract(step.Live)
		if len(last) > 0 {
			for _, d := range deleted.Array() {
				steps = append(steps, Step{
					Instr: NewGCInstr(d),
					Live:  last.Copy(),
				})
				last.Remove(d)
			}
		}
		last = step.Live
		steps = append(steps, step)
	}
	prog.Steps = steps
}

func (prog *Program) PP(out io.Writer) {
	for i, in := range prog.Inputs {
		fmt.Fprintf(out, "# Input%d: %s\n", i, in)
	}
	for i, in := range prog.Outputs {
		fmt.Fprintf(out, "# Output%d: %s\n", i, in)
	}
	for _, step := range prog.Steps {
		if len(step.Label) > 0 {
			fmt.Fprintf(out, "# %s:\n", step.Label)
		}
		step.Instr.PP(out)
	}
}
