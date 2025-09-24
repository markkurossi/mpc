//
// circuits_test.go
//
// Copyright (c) 2019-2025 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/types"
)

const (
	verbose = false
)

var (
	params = utils.NewParams()
	calloc = NewAllocator()
)

func makeWires(count int, output bool) []*Wire {
	var result []*Wire
	for i := 0; i < count; i++ {
		w := calloc.Wire()
		w.SetOutput(output)
		result = append(result, w)
	}
	return result
}

func NewIO(size int, name string) circuit.IO {
	return circuit.IO{
		circuit.IOArg{
			Name: name,
			Type: types.Info{
				Type:       types.TUint,
				IsConcrete: true,
				Bits:       types.Size(size),
			},
		},
	}
}

func TestAdd4(t *testing.T) {
	bits := 4

	// 2xbits inputs, bits+1 outputs
	inputs := makeWires(bits*2, false)
	outputs := makeWires(bits+1, true)
	c, err := NewCompiler(params, calloc, NewIO(bits*2, "in"),
		NewIO(bits+1, "out"), inputs, outputs)
	if err != nil {
		t.Fatalf("NewCompiler: %s", err)
	}

	cin := calloc.Wire()
	NewHalfAdder(c, inputs[0], inputs[bits], outputs[0], cin)

	for i := 1; i < bits; i++ {
		var cout *Wire
		if i+1 >= bits {
			cout = outputs[bits]
		} else {
			cout = calloc.Wire()
		}

		NewFullAdder(c, inputs[i], inputs[bits+i], cin, outputs[i], cout)

		cin = cout
	}

	result := c.Compile()
	if verbose {
		fmt.Printf("Result: %s\n", result)
		result.Marshal(os.Stdout)
	}
}

func TestFullSubtractor(t *testing.T) {
	inputs := makeWires(1+2, false)
	outputs := makeWires(2, true)
	c, err := NewCompiler(params, calloc, NewIO(1+2, "in"), NewIO(2, "out"),
		inputs, outputs)
	if err != nil {
		t.Fatalf("NewCompiler: %s", err)
	}

	NewFullSubtractor(c, inputs[0], inputs[1], inputs[2],
		outputs[0], outputs[1])

	result := c.Compile()
	if verbose {
		fmt.Printf("Result: %s\n", result)
		result.Marshal(os.Stdout)
	}
}

func TestMultiply1(t *testing.T) {
	inputs := makeWires(2, false)
	outputs := makeWires(2, true)
	c, err := NewCompiler(params, calloc, NewIO(2, "in"), NewIO(2, "out"),
		inputs, outputs)
	if err != nil {
		t.Fatalf("NewCompiler: %s", err)
	}

	err = NewMultiplier(c, 0, inputs[0:1], inputs[1:2], outputs)
	if err != nil {
		t.Error(err)
	}
}

func TestMultiply(t *testing.T) {
	bits := 64

	inputs := makeWires(bits*2, false)
	outputs := makeWires(bits*2, true)

	c, err := NewCompiler(params, calloc, NewIO(bits*2, "in"),
		NewIO(bits*2, "out"), inputs, outputs)
	if err != nil {
		t.Fatalf("NewCompiler: %s", err)
	}

	err = NewMultiplier(c, 0, inputs[0:bits], inputs[bits:2*bits], outputs)
	if err != nil {
		t.Error(err)
	}

	result := c.Compile()
	if verbose {
		fmt.Printf("Result: %s\n", result)
		result.Marshal(os.Stdout)
	}
}

