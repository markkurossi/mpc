//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/utils"
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
	cache := make(map[string]*circuit.Circuit)

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

		var o []*circuits.Wire
		var err error
		if instr.Out != nil {
			o, err = prog.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
		}

		switch instr.Op {

		case Slice, Mov, Ret:

		case GC:
			_, ok := prog.wires[instr.GC]
			if ok {
				delete(prog.wires, instr.GC)
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
				// XXX this is wrong
				cc, err := circuits.NewCompiler(params,
					prog.Inputs, prog.Outputs,
					prog.InputWires, prog.OutputWires)
				if err != nil {
					return err
				}

				if params.Verbose {
					fmt.Printf("%d: %s\n", idx, instr.StringTyped())
				}
				err = f(cc, instr, wires, o)
				if err != nil {
					return err
				}
				pruned := cc.Prune()
				if params.Verbose {
					fmt.Printf("%d: %s: pruned %d gates\n",
						idx, instr.StringTyped(), pruned)
				}
				circ = cc.Compile()
				cache[instr.StringTyped()] = circ
				if params.Verbose {
					fmt.Printf("%d: %s: %s\n", idx, instr.StringTyped(), circ)
				}
			}
		}
	}

	return nil
}
