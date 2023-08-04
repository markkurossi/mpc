//
// Copyright (c) 2020-2023 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"io"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/types"
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
	freeWires      map[types.Size][][]*circuits.Wire
	nextWireID     uint32
	zeroWire       *circuits.Wire
	oneWire        *circuits.Wire
	stats          circuit.Stats
	numWires       int
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
		freeWires: make(map[types.Size][][]*circuits.Wire),
	}

	// Inputs into wires.
	for idx, arg := range in {
		if len(arg.Name) == 0 {
			arg.Name = fmt.Sprintf("arg{%d}", idx)
		}
		wires, err := prog.Wires(arg.Name, types.Size(arg.Size))
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

// Wires allocates unassigned wires for the argument value.
func (prog *Program) Wires(v string, bits types.Size) (
	[]*circuits.Wire, error) {

	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	alloc, ok := prog.wires[v]
	if !ok {
		alloc = prog.allocWires(bits, false)
		prog.wires[v] = alloc
	}
	return alloc.Wires, nil
}

// AssignedWires allocates assigned wires for the argument value.
func (prog *Program) AssignedWires(v string, bits types.Size) (
	[]*circuits.Wire, error) {
	if bits <= 0 {
		return nil, fmt.Errorf("size not set for value %v", v)
	}
	alloc, ok := prog.wires[v]
	if !ok {
		alloc = prog.allocWires(bits, true)
		prog.wires[v] = alloc
	}
	return alloc.Wires, nil
}

func (prog *Program) allocWires(bits types.Size, assign bool) *wireAlloc {

	result := &wireAlloc{
		Base: circuits.UnassignedID,
	}

	fl, ok := prog.freeWires[bits]
	if ok && len(fl) > 0 {
		result.Wires = fl[0]
		result.Base = result.Wires[0].ID()
		prog.freeWires[bits] = fl[1:]
	} else {
		result.Wires = circuits.MakeWires(bits)
	}

	if assign && result.Base == circuits.UnassignedID {
		// Assign wire IDs.
		result.Base = prog.nextWireID
		for i := 0; i < int(bits); i++ {
			result.Wires[i].SetID(prog.nextWireID + uint32(i))
		}
		prog.nextWireID += uint32(bits)
	}

	return result
}

func (prog *Program) recycleWires(alloc *wireAlloc) {
	if alloc.Base == circuits.UnassignedID {
		alloc.Base = alloc.Wires[0].ID()
	}
	// Clear wires and reassign their IDs.
	bits := types.Size(len(alloc.Wires))
	for i := 0; i < int(bits); i++ {
		alloc.Wires[i].Reset(alloc.Base + uint32(i))
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

// SetWires allocates wire IDs for the value's wires.
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
		alloc.Base = w[0].ID()
	}

	prog.wires[v] = alloc

	return nil
}

func (prog *Program) liveness() {
	aliases := make(map[ValueID]Value)

	// Collect value aliases.
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

// GC adds garbage collect (gc) instructions to recycle dead value
// wires.
func (prog *Program) GC() {
	set := big.NewInt(0)

	// Return values are live at the end of the program.
	if len(prog.Steps) == 0 {
		panic("empty program")
	}
	last := prog.Steps[len(prog.Steps)-1]
	if last.Instr.Op != Ret {
		panic("last instruction is not return")
	}
	for _, in := range last.Instr.In {
		set.SetBit(set, int(in.ID), 1)
	}

	start := time.Now()

	// Collect value aliases.
	aliases := make(map[ValueID][]Value)
	for i := 0; i < len(prog.Steps); i++ {
		step := &prog.Steps[i]
		switch step.Instr.Op {
		case Lshift, Rshift, Srshift, Slice, Mov, Smov, Amov:
			// Output is an alias for all non-const inputs.
			for _, in := range step.Instr.In {
				if in.Const {
					continue
				}
				aliases[in.ID] = append(aliases[in.ID], *step.Instr.Out)
			}
		}
	}

	steps := make([]Step, 0, len(prog.Steps))

	for i := len(prog.Steps) - 1; i >= 0; i-- {
		step := &prog.Steps[i]
		var gcs []Step

		for _, in := range step.Instr.In {
			if in.Const {
				continue
			}
			// Is input live after this instruction?
			if set.Bit(int(in.ID)) == 0 {
				var live bool
				// Check if input aliases are live.
				for _, alias := range aliases[in.ID] {
					if set.Bit(int(alias.ID)) == 1 {
						live = true
					}
				}
				if !live {
					// Input is not live.
					gcs = append(gcs, Step{
						Instr: NewGCInstr(in.String()),
					})
				}
			}
			set.SetBit(set, int(in.ID), 1)
		}
		if step.Instr.Out != nil {
			set.SetBit(set, int(step.Instr.Out.ID), 0)
		}

		reverse(gcs)
		steps = append(steps, gcs...)
		steps = append(steps, *step)
	}
	reverse(steps)
	prog.Steps = steps

	elapsed := time.Since(start)

	if prog.Params.Diagnostics {
		fmt.Printf(" - Program.GC: %s\n", elapsed)
	}
}

func reverse(steps []Step) {
	for i, j := 0, len(steps)-1; i < j; i, j = i+1, j-1 {
		steps[i], steps[j] = steps[j], steps[i]
	}
}

// DefineConstants defines the program constants.
func (prog *Program) DefineConstants(zero, one *circuits.Wire) error {

	var consts []Value
	for _, c := range prog.Constants {
		consts = append(consts, c.Const)
	}
	sort.Slice(consts, func(i, j int) bool {
		return strings.Compare(consts[i].Name, consts[j].Name) == -1
	})

	var constWires int
	for _, c := range consts {
		_, ok := prog.wires[c.String()]
		if ok {
			continue
		}

		constWires += int(c.Type.Bits)

		var wires []*circuits.Wire
		for bit := types.Size(0); bit < c.Type.Bits; bit++ {
			var w *circuits.Wire
			if c.Bit(bit) {
				w = one
			} else {
				w = zero
			}
			wires = append(wires, w)
		}

		err := prog.SetWires(c.String(), wires)
		if err != nil {
			return err
		}
	}
	if len(consts) > 0 && prog.Params.Verbose {
		fmt.Printf("Defined %d constants: %d wires\n",
			len(consts), constWires)
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
				fmt.Fprintf(out, "#\t\t- %v\n", live)
			}
		}
	}
}
