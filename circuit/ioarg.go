//
// Copyright (c) 2019-2025 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/markkurossi/mpc/types"
)

// IO specifies circuit input and output arguments.
type IO []IOArg

var (
	reHexInput = regexp.MustCompilePOSIX(`^([[:digit:]]+)x([[:xdigit:]]*)$`)
)

// Size computes the size of the circuit input and output arguments in
// bits.
func (io IO) Size() int {
	var sum int
	for _, a := range io {
		sum += int(a.Type.Bits)
	}
	return sum
}

func (io IO) String() string {
	var str = ""
	for i, a := range io {
		if i > 0 {
			str += ", "
		}
		if len(a.Name) > 0 {
			str += a.Name + ":"
		}
		str += a.Type.String()
	}
	return str
}

// Split splits the value into separate I/O arguments.
func (io IO) Split(in *big.Int) []*big.Int {
	var result []*big.Int
	var bit int
	for _, arg := range io {
		r := big.NewInt(0)
		for i := 0; i < int(arg.Type.Bits); i++ {
			if in.Bit(bit) == 1 {
				r = big.NewInt(0).SetBit(r, i, 1)
			}
			bit++
		}
		result = append(result, r)
	}
	return result
}

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

// Len returns the number of values the IOArg takes.
func (io IOArg) Len() int {
	if len(io.Compound) == 0 {
		return 1
	}
	return len(io.Compound)
}

// Set sets the I/O argument from the input values.
func (io IOArg) Set(result *big.Int, inputs []interface{}) (*big.Int, error) {
	if result == nil {
		result = new(big.Int)
	} else {
		result.SetInt64(0)
	}
	_, err := io.set(result, inputs, 0)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (io IOArg) set(result *big.Int, inputs []interface{}, ofs int) (
	int, error) {

	if len(io.Compound) > 0 {
		if len(inputs) != len(io.Compound) {
			return ofs,
				fmt.Errorf("invalid amount of arguments, got %d, expected %d",
					len(inputs), len(io.Compound))
		}
		var err error
		for idx, arg := range io.Compound {
			ofs, err = arg.set(result, inputs[idx:idx+1], ofs)
			if err != nil {
				return ofs, err
			}
		}
		return ofs, nil
	}

	if len(inputs) != 1 {
		return ofs,
			fmt.Errorf("invalid amount of arguments, got %d, expected 1",
				len(inputs))
	}

	switch io.Type.Type {
	case types.TInt, types.TUint:
		return setInt(io.Type, result, inputs[0], ofs)

	case types.TBool:
		return setBool(result, inputs[0], ofs)

	case types.TArray:
		count := int(io.Type.ArraySize)
		if count == 0 {
			// Handle empty types.TArray arguments.
			return ofs, nil
		}
		c, _, err := setArray(io.Type, result, inputs[0], ofs)
		if err != nil {
			return ofs, err
		}
		if c > count {
			return ofs, fmt.Errorf("too many values for input: %s", io.Type)
		}
		return ofs + count*int(io.Type.ElementType.Bits), nil

	case types.TSlice:
		count := int(io.Type.ArraySize)
		c, ofs, err := setArray(io.Type, result, inputs[0], ofs)
		if err != nil {
			return ofs, err
		}
		if c > count {
			return ofs, fmt.Errorf("too many values for input: %s", io.Type)
		}
		return ofs, nil

	default:
		return ofs, fmt.Errorf("unsupported input type: %s", io.Type)
	}
}

func setArray(t types.Info, result *big.Int, val interface{}, ofs int) (
	int, int, error) {

	switch t.ElementType.Type {
	case types.TInt, types.TUint:
		return setIntArray(t, result, val, ofs)

	default:
		return 0, ofs, fmt.Errorf("unsupported array element type: %s", t)
	}
}

func setIntArray(t types.Info, result *big.Int, val interface{}, ofs int) (
	int, int, error) {

	elSize := t.ElementType.Bits

	switch v := val.(type) {
	case []byte:
		if elSize < 8 {
			return 0, ofs, fmt.Errorf("invalid input '%T' for %s", v, t)
		}
		for i := 0; i < len(v); i++ {
			setInt(*t.ElementType, result, v[i], ofs)
			ofs += int(elSize)
		}
		return len(v), ofs, nil

	case nil:
		return 0, ofs, nil

	default:
		return 0, ofs, fmt.Errorf("invalid input '%T' for %s", v, t)
	}
}

func setInt(t types.Info, result *big.Int, val interface{}, ofs int) (
	int, error) {

	var ival uint64

	switch v := val.(type) {
	case int8:
		ival = uint64(v)
	case uint8:
		ival = uint64(v)
	case int16:
		ival = uint64(v)
	case uint16:
		ival = uint64(v)
	case int32:
		ival = uint64(v)
	case uint32:
		ival = uint64(v)
	case int64:
		ival = uint64(v)
	case uint64:
		ival = uint64(v)
	default:
		return ofs, fmt.Errorf("invalid input '%v' for %s", v, t)
	}

	for i := 0; i < 64; i++ {
		result.SetBit(result, ofs+i, uint((ival>>i)&0x1))
	}

	return ofs + int(t.Bits), nil
}

func setBool(result *big.Int, val interface{}, ofs int) (int, error) {
	v, ok := val.(bool)
	if !ok {
		return ofs, fmt.Errorf("invalid input '%v' for %s", val, types.TBool)
	}
	var flag uint
	if v {
		flag = 1
	}
	result.SetBit(result, ofs, flag)

	return ofs + 1, nil
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
				return nil, fmt.Errorf("invalid input '%s' for %s",
					inputs[0], io.Type)
			}

		case types.TBool:
			switch inputs[0] {
			case "0", "f", "false":
			case "1", "t", "true":
				result.SetInt64(1)
			default:
				return nil, fmt.Errorf("invalid bool constant: %s", inputs[0])
			}

		case types.TArray, types.TSlice:
			count := int(io.Type.ArraySize)
			elSize := int(io.Type.ElementType.Bits)
			if io.Type.Type == types.TArray && count == 0 {
				// Handle empty types.TArray arguments.
				break
			}

			val := new(big.Int)
			_, ok := val.SetString(inputs[0], 0)
			if !ok {
				return nil, fmt.Errorf("invalid input '%s' for %s",
					inputs[0], io.Type)
			}
			var bitLen int
			if strings.HasPrefix(inputs[0], "0x") {
				bitLen = (len(inputs[0]) - 2) * 4
			} else {
				bitLen = val.BitLen()
			}

			valElCount := bitLen / elSize
			if bitLen%elSize != 0 {
				valElCount++
			}
			if io.Type.Type == types.TSlice {
				// Set the count=valElCount for types.TSlice arguments.
				count = valElCount
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

		for i := 0; i < int(arg.Type.Bits); i++ {
			result.SetBit(result, offset+i, input.Bit(i))
		}

		offset += int(arg.Type.Bits)
	}
	return result, nil
}

