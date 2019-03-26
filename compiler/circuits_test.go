//
// circuits_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"fmt"
	"os"
	"testing"
)

func TestAdd4(t *testing.T) {
	bits := 4

	// 2xbits inputs, bits+1 outputs
	c := NewCompiler(bits, bits, bits+1)

	cin := NewWire()
	NewHalfAdder(c, c.Inputs[0], c.Inputs[bits], c.Outputs[0], cin)

	for i := 1; i < bits; i++ {
		var cout *Wire
		if i+1 >= bits {
			cout = c.Outputs[bits]
		} else {
			cout = NewWire()
		}

		NewFullAdder(c, c.Inputs[i], c.Inputs[bits+i], cin, c.Outputs[i], cout)

		cin = cout
	}

	result := c.Compile()
	fmt.Printf("Result: %s\n", result)
	result.Marshal(os.Stdout)
}