var cmpTests = []struct {
	t      types.Info
	x      string
	y      string
	name   string
	cmp    func(cc *Compiler, x, y, r []*Wire) error
	result bool
}{
	// >
	{
		t:      types.Int32,
		x:      "43",
		y:      "42",
		name:   ">",
		cmp:    NewIntGtComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "43",
		y:      "43",
		name:   ">",
		cmp:    NewIntGtComparator,
		result: false,
	},
	{
		t:      types.Int32,
		x:      "-42",
		y:      "-43",
		name:   ">",
		cmp:    NewIntGtComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "-43",
		y:      "-43",
		name:   ">",
		cmp:    NewIntGtComparator,
		result: false,
	},
	{
		t:      types.Int32,
		x:      "0",
		y:      "-1",
		name:   ">",
		cmp:    NewIntGtComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "-1",
		y:      "0",
		name:   ">",
		cmp:    NewIntGtComparator,
		result: false,
	},
	// >=
	{
		t:      types.Int32,
		x:      "43",
		y:      "42",
		name:   ">=",
		cmp:    NewIntGeComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "43",
		y:      "43",
		name:   ">=",
		cmp:    NewIntGeComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "43",
		y:      "44",
		name:   ">=",
		cmp:    NewIntGeComparator,
		result: false,
	},
	{
		t:      types.Int32,
		x:      "-42",
		y:      "-43",
		name:   ">=",
		cmp:    NewIntGeComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "-43",
		y:      "-43",
		name:   ">=",
		cmp:    NewIntGeComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "-44",
		y:      "-43",
		name:   ">=",
		cmp:    NewIntGeComparator,
		result: false,
	},
	{
		t:      types.Int32,
		x:      "0",
		y:      "-1",
		name:   ">=",
		cmp:    NewIntGeComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "-1",
		y:      "0",
		name:   ">=",
		cmp:    NewIntGeComparator,
		result: false,
	},
	// <
	{
		t:      types.Int32,
		x:      "42",
		y:      "43",
		name:   "<",
		cmp:    NewIntLtComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "43",
		y:      "43",
		name:   "<",
		cmp:    NewIntLtComparator,
		result: false,
	},
	{
		t:      types.Int32,
		x:      "-43",
		y:      "-42",
		name:   "<",
		cmp:    NewIntLtComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "-43",
		y:      "-43",
		name:   "<",
		cmp:    NewIntLtComparator,
		result: false,
	},
	{
		t:      types.Int32,
		x:      "-1",
		y:      "0",
		name:   "<",
		cmp:    NewIntLtComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "0",
		y:      "-1",
		name:   "<",
		cmp:    NewIntLtComparator,
		result: false,
	},
	// <=
	{
		t:      types.Int32,
		x:      "42",
		y:      "43",
		name:   "<=",
		cmp:    NewIntLeComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "43",
		y:      "43",
		name:   "<=",
		cmp:    NewIntLeComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "44",
		y:      "43",
		name:   "<=",
		cmp:    NewIntLeComparator,
		result: false,
	},
	{
		t:      types.Int32,
		x:      "-43",
		y:      "-42",
		name:   "<=",
		cmp:    NewIntLeComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "-43",
		y:      "-43",
		name:   "<=",
		cmp:    NewIntLeComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "-43",
		y:      "-44",
		name:   "<=",
		cmp:    NewIntLeComparator,
		result: false,
	},
	{
		t:      types.Int32,
		x:      "-1",
		y:      "0",
		name:   "<=",
		cmp:    NewIntLeComparator,
		result: true,
	},
	{
		t:      types.Int32,
		x:      "0",
		y:      "-1",
		name:   "<=",
		cmp:    NewIntLeComparator,
		result: false,
	},
}

func TestComparison(t *testing.T) {
	for idx, test := range cmpTests {
		// Create comparison circuit.

		bits := int(test.t.Bits)

		inputs := makeWires(int(bits*2), false)
		outputs := makeWires(1, true)

		cc, err := NewCompiler(params, calloc,
			circuit.IO{
				circuit.IOArg{
					Name: "x",
					Type: test.t,
				},
				circuit.IOArg{
					Name: "y",
					Type: test.t,
				},
			},
			circuit.IO{
				circuit.IOArg{
					Name: "r",
					Type: types.Bool,
				},
			}, inputs, outputs)
		if err != nil {
			t.Fatalf("NewCompiler: %s", err)
		}

		err = test.cmp(cc, inputs[0:bits], inputs[bits:], outputs)
		if err != nil {
			t.Fatal(err)
		}

		circ := cc.Compile()

		// Evaluate circuit.

		ioArg := circuit.IOArg{
			Type: test.t,
		}

		x, err := ioArg.Parse([]string{test.x})
		if err != nil {
			t.Fatal(err)
		}
		y, err := ioArg.Parse([]string{test.y})
		if err != nil {
			t.Fatal(err)
		}

		result, err := circ.Compute([]*big.Int{x, y})
		if err != nil {
			t.Fatal(err)
		}

		resultBool := result[0].Bit(0) == 1
		if resultBool != test.result {
			t.Errorf("comparision %d failed: %v %v %v = %v, expected %v\n",
				idx, test.x, test.name, test.y, resultBool, test.result)
		}
	}
}
