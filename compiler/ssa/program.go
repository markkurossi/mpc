//
// Copyright (c) 2020-2021 Markku Rossi
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

// Program implements SSA program.
type Program struct {
	Params         *utils.Params
	Inputs         circuit.IO
	Outputs        circuit.IO
	InputWires     []*circuits.Wire
	OutputWires    []*circuits.Wire
	Constants      map[string]ConstantInst
	Steps          []Step
	wires          map[string]*wireAlloc
	freeWires      map[int][][]*circuits.Wire
	nextWireID     uint32
	zeroWire       *circuits.Wire
	oneWire        *circuits.Wire
	numGates       uint64
	numNonXOR      uint64
	garbleDuration time.Duration
}

type wireAlloc struct {
	Base  uint32
	Wires []*circuits.Wire
}

// NewProgram creates a new program for the constants and program
// steps.
func NewProgram(params *utils.Params, in, out circuit.IO,
	consts map[string]ConstantInst, steps []Step) (*Program, error) {

	prog := &Program{
		Params:    params,
		Inputs:    in,
		Outputs:   out,
		Constants: consts,
		Steps:     steps,
		wires:     make(map[string]*wireAlloc),
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

// Step defines one SSA program step.
type Step struct {
	Label string
	Instr Instr
	Live  Set
}

// Wires allocates unassigned wires for the argument variable.
func (prog *Program) Wires(v string, bits int) ([]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for variable %v", v)
	}
	alloc, ok := prog.wires[v]
	if !ok {
		alloc = prog.allocWires(bits, false)
		prog.wires[v] = alloc
	}
	return alloc.Wires, nil
}

// AssignedWires allocates assigned wires for the argument variable.
func (prog *Program) AssignedWires(v string, bits int) (
	[]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for variable %v", v)
	}
	alloc, ok := prog.wires[v]
	if !ok {
		alloc = prog.allocWires(bits, true)
		prog.wires[v] = alloc
	}
	return alloc.Wires, nil
}

func (prog *Program) allocWires(bits int, assign bool) *wireAlloc {

	result := &wireAlloc{
		Base: circuits.UnassignedID,
	}

	fl, ok := prog.freeWires[bits]
	if ok && len(fl) > 0 {
		result.Wires = fl[0]
		result.Base = result.Wires[0].ID
		prog.freeWires[bits] = fl[1:]
	} else {
		result.Wires = circuits.MakeWires(bits)
	}

	if assign && result.Base == circuits.UnassignedID {
		// Assign wire IDs.
		result.Base = prog.nextWireID
		for i := 0; i < bits; i++ {
			result.Wires[i].ID = prog.nextWireID + uint32(i)
		}
		prog.nextWireID += uint32(bits)
	}

	return result
}

func (prog *Program) recycleWires(alloc *wireAlloc) {
	if alloc.Base == circuits.UnassignedID {
		alloc.Base = alloc.Wires[0].ID
	}
	// Clear wires and reassign their IDs.
	bits := len(alloc.Wires)
	for i := 0; i < bits; i++ {
		alloc.Wires[i].ID = alloc.Base + uint32(i)
		alloc.Wires[i].Output = false
		alloc.Wires[i].NumOutputs = 0
		alloc.Wires[i].Input = nil
		alloc.Wires[i].Outputs = nil
	}

	fl := prog.freeWires[bits]
	fl = append(fl, alloc.Wires)
	prog.freeWires[bits] = fl
	if false {
		fmt.Printf("FL: %d: ", bits)
		for k, v := range prog.freeWires {
			fmt.Printf(" %d:%d", k, len(v))
		}
		fmt.Println()
	}
}

// SetWires allocates wire IDs for the variable's wires.
func (prog *Program) SetWires(v string, w []*circuits.Wire) error {
	_, ok := prog.wires[v]
	if ok {
		return fmt.Errorf("wires already set for %v", v)
	}
	alloc := &wireAlloc{
		Wires: w,
	}
	if len(w) == 0 {
		alloc.Base = circuits.UnassignedID
	} else {
		alloc.Base = w[0].ID
	}

	prog.wires[v] = alloc

	return nil
}

func (prog *Program) liveness() {
	aliases := make(map[VariableID]Variable)

	// Collect variable aliases.
	for i := 0; i < len(prog.Steps); i++ {
		step := &prog.Steps[i]
		switch step.Instr.Op {
		case Slice, Mov:
			if !step.Instr.In[0].Const {
				// The `out' will be an alias for `in[0]'.
				aliases[step.Instr.Out.ID] = step.Instr.In[0]
			}
		case Amov:
			// v arr from to o: v | arr[from:to] = o
			// XXX aliases are 1:1 mapping but here amov's output
			// aliases two inputs.
			if !step.Instr.In[0].Const && false {
				// The `out' will be an alias for `in[0]'
				aliases[step.Instr.Out.ID] = step.Instr.In[0]
			}
			if !step.Instr.In[1].Const {
				// The `out' will be an alias for `in[1]'
				aliases[step.Instr.Out.ID] = step.Instr.In[1]
			}
		}
	}

	live := NewSet()

	for i := len(prog.Steps) - 1; i >= 0; i-- {
		step := &prog.Steps[i]
		for _, in := range step.Instr.In {
			if in.Const {
				continue
			}
			live.Add(in)
		}

		if step.Instr.Out != nil {
			delete(live, step.Instr.Out.ID)
		}
		step.Live = NewSet()
		for _, v := range live {
			step.Live.Add(v)
			// Follow alias chains.
			from := v
			for {
				to, ok := aliases[from.ID]
				if !ok {
					break
				}
				step.Live.Add(to)
				from = to
			}
		}
	}
}

// GC adds garbage collect (gc) instructions to recycle dead
// variable wires.
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
					Instr: NewGCInstr(d.String()),
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

// DefineConstants defines the program constants.
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

// PP pretty-prints the program to the argument io.Writer.
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
		if false {
			for _, live := range step.Live {
				fmt.Fprintf(out, "#\t\t- %s\n", live)
			}
		}
	}
}
