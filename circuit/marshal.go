//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	MAGIC = 0x63726300 // crc0
)

func (c *Circuit) Marshal(out io.Writer) error {
	var data = []interface{}{
		uint32(MAGIC),
		uint32(c.NumGates),
		uint32(c.NumWires),
		uint32(len(c.Inputs)),
		uint32(len(c.Outputs)),
	}
	for _, input := range c.Inputs {
		data = append(data, uint32(input.Size))
	}
	for _, output := range c.Outputs {
		data = append(data, uint32(output.Size))
	}
	for _, v := range data {
		if err := binary.Write(out, binary.BigEndian, v); err != nil {
			return err
		}
	}

	for _, g := range c.Gates {
		switch g.Op {
		case XOR, XNOR, AND, OR:
			data = []interface{}{
				byte(g.Op),
				uint32(g.Input0), uint32(g.Input1), uint32(g.Output),
			}

		case INV:
			data = []interface{}{
				byte(g.Op),
				uint32(g.Input0), uint32(g.Output),
			}
		default:
			return fmt.Errorf("unsupported gate type %s", g.Op)
		}
		for _, v := range data {
			if err := binary.Write(out, binary.BigEndian, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Circuit) MarshalBristol(out io.Writer) {
	fmt.Fprintf(out, "%d %d\n", c.NumGates, c.NumWires)
	fmt.Fprintf(out, "%d", len(c.Inputs))
	for _, input := range c.Inputs {
		fmt.Fprintf(out, " %d", input.Size)
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%d", len(c.Outputs))
	for _, ret := range c.Outputs {
		fmt.Fprintf(out, " %d", ret.Size)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out)

	for _, g := range c.Gates {
		fmt.Fprintf(out, "%d 1", len(g.Inputs()))
		for _, w := range g.Inputs() {
			fmt.Fprintf(out, " %d", w)
		}
		fmt.Fprintf(out, " %d", g.Output)
		fmt.Fprintf(out, " %s\n", g.Op)
	}
}
