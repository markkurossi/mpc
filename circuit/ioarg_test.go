//
// Copyright (c) 2023-2025 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"math/big"
	"testing"

	"github.com/markkurossi/mpc/types"
)

var inputSizeTests = []struct {
	strings []string
	values  []interface{}
	results []int
}{
	{
		strings: []string{
			"0", "f", "false", "1", "t", "true",
		},
		results: []int{
			1, 1, 1, 1, 1, 1,
		},
	},
	{
		strings: []string{
			"0xdeadbeef", "255",
		},
		values: []interface{}{
			uint32(0xdeadbeef), uint8(255),
		},
		results: []int{
			32, 8,
		},
	},
	{
		strings: []string{
			"0xdeadbeef", "5",
		},
		values: []interface{}{
			uint32(0xdeadbeef), uint32(5),
		},
		results: []int{
			32, 3,
		},
	},
	{
		strings: []string{
			"0x0", "0x00", "0x000", "0x0000",
		},
		results: []int{
			4, 8, 12, 16,
		},
	},
	{
		strings: []string{
			"42x00", "0x00",
		},
		values: []interface{}{
			[]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00,
			},
			[]byte{0x00},
		},
		results: []int{
			336, 8,
		},
	},
	{
		strings: []string{
			"_", "255",
		},
		values: []interface{}{
			nil, uint8(255),
		},
		results: []int{
			0, 8,
		},
	},
}

func TestInputSizes(t *testing.T) {
	for idx, test := range inputSizeTests {
		sizes, err := InputSizes(test.strings)
		if err != nil {
			t.Errorf("t%v: InputSizes(%v) failed: %v", idx, test.strings, err)
			continue
		}
		if len(sizes) != len(test.results) {
			t.Errorf("t%v: unexpected # of sizes: got %v, expected %v",
				idx, len(sizes), len(test.results))
			continue
		}
		for i := 0; i < len(sizes); i++ {
			if sizes[i] != test.results[i] {
				t.Errorf("t%v: sizes[%v]=%v, expected %v",
					idx, i, sizes[i], test.results[i])
			}
		}

		if len(test.values) == 0 {
			continue
		}
		sizes, err = Sizes(test.values)
		if err != nil {
			t.Errorf("t%v: Sizes(%v) failed: %v", idx, test.values, err)
			continue
		}
		if len(sizes) != len(test.results) {
			t.Errorf("t%v: unexpected # of sizes: got %v, expected %v",
				idx, len(sizes), len(test.results))
			continue
		}
		for i := 0; i < len(sizes); i++ {
			if sizes[i] != test.results[i] {
				t.Errorf("t%v: sizes[%v]=%v, expected %v",
					idx, i, sizes[i], test.results[i])
			}
		}
	}
}

var valuesTests = []struct {
	ioarg   IOArg
	strings []string
	values  []interface{}
}{
	{
		ioarg: IOArg{
			Name: "bool",
			Type: types.Bool,
		},
		strings: []string{
			"true",
		},
		values: []interface{}{
			true,
		},
	},
	{
		ioarg: IOArg{
			Name: "int",
			Type: types.Int32,
		},
		strings: []string{
			"0x11121314",
		},
		values: []interface{}{
			int32(0x11121314),
		},
	},
	{
		ioarg: IOArg{
			Name: "[]byte",
			Type: types.Info{
				Type:        types.TArray,
				ArraySize:   8,
				ElementType: &types.Byte,
			},
		},
		strings: []string{
			"0xa0a1a2a3a4a5a6a7",
		},
		values: []interface{}{
			[]byte{
				0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7,
			},
		},
	},
	{
		ioarg: IOArg{
			Name: "G",
			Compound: IO{
				IOArg{
					Name: "arg0",
					Type: types.Uint32,
				},
				IOArg{
					Name: "key",
					Type: types.Info{
						Type:        types.TArray,
						Bits:        8 * 8,
						ArraySize:   8,
						ElementType: &types.Byte,
					},
				},
				IOArg{
					Name: "mem",
					Type: types.Info{
						Type:        types.TSlice,
						ElementType: &types.Byte,
					},
				},
				IOArg{
					Name: "arg1",
					Type: types.Uint32,
				},
			},
		},
		strings: []string{
			"0x21222324",
			"0xa0a1a2a3a4a5a6a7",
			"0",
			"0x31323334",
		},
		values: []interface{}{
			uint32(0x21222324),
			[]byte{
				0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7,
			},
			nil,
			uint32(0x31323334),
		},
	},
}

func TestValues(t *testing.T) {
	var vv *big.Int
	for idx, test := range valuesTests {
		sv, err := test.ioarg.Parse(test.strings)
		if err != nil {
			t.Errorf("t%v: Parse failed: %s", idx, err)
			continue
		}
		vv, err = test.ioarg.Set(nil, test.values)
		if err != nil {
			t.Errorf("t%v: Set failed: %s", idx, err)
			continue
		}
		if sv.Cmp(vv) != 0 {
			t.Errorf("t%v: %v != %v", idx, sv.Text(16), vv.Text(16))
			t.Errorf("t%v: str: %v", idx, sv.Text(16))
			t.Errorf("t%v: val: %v", idx, vv.Text(16))
		}
	}
}
