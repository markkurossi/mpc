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

const (
	verbose = false
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
	if verbose {
		fmt.Printf("Result: %s\n", result)
		result.Marshal(os.Stdout)
	}
}

func TestFullSubtractor(t *testing.T) {
	c := NewCompiler(1, 2, 2)
	NewFullSubtractor(c, c.Inputs[0], c.Inputs[1], c.Inputs[2],
		c.Outputs[0], c.Outputs[1])

	result := c.Compile()
	if verbose {
		fmt.Printf("Result: %s\n", result)
		result.Marshal(os.Stdout)
	}
}

func TestSub4(t *testing.T) {
	bits := 4

	// 2xbits inputs, bits+1 outputs
	c := NewCompiler(bits, bits, bits+1)

	bin := NewWire()
	NewHalfSubtractor(c, c.Inputs[0], c.Inputs[bits], c.Outputs[0], bin)

	for i := 1; i < bits; i++ {
		var bout *Wire
		if i+1 >= bits {
			bout = c.Outputs[bits]
		} else {
			bout = NewWire()
		}

		NewFullSubtractor(c, c.Inputs[i], c.Inputs[bits+i], bin, c.Outputs[i],
			bout)

		bin = bout
	}

	result := c.Compile()
	if verbose {
		fmt.Printf("Result: %s\n", result)
		result.Marshal(os.Stdout)
	}
}

func TestMultiply1(t *testing.T) {
	c := NewCompiler(1, 1, 2)

	err := NewMultiplier(c, c.Inputs[0:1], c.Inputs[1:2], c.Outputs)
	if err != nil {
		t.Error(err)
	}
}

func TestMultiply(t *testing.T) {
	bits := 64

	c := NewCompiler(bits, bits, bits*2)
	err := NewMultiplier(c, c.Inputs[0:bits], c.Inputs[bits:2*bits], c.Outputs)
	if err != nil {
		t.Error(err)
	}

	result := c.Compile()
	if verbose {
		fmt.Printf("Result: %s\n", result)
		result.Marshal(os.Stdout)
	}
}
