//
// main.go
//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"math/big"
	"unicode"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/types"
)

func printResults(results []*big.Int, outputs circuit.IO) {
	for idx, result := range results {
		if outputs == nil {
			fmt.Printf("Result[%d]: %v\n", idx, result)
			fmt.Printf("Result[%d]: 0b%s\n", idx, result.Text(2))
			bytes := result.Bytes()
			if len(bytes) == 0 {
				bytes = []byte{0}
			}
			fmt.Printf("Result[%d]: 0x%x\n", idx, bytes)
		} else {
			fmt.Printf("Result[%d]: %s\n", idx,
				printResult(result, outputs[idx], false))
		}
	}
}

func printResult(result *big.Int, output circuit.IOArg, short bool) string {
	var str string

	switch output.Type.Type {
	case types.TString:
		mask := big.NewInt(0xff)

		for i := 0; i < int(output.Type.Bits)/8; i++ {
			tmp := new(big.Int).Rsh(result, uint(i*8))
			r := rune(tmp.And(tmp, mask).Uint64())
			if unicode.IsPrint(r) {
				str += string(r)
			} else {
				str += fmt.Sprintf("\\u%04x", r)
			}
		}

	case types.TUint, types.TInt:
		if output.Type.Type == types.TInt {
			bits := int(output.Type.Bits)
			if result.Bit(bits-1) == 1 {
				// Negative number.
				tmp := new(big.Int)
				tmp.SetBit(tmp, bits, 1)
				result.Sub(tmp, result)
				result.Neg(result)
			}
		}

		bytes := result.Bytes()
		if len(bytes) == 0 {
			bytes = []byte{0}
		}
		if short {
			str = fmt.Sprintf("%v", result)
		} else if output.Type.Bits <= 64 {
			str = fmt.Sprintf("0x%x\t%v", bytes, result)
		} else {
			str = fmt.Sprintf("0x%x", bytes)
		}

	case types.TBool:
		str = fmt.Sprintf("%v", result.Uint64() != 0)

	case types.TArray:
		count := int(output.Type.ArraySize)
		elSize := int(output.Type.ElementType.Bits)

		mask := new(big.Int)
		for i := 0; i < elSize; i++ {
			mask.SetBit(mask, i, 1)
		}

		var hexString bool
		if output.Type.ElementType.Type == types.TUint &&
			output.Type.ElementType.Bits == 8 {
			hexString = true
		}
		if !hexString {
			str = "["
		}
		for i := 0; i < count; i++ {
			r := new(big.Int).Rsh(result, uint(i*elSize))
			r = r.And(r, mask)

			if hexString {
				str += fmt.Sprintf("%02x", r.Int64())
			} else {
				if i > 0 {
					str += " "
				}
				str += printResult(r, circuit.IOArg{
					Type: *output.Type.ElementType,
				}, true)
			}
		}
		if !hexString {
			str += "]"
		}

	default:
		str = fmt.Sprintf("%v (%s)", result, output.Type)
	}

	return str
}
