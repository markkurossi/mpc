//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
	"math/big"

	"github.com/markkurossi/mpc/types"
)

// IOArg describes circuit input argument.
type IOArg struct {
	Name     string
	Type     types.Info
	Compound IO
}

func (io IOArg) String() string {
	if len(io.Compound) > 0 {
		return io.Compound.String()
	}

	if len(io.Name) > 0 {
		return io.Name + ":" + io.Type.String()
	}
	return io.Type.String()
}

// Parse parses the I/O argument from the input string values.
func (io IOArg) Parse(inputs []string) (*big.Int, error) {
	result := new(big.Int)

	if len(io.Compound) == 0 {
		if len(inputs) != 1 {
			return nil,
				fmt.Errorf("invalid amount of arguments, got %d, expected 1",
					len(inputs))
		}

		switch io.Type.Type {
		case types.TInt, types.TUint:
			_, ok := result.SetString(inputs[0], 0)
			if !ok {
				return nil, fmt.Errorf("invalid input: %s", inputs[0])
			}

		case types.TBool:
			switch inputs[0] {
			case "0", "f", "false":
			case "1", "t", "true":
				result.SetInt64(1)
			default:
				return nil, fmt.Errorf("invalid bool constant: %s", inputs[0])
			}

		case types.TArray:
			count := int(io.Type.ArraySize)
			elSize := int(io.Type.ElementType.Bits)

			val := new(big.Int)
			_, ok := val.SetString(inputs[0], 0)
			if !ok {
				return nil, fmt.Errorf("invalid input: %s", inputs[0])
			}

			valElCount := val.BitLen() / elSize
			if val.BitLen()%elSize != 0 {
				valElCount++
			}
			if valElCount > count {
				return nil, fmt.Errorf("too many values for input: %s",
					inputs[0])
			}
			pad := count - valElCount
			val.Lsh(val, uint(pad*elSize))

			mask := new(big.Int)
			for i := 0; i < elSize; i++ {
				mask.SetBit(mask, i, 1)
			}

			for i := 0; i < count; i++ {
				next := new(big.Int).Rsh(val, uint((count-i-1)*elSize))
				next = next.And(next, mask)

				next.Lsh(next, uint(i*elSize))
				result.Or(result, next)
			}

		default:
			return nil, fmt.Errorf("unsupported input type: %s", io.Type)
		}

		return result, nil
	}
	if len(inputs) != len(io.Compound) {
		return nil,
			fmt.Errorf("invalid amount of arguments, got %d, expected %d",
				len(inputs), len(io.Compound))
	}

	var offset int

	for idx, arg := range io.Compound {
		input, err := arg.Parse(inputs[idx : idx+1])
		if err != nil {
			return nil, err
		}

		input.Lsh(input, uint(offset))
		result.Or(result, input)

		offset += int(arg.Type.Bits)
	}
	return result, nil
}

// InputSizes computes the bit sizes of the input arguments. This is
// used for parametrized main() when the program is instantiated based
// on input sizes.
func InputSizes(inputs []string) ([]int, error) {
	var result []int

	for _, input := range inputs {
		switch input {
		case "0", "f", "false", "1", "t", "true":
			result = append(result, 1)

		default:
			val := new(big.Int)
			_, ok := val.SetString(input, 0)
			if !ok {
				return nil, fmt.Errorf("invalid input: %s", input)
			}
			result = append(result, val.BitLen())
		}
	}

	return result, nil
}
