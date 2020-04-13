//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/utils"
)

type Program struct {
	Params         *utils.Params
	Inputs         circuit.IO
	Outputs        circuit.IO
	InputWires     []*circuits.Wire
	OutputWires    []*circuits.Wire
	Constants      map[string]ConstantInst
	Steps          []Step
	wires          map[string][]*circuits.Wire
	freeWires      map[int][][]*circuits.Wire
	nextWireID     uint32
	zeroWire       *circuits.Wire
	oneWire        *circuits.Wire
	numGates       uint64
	numNonXOR      uint64
	garbleDuration time.Duration
}

func NewProgram(params *utils.Params, in, out circuit.IO,
	consts map[string]ConstantInst, steps []Step) (*Program, error) {

	prog := &Program{
		Params:    params,
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
		wires = prog.allocWires(bits, false)
		prog.wires[v] = wires
	}
	return wires, nil
}

func (prog *Program) AssignedWires(v string, bits int) (
	[]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for variable %v", v)
	}
	wires, ok := prog.wires[v]
	if !ok {
		wires = prog.allocWires(bits, true)
		prog.wires[v] = wires
	}
	return wires, nil
}

func (prog *Program) allocWires(bits int, assign bool) (
	result []*circuits.Wire) {

	fl, ok := prog.freeWires[bits]
	if ok && len(fl) > 0 {
		result = fl[0]
		prog.freeWires[bits] = fl[1:]
		return
	}

	result = circuits.MakeWires(bits)

	if assign {
		// Assign wire IDs.
		for i := 0; i < bits; i++ {
			result[i].ID = prog.nextWireID + uint32(i)
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

func (prog *Program) DefineConstants(zero, one *circuits.Wire) error {

	var consts []Variable
	for _, c := range prog.Constants {
		consts = append(consts, c.Const)
	}
	sort.Slice(consts, func(i, j int) bool {
		return strings.Compare(consts[i].Name, consts[j].Name) == -1
	})

	if len(consts) > 0 && prog.Params.Verbose {
		fmt.Printf("Defining constants:\n")
	}
	for _, c := range consts {
		msg := fmt.Sprintf(" - %v(%d)", c, c.Type.MinBits)

		_, ok := prog.wires[c.String()]
		if ok {
			fmt.Printf("%s\talready defined\n", msg)
			continue
		}

		var wires []*circuits.Wire
		var bitString string
		for bit := 0; bit < c.Type.MinBits; bit++ {
			var w *circuits.Wire
			if c.Bit(bit) {
				bitString = "1" + bitString
				w = one
			} else {
				bitString = "0" + bitString
				w = zero
			}
			wires = append(wires, w)
		}
		if prog.Params.Verbose {
			fmt.Printf("%s\t%s\n", msg, bitString)
		}

		err := prog.SetWires(c.String(), wires)
		if err != nil {
			return err
		}
	}
	return nil
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
