//
// circuits_test.go
//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package circuits

import (
	"math/big"
	"testing"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/types"
)

var gmwDividerTests = []struct {
	t types.Info
	a string
	b string
	q string
	r string
}{
	{
		t: types.Int32,
		a: "1",
		b: "1",
		q: "1",
		r: "0",
	},
	{
		t: types.Int32,
		a: "2",
		b: "1",
		q: "2",
		r: "0",
	},
	{
		t: types.Int32,
		a: "4",
		b: "2",
		q: "2",
		r: "0",
	},
	{
		t: types.Uint64,
		a: "4611686018427387904",
		b: "4611686018427387904",
		q: "1",
		r: "0",
	},
	{
		t: types.Uint64,
		a: "4611686018427387904",
		b: "1",
		q: "4611686018427387904",
		r: "0",
	},
	{
		t: types.Uint64,
		a: "9223372036854775807",
		b: "9223372036854775807",
		q: "1",
		r: "0",
	},
	{
		t: types.Uint64,
		a: "9223372036854775807",
		b: "1",
		q: "9223372036854775807",
		r: "0",
	},
	{
		t: types.Uint32,
		a: "6",
		b: "2",
		q: "3",
		r: "0",
	},
	{
		t: types.Uint32,
		a: "7",
		b: "2",
		q: "3",
		r: "1",
	},
	{
		t: types.Uint32,
		a: "2",
		b: "4",
		q: "0",
		r: "2",
	},
	{
		t: types.Uint32,
		a: "1",
		b: "100",
		q: "0",
		r: "1",
	},
	{
		t: types.Uint32,
		a: "0",
		b: "5",
		q: "0",
		r: "0",
	},
	{
		t: types.Uint32,
		a: "63",
		b: "7",
		q: "9",
		r: "0",
	},
	{
		t: types.Uint32,
		a: "56",
		b: "7",
		q: "8",
		r: "0",
	},
	{
		t: types.Uint32,
		a: "128",
		b: "2",
		q: "64",
		r: "0",
	},
}

func TestGMWDivider(t *testing.T) {
	params := utils.NewParams()
	params.Target = utils.TargetGMW

	for idx, test := range gmwDividerTests {
		// Create circuit.

		bits := int(test.t.Bits)

		iWires := makeWires(bits*2, false)
		oWires := makeWires(bits*2, true)

		inputs := circuit.IO{
			circuit.IOArg{
				Name: "a",
				Type: test.t,
			},
			circuit.IOArg{
				Name: "b",
				Type: test.t,
			},
		}
		outputs := circuit.IO{
			circuit.IOArg{
				Name: "r",
				Type: test.t,
			},
			circuit.IOArg{
				Name: "q",
				Type: test.t,
			},
		}

		cc, err := NewCompiler(params, calloc, inputs, outputs, iWires, oWires)
		if err != nil {
			t.Fatalf("NewCompiler: %v", err)
		}
		err = NewUDivider(cc, iWires[:bits], iWires[bits:],
			oWires[:bits], oWires[bits:])
		if err != nil {
			t.Fatal(err)
		}

		circ := cc.Compile()
		t.Logf("test-%v: xor=%v, !xor=%v\n", idx,
			circ.Stats[circuit.XOR]+circ.Stats[circuit.XNOR],
			circ.Stats[circuit.AND]+circ.Stats[circuit.INV])

		// Evaluate circuit.
		a, err := inputs[0].Parse([]string{test.a})
		if err != nil {
			t.Fatal(err)
		}
		b, err := inputs[1].Parse([]string{test.b})
		if err != nil {
			t.Fatal(err)
		}
		result, err := circ.Compute([]*big.Int{a, b})
		if err != nil {
			t.Fatal(err)
		}

		q, ok := new(big.Int).SetString(test.q, 10)
		if !ok {
			t.Fatalf("test-%v: invalid q: %v", idx, test.q)
		}
		r, ok := new(big.Int).SetString(test.r, 10)
		if !ok {
			t.Fatalf("test-%v: invalid r: %v", idx, test.r)
		}

		if q.Cmp(result[0]) != 0 {
			t.Errorf("quotient mismatch: got %v, expected %v", q, result[0])
		}
		if r.Cmp(result[1]) != 0 {
			t.Errorf("remainder mismatch: got %v, expected %v", r, result[1])
		}
		t.Logf("test-%v: %v / %v: q=%v, r=%v\n", idx, a, b, q, r)
	}
}