// Sizes computes the bit sizes of the input arguments. This is used
// for parametrized main() when the program is instantiated based on
// input sizes.
func Sizes(inputs []interface{}) ([]int, error) {
	var result []int

	for _, input := range inputs {
		var size int
		switch v := input.(type) {
		case nil:
			size = 0
		case bool:
			size = 1
		case int8:
			size = bitLen(uint64(v))
		case uint8:
			size = bitLen(uint64(v))
		case int16:
			size = bitLen(uint64(v))
		case uint16:
			size = bitLen(uint64(v))
		case int32:
			size = bitLen(uint64(v))
		case uint32:
			size = bitLen(uint64(v))
		case int64:
			size = bitLen(uint64(v))
		case uint64:
			size = bitLen(v)
		case []byte:
			size = len(v) * 8
		default:
			return nil, fmt.Errorf("unsupport input %v[%T]", v, v)
		}
		result = append(result, size)
	}

	return result, nil
}

func bitLen(v uint64) int {
	for i := 63; i > 1; i-- {
		if v&(uint64(1)<<i) != 0 {
			return i + 1
		}
	}
	return 1
}

// InputSizes computes the bit sizes of the input arguments. This is
// used for parametrized main() when the program is instantiated based
// on input sizes.
func InputSizes(inputs []string) ([]int, error) {
	var result []int

	for _, input := range inputs {
		switch input {
		case "_":
			result = append(result, 0)

		case "0", "f", "false", "1", "t", "true":
			result = append(result, 1)

		default:
			if strings.HasPrefix(input, "0x") {
				result = append(result, (len(input)-2)*4)
			} else {
				m := reHexInput.FindStringSubmatch(input)
				if m != nil {
					count, err := strconv.Atoi(m[1])
					if err != nil {
						return nil, err
					}
					result = append(result, count*len(m[2])*4)
				} else {
					val := new(big.Int)
					_, ok := val.SetString(input, 0)
					if !ok {
						return nil, fmt.Errorf("invalid input: %s", input)
					}
					result = append(result, val.BitLen())
				}
			}
		}
	}

	return result, nil
}
