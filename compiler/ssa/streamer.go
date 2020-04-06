//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"

	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/utils"
)

type NewCircuit func(cc *circuits.Compiler, i0, i1 []*circuits.Wire,
	out []*circuits.Wire) error

func newMultiplier(cc *circuits.Compiler, i0, i1 []*circuits.Wire,
	out []*circuits.Wire) error {
	return circuits.NewMultiplier(cc, cc.Params.CircMultArrayTreshold,
		i0, i1, out)
}

func newDivider(cc *circuits.Compiler, i0, i1 []*circuits.Wire,
	out []*circuits.Wire) error {
	return circuits.NewDivider(cc, i0, i1, out, nil)
}

func newModulo(cc *circuits.Compiler, i0, i1 []*circuits.Wire,
	out []*circuits.Wire) error {
	return circuits.NewDivider(cc, i0, i1, nil, out)
}

var circuitGenerators = map[Operand]NewCircuit{
	Iadd:  circuits.NewAdder,
	Uadd:  circuits.NewAdder,
	Isub:  circuits.NewSubtractor,
	Usub:  circuits.NewSubtractor,
	Imult: newMultiplier,
	Umult: newMultiplier,
	Idiv:  newDivider,
	Udiv:  newDivider,
	Imod:  newModulo,
	Umod:  newModulo,
	Ilt:   circuits.NewLtComparator,
	Ult:   circuits.NewLtComparator,
	Ile:   circuits.NewLeComparator,
	Ule:   circuits.NewLeComparator,
	Igt:   circuits.NewGtComparator,
	Ugt:   circuits.NewGtComparator,
	Ige:   circuits.NewGeComparator,
	Uge:   circuits.NewGeComparator,
	Eq:    circuits.NewEqComparator,
	Neq:   circuits.NewNeqComparator,
	And:   circuits.NewLogicalAND,
	Or:    circuits.NewLogicalOR,
	Band:  circuits.NewBinaryAND,
	Bclr:  circuits.NewBinaryClear,
	Bor:   circuits.NewBinaryOR,
	Bxor:  circuits.NewBinaryXOR,
}

func (prog *Program) StreamCircuit(params *utils.Params) error {
	for _, step := range prog.Steps {
		instr := step.Instr
		var wires [][]*circuits.Wire
		for _, in := range instr.In {
			w, err := prog.Wires(in.String(), in.Type.Bits)
			if err != nil {
				return err
			}
			wires = append(wires, w)
		}

		o, err := prog.Wires(instr.Out.String(), instr.Out.Type.Bits)
		if err != nil {
			return err
		}

		// XXX this is wrong
		cc, err := circuits.NewCompiler(params, prog.Inputs, prog.Outputs,
			prog.InputWires, prog.OutputWires)
		if err != nil {
			return err
		}

		switch instr.Op {
		case Bts:
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
			err = circuits.NewBitSetTest(cc, wires[0], index, o)
			if err != nil {
				return err
			}

		case Btc:
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
			err = circuits.NewBitClrTest(cc, wires[0], index, o)
			if err != nil {
				return err
			}

		case Builtin:
			err = instr.Builtin(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Phi:
			err = circuits.NewMUX(cc, wires[0], wires[1], wires[2], o)
			if err != nil {
				return err
			}

		default:
			f, ok := circuitGenerators[instr.Op]
			if !ok {
				return fmt.Errorf("Program.Stream: %s not implemented yet",
					instr.Op)
			}
			err = f(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
