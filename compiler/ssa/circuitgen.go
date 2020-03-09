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
)

func (gen *Generator) DefineConstants(cc *circuits.Compiler) error {
	var consts []Variable
	for _, c := range gen.constants {
		consts = append(consts, c.Const)
	}
	sort.Slice(consts, func(i, j int) bool {
		return strings.Compare(consts[i].Name, consts[j].Name) == -1
	})

	if len(consts) > 0 && gen.verbose {
		fmt.Printf("Defining constants:\n")
	}
	for _, c := range consts {
		msg := fmt.Sprintf(" - %v(%d)\t", c, c.Type.Bits)

		var wires []*circuits.Wire
		for bit := 0; bit < c.Type.Bits; bit++ {
			w := circuits.NewWire()
			if c.Bit(bit) {
				msg += "1"
				cc.One(w)
			} else {
				msg += "0"
				cc.Zero(w)
			}
			wires = append(wires, w)
		}
		if gen.verbose {
			fmt.Printf("%s\n", msg)
		}

		err := cc.SetWires(c.String(), wires)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Block) Circuit(gen *Generator, cc *circuits.Compiler) error {
	if b.Processed {
		return nil
	}
	// Check that all from blocks have been processed.
	for _, from := range b.From {
		if !from.Processed {
			return nil
		}
	}
	b.Processed = true

	for _, instr := range b.Instr {
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
			err = circuits.NewMultiplier(cc, wires[0], wires[1], o)
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
			err = circuits.NewLtComparator(cc, wires[1], wires[0], o)
			if err != nil {
				return err
			}

		case Ige, Uge:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewLeComparator(cc, wires[1], wires[0], o)
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

		case If, Jump:
			// Branch operations are no-ops in circuits.

		case Mov:
			o := make([]*circuits.Wire, instr.Out.Type.Bits)

			for bit := 0; bit < instr.Out.Type.Bits; bit++ {
				var w *circuits.Wire
				if bit < len(wires[0]) {
					w = wires[0][bit]
				} else {
					w = circuits.NewWire()
					// XXX Types, sign bit expansion on signed values.
					cc.Zero(w)
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
			for _, w := range wires {
				cc.Outputs = append(cc.Outputs, w...)
			}
			for _, o := range cc.Outputs {
				o.Output = true
			}

		case Circ:
			var circWires []*circuits.Wire

			// Flatten input wires.
			for _, w := range wires {
				circWires = append(circWires, w...)
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
						circWires[gate.Inputs[0]],
						circWires[gate.Inputs[1]],
						circWires[gate.Outputs[0]]))
				case circuit.AND:
					cc.AddGate(circuits.NewBinary(circuit.AND,
						circWires[gate.Inputs[0]],
						circWires[gate.Inputs[1]],
						circWires[gate.Outputs[0]]))
				case circuit.OR:
					cc.AddGate(circuits.NewBinary(circuit.OR,
						circWires[gate.Inputs[0]],
						circWires[gate.Inputs[1]],
						circWires[gate.Outputs[0]]))
				case circuit.INV:
					cc.AddGate(circuits.NewINV(
						circWires[gate.Inputs[0]],
						circWires[gate.Outputs[0]]))
				default:
					return fmt.Errorf("Unknown gate %s", gate)
				}
			}

		default:
			return fmt.Errorf("Block.Circuit: %s not implemented yet", instr.Op)
		}
	}

	if b.Branch != nil {
		err := b.Branch.Circuit(gen, cc)
		if err != nil {
			return err
		}
	}
	if b.Next != nil {
		err := b.Next.Circuit(gen, cc)
		if err != nil {
			return err
		}
	}

	return nil
}
