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
)

func (prog *Program) StreamCircuit(params *utils.Params) error {
	var key [32]byte
	_, err := rand.Read(key[:])
	if err != nil {
		return err
	}

	// Collect input wire IDs.
	var ids []circuit.Wire
	for _, w := range prog.InputWires {
		// Program's inputs are unassigned because parser is shared
		// between streaming and non-streaming modes.
		w.ID = prog.nextWireID
		prog.nextWireID++
		ids = append(ids, circuit.Wire(w.ID))
	}

	streaming, err := circuit.NewStreaming(key[:], ids)
	if err != nil {
		return err
	}

	var numGates uint64
	var numNonXOR uint64
	cache := make(map[string]*circuit.Circuit)

	start := time.Now()

	for idx, step := range prog.Steps {
		if idx%100 == 0 {
			elapsed := time.Now().Sub(start)
			done := float64(idx) / float64(len(prog.Steps))
			if done > 0 {
				total := time.Duration(float64(elapsed) / done)
				fmt.Printf("%d/%d\t%s remaining, ready at %s\n",
					idx, len(prog.Steps),
					total-elapsed, start.Add(total).Format(time.Stamp))
			} else {
				fmt.Printf("%d/%d\n", idx, len(prog.Steps))
			}
		}
		instr := step.Instr
		var wires [][]*circuits.Wire
		for _, in := range instr.In {
			w, err := prog.AssignedWires(in.String(), in.Type.Bits)
			if err != nil {
				return err
			}
			wires = append(wires, w)
		}

		var out []*circuits.Wire
		var err error
		if instr.Out != nil {
			out, err = prog.AssignedWires(instr.Out.String(),
				instr.Out.Type.Bits)
			if err != nil {
				return err
			}
		}

		switch instr.Op {

		case Slice, Mov:

		case Ret:
			fmt.Printf("Ret: %v\n", wires)

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
			circ, ok := cache[instr.StringTyped()]
			if !ok {
				var cIn [][]*circuits.Wire
				var flat []*circuits.Wire

				for _, in := range instr.In {
					w := circuits.MakeWires(in.Type.Bits)
					cIn = append(cIn, w)
					flat = append(flat, w...)
				}

				cOut := circuits.MakeWires(instr.Out.Type.Bits)
				for i := 0; i < instr.Out.Type.Bits; i++ {
					cOut[i].Output = true
				}

				cc, err := circuits.NewCompiler(params, nil, nil, flat, cOut)
				if err != nil {
					return err
				}
				if params.Verbose {
					fmt.Printf("%05d: %s\n", idx, instr.StringTyped())
				}
				err = f(cc, instr, cIn, cOut)
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
				fmt.Printf("%05d: - circuit: %s\n", idx, circ)
			}

			// Collect input and output IDs
			var iIDs, oIDs []circuit.Wire
			for _, vars := range wires {
				for _, w := range vars {
					iIDs = append(iIDs, circuit.Wire(w.ID))
				}
			}
			for _, w := range out {
				oIDs = append(oIDs, circuit.Wire(w.ID))
			}

			gStart := time.Now()
			err := streaming.Garble(circ, iIDs, oIDs)
			if err != nil {
				return err
			}
			dt := time.Now().Sub(gStart)
			elapsed := float64(time.Now().UnixNano() - gStart.UnixNano())
			elapsed /= 1000000000
			if elapsed > 0 && false {
				fmt.Printf("%05d: - garbled %10.0f gates/s (%s)\n",
					idx, float64(circ.NumGates)/elapsed, dt)
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
