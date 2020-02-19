//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"

	"github.com/markkurossi/mpc/compiler/circuits"
)

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

		case Ilt, Ult:
			o, err := cc.Wires(instr.Out.String(), instr.Out.Type.Bits)
			if err != nil {
				return err
			}
			err = circuits.NewLtComparator(cc, wires[0], wires[1], o)
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

		case If, Jump:
			// Branch operations are no-ops in circuits.

		case Mov:
			var o []*circuits.Wire
			if instr.In[0].Const {
				// Create constant value bits.
				for bit := 0; bit < instr.Out.Type.Bits; bit++ {
					var w *circuits.Wire
					if bit < len(wires[0]) {
						w = wires[0][bit]
					} else {
						w = circuits.NewWire()
					}
					if instr.In[0].Bit(bit) {
						cc.One(w)
					} else {
						cc.Zero(w)
					}
					o = append(o, w)
				}
			} else {
				o = wires[0]
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
			cc.Outputs = nil // XXX check cc constructor
			for _, w := range wires {
				cc.Outputs = append(cc.Outputs, w...)
			}
			for _, o := range cc.Outputs {
				o.Output = true
			}

		default:
			return fmt.Errorf("%s.Circuit not implemented yet", instr.Op)
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
