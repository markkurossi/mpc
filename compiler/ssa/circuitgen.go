//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"sort"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/utils"
)

func (code *SSA) CompileCircuit(params *utils.Params) (
	*circuit.Circuit, error) {

	cc, err := circuits.NewCompiler(code.Inputs, code.Outputs)
	if err != nil {
		return nil, err
	}

	err = code.Generator.DefineConstants(cc)
	if err != nil {
		return nil, err
	}

	if params.Verbose {
		fmt.Printf("Creating circuit...\n")
	}
	err = code.Circuit(code.Generator, cc)
	if err != nil {
		return nil, err
	}

	if params.Verbose {
		fmt.Printf("Compiling circuit...\n")
	}
	if params.OptPruneGates {
		pruned := cc.Prune()
		if params.Verbose {
			fmt.Printf(" - Pruned %d gates\n", pruned)
		}
	}
	circ := cc.Compile()
	if params.CircOut != nil {
		if params.Verbose {
			fmt.Printf("Serializing circuit...\n")
		}
		switch params.CircFormat {
		case "mpclc":
			if err := circ.Marshal(params.CircOut); err != nil {
				return nil, err
			}
		case "bristol":
			circ.MarshalBristol(params.CircOut)
		default:
			return nil, fmt.Errorf("unsupported circuit format: %s",
				params.CircFormat)
		}
	}
	if params.CircDotOut != nil {
		circ.Dot(params.CircDotOut)
	}

	return circ, nil
}

func (gen *Generator) DefineConstants(cc *circuits.Compiler) error {
	var consts []Variable
	for _, c := range gen.constants {
		consts = append(consts, c.Const)
	}
	sort.Slice(consts, func(i, j int) bool {
		return strings.Compare(consts[i].Name, consts[j].Name) == -1
	})

	if len(consts) > 0 && gen.Params.Verbose {
		fmt.Printf("Defining constants:\n")
	}
	for _, c := range consts {
		msg := fmt.Sprintf(" - %v(%d)", c, c.Type.MinBits)

		var wires []*circuits.Wire
		var bitString string
		for bit := 0; bit < c.Type.MinBits; bit++ {
			var w *circuits.Wire
			if c.Bit(bit) {
				bitString = "1" + bitString
				w = cc.OneWire()
			} else {
				bitString = "0" + bitString
				w = cc.ZeroWire()
			}
			wires = append(wires, w)
		}
		if gen.Params.Verbose {
			fmt.Printf("%s\t%s\n", msg, bitString)
		}

		err := cc.SetWires(c.String(), wires)
		if err != nil {
			return err
		}
	}
	return nil
}

