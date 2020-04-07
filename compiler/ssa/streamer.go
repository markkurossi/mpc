//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/ot"
)

type NewCircuit func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) error

type NewBinary func(cc *circuits.Compiler, a, b []*circuits.Wire,
	out []*circuits.Wire) error

func newBinary(bin NewBinary) NewCircuit {
	return func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) error {
		return bin(cc, in[0], in[1], out)
	}
}

func newMultiplier(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) error {
	return circuits.NewMultiplier(cc, cc.Params.CircMultArrayTreshold,
		in[0], in[1], out)
}

func newDivider(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) error {
	return circuits.NewDivider(cc, in[0], in[1], out, nil)
}

func newModulo(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
	out []*circuits.Wire) error {
	return circuits.NewDivider(cc, in[0], in[1], nil, out)
}

var circuitGenerators = map[Operand]NewCircuit{
	Iadd:  newBinary(circuits.NewAdder),
	Uadd:  newBinary(circuits.NewAdder),
	Isub:  newBinary(circuits.NewSubtractor),
	Usub:  newBinary(circuits.NewSubtractor),
	Imult: newMultiplier,
	Umult: newMultiplier,
	Idiv:  newDivider,
	Udiv:  newDivider,
	Imod:  newModulo,
	Umod:  newModulo,
	Ilt:   newBinary(circuits.NewLtComparator),
	Ult:   newBinary(circuits.NewLtComparator),
	Ile:   newBinary(circuits.NewLeComparator),
	Ule:   newBinary(circuits.NewLeComparator),
	Igt:   newBinary(circuits.NewGtComparator),
	Ugt:   newBinary(circuits.NewGtComparator),
	Ige:   newBinary(circuits.NewGeComparator),
	Uge:   newBinary(circuits.NewGeComparator),
	Eq:    newBinary(circuits.NewEqComparator),
	Neq:   newBinary(circuits.NewNeqComparator),
	And:   newBinary(circuits.NewLogicalAND),
	Or:    newBinary(circuits.NewLogicalOR),
	Band:  newBinary(circuits.NewBinaryAND),
	Bclr:  newBinary(circuits.NewBinaryClear),
	Bor:   newBinary(circuits.NewBinaryOR),
	Bxor:  newBinary(circuits.NewBinaryXOR),

	Builtin: func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) error {
		return instr.Builtin(cc, in[0], in[1], out)
	},
	Phi: func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) error {
		return circuits.NewMUX(cc, in[0], in[1], in[2], out)
	},
	Bts: func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) error {
		if !instr.In[1].Const {
			return fmt.Errorf("%s only constant index supported", instr.Op)
		}
		var index int
		switch val := instr.In[1].ConstValue.(type) {
		case int32:
			index = int(val)
		default:
			return fmt.Errorf("%s unsupported index type %T", instr.Op, val)
		}
		return circuits.NewBitSetTest(cc, in[0], index, out)
	},
	Btc: func(cc *circuits.Compiler, instr Instr, in [][]*circuits.Wire,
		out []*circuits.Wire) error {
		if !instr.In[1].Const {
			return fmt.Errorf("%s only constant index supported", instr.Op)
		}
		var index int
		switch val := instr.In[1].ConstValue.(type) {
		case int32:
			index = int(val)
		default:
			return fmt.Errorf("%s unsupported index type %T", instr.Op, val)
		}
		return circuits.NewBitClrTest(cc, in[0], index, out)
	},
}

func (prog *Program) StreamCircuit(params *utils.Params) error {
	var key [32]byte
	_, err := rand.Read(key[:])
	if err != nil {
		return err
	}
	prog.assignWires = true

	var numGates uint64
	var numNonXOR uint64
	cache := make(map[string]*circuit.Circuit)

	r, err := ot.NewLabel(rand.Reader)
	if err != nil {
		return err
	}
	r.SetS(true)

	for idx, step := range prog.Steps {
		if idx%1000 == 0 {
			fmt.Printf("%d/%d\n", idx, len(prog.Steps))
		}
		instr := step.Instr
		var wires [][]*circuits.Wire
		for _, in := range instr.In {
			w, err := prog.Wires(in.String(), in.Type.Bits)
			if err != nil {
				return err
			}
			wires = append(wires, w)
		}

		var out []*circuits.Wire
		var err error
		if instr.Out != nil {
			out, err = prog.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
		}

		switch instr.Op {

		case Slice, Mov, Ret:

		case GC:
			wires, ok := prog.wires[instr.GC]
			if ok {
				delete(prog.wires, instr.GC)
				prog.recycleWires(wires)
			} else {
				fmt.Printf("GC: %s not known\n", instr.GC)
			}

		default:
			f, ok := circuitGenerators[instr.Op]
			if !ok {
				return fmt.Errorf("Program.Stream: %s not implemented yet",
					instr.Op)
			}

			// Flatten input wires.
			var flat []*circuits.Wire
			for _, w := range wires {
				flat = append(flat, w...)
			}

			circ, ok := cache[instr.StringTyped()]
			if !ok {
				// Clear output wires from input wires (they could be
				// outputs of previous computation).
				for _, w := range flat {
					w.Output = false
				}
				// Mark outputs as output wires.
				for _, o := range out {
					o.Output = true
				}

				cc, err := circuits.NewCompiler(params, nil, nil, flat, out)
				if err != nil {
					return err
				}
				cc.OutputsAssigned = true
				cc.SetNextWireID(0x80000000)

				if params.Verbose {
					fmt.Printf("%05d: %s\n", idx, instr.StringTyped())
				}
				err = f(cc, instr, wires, out)
				if err != nil {
					return err
				}
				pruned := cc.Prune()
				if params.Verbose {
					fmt.Printf("%05d: - pruned %d gates\n",
						idx, pruned)
				}
				circ = cc.Compile()
				cache[instr.StringTyped()] = circ
				if params.Verbose {
					fmt.Printf("%05d: - %s\n", idx, circ)
				}
			}
			if false {
				circ.Dump()
			}
			if true {
				fmt.Printf("%05d: - garble %d gates\n", idx, circ.NumGates)

				var inputIDs []circuit.Wire
				for _, in := range flat {
					inputIDs = append(inputIDs, circuit.Wire(in.ID))
				}

				start := time.Now()
				err := circ.GarbleStream(key[:], r, inputIDs)
				if err != nil {
					return err
				}
				dt := time.Now().Sub(start)
				elapsed := time.Now().UnixNano() - start.UnixNano()
				elapsed /= 1000000000
				if elapsed > 0 {
					fmt.Printf("%05d: - garbled %d gates/s (%s)\n",
						idx, int64(circ.NumGates)/elapsed, dt)
				}
			}
			numGates += uint64(circ.NumGates)
			numNonXOR += uint64(circ.Stats[circuit.AND])
			numNonXOR += uint64(circ.Stats[circuit.OR])
			numNonXOR += uint64(circ.Stats[circuit.INV])
		}
	}

	fmt.Printf("Max permanent wires: %d, cached circuits: %d\n",
		prog.nextWireID, len(cache))
	fmt.Printf("#gates=%d, #non-XOR=%d\n", numGates, numNonXOR)

	return nil
}
