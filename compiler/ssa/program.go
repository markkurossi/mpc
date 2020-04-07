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
	"github.com/markkurossi/mpc/compiler/circuits"
)

type Program struct {
	Inputs      circuit.IO
	Outputs     circuit.IO
	InputWires  []*circuits.Wire
	OutputWires []*circuits.Wire
	Constants   map[string]ConstantInst
	Steps       []Step
	wires       map[string][]*circuits.Wire
	freeWires   map[int][][]*circuits.Wire
	assignWires bool
	nextWireID  uint32
}

func NewProgram(in, out circuit.IO, consts map[string]ConstantInst,
	steps []Step) (*Program, error) {

	prog := &Program{
		Inputs:    in,
		Outputs:   out,
		Constants: consts,
		Steps:     steps,
		wires:     make(map[string][]*circuits.Wire),
		freeWires: make(map[int][][]*circuits.Wire),
	}

	// Inputs into wires.
	for idx, arg := range in {
		if len(arg.Name) == 0 {
			arg.Name = fmt.Sprintf("arg{%d}", idx)
		}
		wires, err := prog.Wires(arg.Name, arg.Size)
		if err != nil {
			return nil, err
		}
		prog.InputWires = append(prog.InputWires, wires...)
	}

	return prog, nil
}

type Step struct {
	Label string
	Instr Instr
	Live  Set
}

func (prog *Program) Wires(v string, bits int) ([]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for variable %v", v)
	}
	wires, ok := prog.wires[v]
	if !ok {
		wires = prog.allocWires(bits)
		prog.wires[v] = wires
	}
	return wires, nil
}

func (prog *Program) allocWires(bits int) (result []*circuits.Wire) {
	fl, ok := prog.freeWires[bits]
	if ok && len(fl) > 0 {
		result = fl[0]
		prog.freeWires[bits] = fl[1:]
		return
	}

	result = circuits.MakeWires(bits)

	if prog.assignWires {
		// Assign wire IDs.
		for i := 0; i < bits; i++ {
			result[i].ID = prog.nextWireID + uint32(i)
			fmt.Printf("Program: assigned wire %d\n", result[i].ID)
			if result[i].ID == 2 {
				panic(42)
			}
		}
		prog.nextWireID += uint32(bits)
	}

	return
}

func (prog *Program) recycleWires(wires []*circuits.Wire) {
	// Clear wires but keep their IDs.
	for _, w := range wires {
		w.Output = false
		w.NumOutputs = 0
		w.Input = nil
		w.Outputs = nil
	}

	bits := len(wires)
	fl := prog.freeWires[bits]
	fl = append(fl, wires)
	prog.freeWires[bits] = fl
	if false {
		fmt.Printf("FL: %d: ", len(wires))
		for k, v := range prog.freeWires {
			fmt.Printf(" %d:%d", k, len(v))
		}
		fmt.Println()
	}
}

func (prog *Program) SetWires(v string, w []*circuits.Wire) error {
	_, ok := prog.wires[v]
	if ok {
		return fmt.Errorf("wires already set for %v", v)
	}
	prog.wires[v] = w
	return nil
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
