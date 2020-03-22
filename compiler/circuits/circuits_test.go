//
// circuits_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"
	"os"
	"testing"
)

const (
	verbose = false
)

func makeWires(count int) []*Wire {
	var result []*Wire
	for i := 0; i < count; i++ {
		result = append(result, NewWire())
	}
	return result
}

func TestAdd4(t *testing.T) {
	bits := 4

	// 2xbits inputs, bits+1 outputs
	c, err := NewCompiler(NewIO(bits), NewIO(bits), NewIO(bits+1))
	if err != nil {
		t.Fatalf("NewCompiler: %s", err)
	}

	outputs := makeWires(bits + 1)

	cin := NewWire()
	NewHalfAdder(c, c.Inputs[0], c.Inputs[bits], outputs[0], cin)

	for i := 1; i < bits; i++ {
		var cout *Wire
		if i+1 >= bits {
			cout = outputs[bits]
		} else {
			cout = NewWire()
		}

		NewFullAdder(c, c.Inputs[i], c.Inputs[bits+i], cin, outputs[i], cout)

		cin = cout
	}

	result := c.Compile()
	if verbose {
		fmt.Printf("Result: %s\n", result)
		result.Marshal(os.Stdout)
	}
}

func TestFullSubtractor(t *testing.T) {
	c, err := NewCompiler(NewIO(1), NewIO(2), NewIO(2))
	if err != nil {
		t.Fatalf("NewCompiler: %s", err)
	}

	outputs := makeWires(2)

	NewFullSubtractor(c, c.Inputs[0], c.Inputs[1], c.Inputs[2],
		outputs[0], outputs[1])

	result := c.Compile()
	if verbose {
		fmt.Printf("Result: %s\n", result)
		result.Marshal(os.Stdout)
	}
}

func TestMultiply1(t *testing.T) {
	c, err := NewCompiler(NewIO(1), NewIO(1), NewIO(2))
	if err != nil {
		t.Fatalf("NewCompiler: %s", err)
	}

	outputs := makeWires(2)

	err = NewMultiplier(c, c.Inputs[0:1], c.Inputs[1:2], outputs)
	if err != nil {
		t.Error(err)
	}
}

func TestMultiply(t *testing.T) {
	bits := 64

	c, err := NewCompiler(NewIO(bits), NewIO(bits), NewIO(bits*2))
	if err != nil {
		t.Fatalf("NewCompiler: %s", err)
	}

	outputs := makeWires(bits * 2)

	err = NewMultiplier(c, c.Inputs[0:bits], c.Inputs[bits:2*bits], outputs)
	if err != nil {
		t.Error(err)
	}

	result := c.Compile()
	if verbose {
		fmt.Printf("Result: %s\n", result)
		result.Marshal(os.Stdout)
	}
}
