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

		case Jump:
			// nop

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