func (code *SSA) Circuit(gen *Generator, cc *circuits.Compiler) error {
	for _, step := range code.Program.Steps {
		instr := step.Instr
		var wires [][]*circuits.Wire
		for _, in := range instr.In {
			w, err := cc.Wires(in.String(), in.Type.Bits)
			if err != nil {
				return err
			}
			wires = append(wires, w)
		}
		switch instr.Op {
		case Iadd, Uadd:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewAdder(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Isub, Usub:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewSubtractor(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Imult, Umult:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewMultiplier(cc, gen.Params.CircMultArrayTreshold,
				wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Idiv, Udiv:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}

			err = circuits.NewDivider(cc, wires[0], wires[1], o, nil)
			if err != nil {
				return err
			}

		case Imod, Umod:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}

			err = circuits.NewDivider(cc, wires[0], wires[1], nil, o)
			if err != nil {
				return err
			}

		case Slice:
			if !instr.In[1].Const {
				return fmt.Errorf("%s only constant index supported", instr.Op)
			}
			var from int
			switch val := instr.In[1].ConstValue.(type) {
			case int32:
				from = int(val)
			default:
				return fmt.Errorf("%s unsupported index type %T", instr.Op, val)
			}

			if !instr.In[2].Const {
				return fmt.Errorf("%s only constant index supported", instr.Op)
			}
			var to int
			switch val := instr.In[2].ConstValue.(type) {
			case int32:
				to = int(val)
			default:
				return fmt.Errorf("%s unsupported index type %T", instr.Op, val)
			}
			if from >= to {
				return fmt.Errorf("%s bounds out of range [%d:%d]",
					instr.Op, from, to)
			}
			o := make([]*circuits.Wire, instr.Out.Type.Bits)

			for bit := from; bit < to; bit++ {
				var w *circuits.Wire
				if bit < len(wires[0]) {
					w = wires[0][bit]
				} else {
					w = cc.ZeroWire()
				}
				o[bit-from] = w
			}
			err := cc.SetWires(instr.Out.String(), o)
			if err != nil {
				return err
			}

		case Ilt, Ult:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewLtComparator(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Ile, Ule:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewLeComparator(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Igt, Ugt:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewGtComparator(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Ige, Uge:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewGeComparator(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Eq:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewEqComparator(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Neq:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewNeqComparator(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

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
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
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
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			err = circuits.NewBitClrTest(cc, wires[0], index, o)
			if err != nil {
				return err
			}

		case And:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewLogicalAND(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Or:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewLogicalOR(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Band:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewBinaryAND(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Bclr:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewBinaryClear(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Bor:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewBinaryOR(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Bxor:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewBinaryXOR(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		case Mov:
			o := make([]*circuits.Wire, instr.Out.Type.Bits)

			for bit := 0; bit < instr.Out.Type.Bits; bit++ {
				var w *circuits.Wire
				if bit < len(wires[0]) {
					w = wires[0][bit]
				} else {
					w = cc.ZeroWire()
				}
				o[bit] = w
			}
			err := cc.SetWires(instr.Out.String(), o)
			if err != nil {
				return err
			}

		case Phi:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewMux(cc, wires[0], wires[1], wires[2], o)
			if err != nil {
				return err
			}

		case Ret:
			// Assign output wires.
			for _, wg := range wires {
				for _, w := range wg {
					o := circuits.NewWire()
					cc.ID(w, o)
					cc.Outputs = append(cc.Outputs, o)
				}
			}
			for _, o := range cc.Outputs {
				o.Output = true
			}

		case Circ:
			var circWires []*circuits.Wire

			// Flatten input wires.
			for idx, w := range wires {
				circWires = append(circWires, w...)
				for i := len(w); i < instr.Circ.Inputs[idx].Size; i++ {
					// Zeroes for unset input wires.
					zw := cc.ZeroWire()
					circWires = append(circWires, zw)
				}
			}

			// Flatten output wires.
			var circOut []*circuits.Wire

			for _, r := range instr.Ret {
				o, err := cc.Wires(r.String(), r.Type.Bits)
				if err != nil {
					return err
				}
				circOut = append(circOut, o...)
			}

			// Add intermediate wires.
			nint := instr.Circ.NumWires - len(circWires) - len(circOut)
			for i := 0; i < nint; i++ {
				circWires = append(circWires, circuits.NewWire())
			}

			// Append output wires.
			circWires = append(circWires, circOut...)

			// Add gates.
			for _, gate := range instr.Circ.Gates {
				switch gate.Op {
				case circuit.XOR:
					cc.AddGate(circuits.NewBinary(circuit.XOR,
						circWires[gate.Input0],
						circWires[gate.Input1],
						circWires[gate.Output]))
				case circuit.XNOR:
					cc.AddGate(circuits.NewBinary(circuit.XNOR,
						circWires[gate.Input0],
						circWires[gate.Input1],
						circWires[gate.Output]))
				case circuit.AND:
					cc.AddGate(circuits.NewBinary(circuit.AND,
						circWires[gate.Input0],
						circWires[gate.Input1],
						circWires[gate.Output]))
				case circuit.OR:
					cc.AddGate(circuits.NewBinary(circuit.OR,
						circWires[gate.Input0],
						circWires[gate.Input1],
						circWires[gate.Output]))
				case circuit.INV:
					cc.INV(circWires[gate.Input0], circWires[gate.Output])
				default:
					return fmt.Errorf("Unknown gate %s", gate)
				}
			}

		case Builtin:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = instr.Builtin(cc, wires[0], wires[1], o)
			if err != nil {
				return err
			}

		default:
			return fmt.Errorf("Block.Circuit: %s not implemented yet", instr.Op)
		}
	}

	return nil
}
