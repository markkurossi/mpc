//
// main.go
//
// Copyright (c) 2019-2025 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"

	"github.com/markkurossi/mpc/circuit"
)

func dumpObjects(files []string) error {
	for _, file := range files {
		if !circuit.IsFilename(file) {
			continue
		}
		c, err := circuit.Parse(file)
		if err != nil {
			return err
		}
		fmt.Printf("%s:\n", file)
		fmt.Printf(" - inputs:\n")
		for idx, input := range c.Inputs {
			fmt.Printf("   %v: %v\n", idx, input)
		}
		fmt.Printf(" - outputs:\n")
		for idx, output := range c.Outputs {
			fmt.Printf("   %v: %v\n", idx, output)
		}
		fmt.Printf(" - gates  : %v\n",
			c.Stats[circuit.XOR]+c.Stats[circuit.XNOR]+
				c.Stats[circuit.AND]+c.Stats[circuit.OR]+c.Stats[circuit.INV])

		fmt.Printf("   - XOR  : %v\n", c.Stats[circuit.XOR])
		fmt.Printf("   - XNOR : %v\n", c.Stats[circuit.XNOR])
		fmt.Printf("   - AND  : %v\n", c.Stats[circuit.AND])
		fmt.Printf("   - OR   : %v\n", c.Stats[circuit.OR])
		fmt.Printf("   - INV  : %v\n", c.Stats[circuit.INV])
		fmt.Printf("   - #xor : %v\n",
			c.Stats[circuit.XOR]+c.Stats[circuit.XNOR])
		fmt.Printf("   - #!xor: %v\n",
			c.Stats[circuit.AND]+c.Stats[circuit.OR]+c.Stats[circuit.INV])
		fmt.Printf(" - wires  : %v\n", c.NumWires)
		fmt.Printf(" - Cost   : %v\n", c.Cost())
	}

	return nil
}
