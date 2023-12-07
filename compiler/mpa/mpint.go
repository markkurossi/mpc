//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.
//

package mpa

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/types"
)

// Int implements multi-precision integer arithmetics.
type Int struct {
	bits   int32
	small  bool
	i64    int64
	values *big.Int
}

// NewInt creates a new Int with init value x.
func NewInt(x int64) *Int {
	return &Int{
		bits:  64,
		small: true,
		i64:   x,
	}
}

// Debug prints debug information about z.
func (z *Int) Debug() {
	if z.small {
		fmt.Printf("mpa.Big: val=%v, bits=%v, i64=%v\n", z, z.bits, z.i64)
	} else {
		fmt.Printf("mpa.Big: val=%v, bits=%v, bitLen=%v, values=%x\n",
			z, z.bits, z.BitLen(), z.values.Bits())
	}
}

// TypeSize returns the type size in bits.
func (z *Int) TypeSize() int {
	return int(z.bits)
}

// SetTypeSize sets the type size in bits.
func (z *Int) SetTypeSize(size int32) {
	z.bits = size
}

// Bit returns the value of the i'th bit of z.
func (z *Int) Bit(i int) uint {
	if z.small {
		return uint((z.i64 >> uint(i)) & 0x1)
	}
	return z.values.Bit(i)
}

// BitLen returns the length of the absolute value of z.
func (z *Int) BitLen() int {
	if z.small {
		var bitLen int
		for bitLen = 1; bitLen < 64; bitLen++ {
			if (0xffffffffffffffff<<bitLen)&uint64(z.i64) == 0 {
				break
			}
		}
		return bitLen
	}
	return z.values.BitLen()
}

// Cmp compares z for x and returns -1, 0, 1 if z is smaller, equal,
// or greater than x.
func (z *Int) Cmp(x *Int) int {
	if z.small && x.small {
		if z.i64 < x.i64 {
			return -1
		} else if z.i64 > x.i64 {
			return 1
		} else {
			return 0
		}
	}
	zv := z.signed(z.bits - 1)
	xv := x.signed(x.bits - 1)
	return zv.Cmp(xv)
}

func (z *Int) big() *big.Int {
	if z.values == nil {
		z.values = big.NewInt(z.i64)
	}
	return z.values
}

func (z *Int) signed(signBit int32) *big.Int {
	bigInt := z.big()
	var sign int
	if signBit >= 0 {
		if bigInt.Bit(int(signBit)) == 1 {
			sign = -1
		} else {
			sign = 1
		}
	}
	result := big.NewInt(0).Set(bigInt)
	rsign := result.Sign()
	if sign != 0 && sign != rsign {
		result.Neg(result)
	}
	return result
}

// Int64 returns the int64 representation of x. If x cannot be
// represented as int64, the result is undefined.
func (z *Int) Int64() int64 {
	if z.small {
		return z.i64
	}
	return z.values.Int64()
}

func (z *Int) String() string {
	if z.small {
		return strconv.FormatInt(z.i64, 10)
	}
	return z.values.String()
}

// Add sets z to x+y and returns z.
func (z *Int) Add(x, y *Int) *Int {
	if x.small && y.small {
		z.setShort(x.i64 + y.i64)
		return z
	}
	return z.bin(circuits.NewAdder, x, y)
}

// And sets z to x&y and returns z.
func (z *Int) And(x, y *Int) *Int {
	if x.small && y.small {
		z.setShort(x.i64 & y.i64)
	} else {
		z.values = big.NewInt(0).And(x.big(), y.big())
		z.small = false
	}
	z.bits = max(x.bits, y.bits)
	return z
}

// Div sets z to x/y and returns z.
func (z *Int) Div(x, y *Int) *Int {
	if x.small && y.small {
		if y.i64 == 0 {
			z.setShort(-1)
		} else {
			z.setShort(x.i64 / y.i64)
		}
		return z
	}
	calloc := circuits.NewAllocator()
	inputs := circuit.IO{
		newIOArg("x", types.TInt, x.bits),
		newIOArg("y", types.TInt, y.bits),
	}
	outputs := circuit.IO{
		newIOArg("q", types.TInt, max(x.bits, y.bits)),
		newIOArg("r", types.TInt, max(x.bits, y.bits)),
	}
	i0w := calloc.Wires(inputs[0].Type.Bits)
	i1w := calloc.Wires(inputs[1].Type.Bits)

	var inputWires []*circuits.Wire
	inputWires = append(inputWires, i0w...)
	inputWires = append(inputWires, i1w...)

	o0w := calloc.Wires(outputs[0].Type.Bits)
	o1w := calloc.Wires(outputs[1].Type.Bits)

	var outputWires []*circuits.Wire
	outputWires = append(outputWires, o0w...)
	outputWires = append(outputWires, o1w...)

	for idx := range outputWires {
		outputWires[idx].SetOutput(true)
	}

	cc, err := circuits.NewCompiler(utils.NewParams(), calloc, inputs, outputs,
		inputWires, outputWires)
	if err != nil {
		panic(err)
	}

	err = circuits.NewDivider(cc, i0w, i1w, o0w, o1w)
	if err != nil {
		panic(err)
	}

	circ := cc.Compile()

	obits, err := circ.Compute([]*big.Int{x.big(), y.big()})
	if err != nil {
		panic(err)
	}

	z.bits = int32(outputs[0].Type.Bits)
	z.values = obits[0]
	z.small = false

	return z
}

// Lsh sets z to x<<n and returns z.
func (z *Int) Lsh(x *Int, n uint) *Int {
	if x.small {
		z.setShort(x.i64 << n)
		return z
	}
	if z != x {
		z.bits = x.bits
		z.values = big.NewInt(0).Set(x.values)
		z.small = false
	}
	z.values.Lsh(z.values, n)
	for i := z.values.BitLen() - 1; i >= int(z.bits); i-- {
		z.values.SetBit(z.values, i, 0)
	}
	return z
}

// Mod sets z to x%y and returns z.
func (z *Int) Mod(x, y *Int) *Int {
	if x.small && y.small {
		if y.i64 == 0 {
			z.setShort(x.i64)
		} else {
			z.setShort(x.i64 % y.i64)
		}
		return z
	}
	calloc := circuits.NewAllocator()
	inputs := circuit.IO{
		newIOArg("x", types.TInt, x.bits),
		newIOArg("y", types.TInt, y.bits),
	}
	outputs := circuit.IO{
		newIOArg("q", types.TInt, max(x.bits, y.bits)),
		newIOArg("r", types.TInt, max(x.bits, y.bits)),
	}
	i0w := calloc.Wires(inputs[0].Type.Bits)
	i1w := calloc.Wires(inputs[1].Type.Bits)

	var inputWires []*circuits.Wire
	inputWires = append(inputWires, i0w...)
	inputWires = append(inputWires, i1w...)

	o0w := calloc.Wires(outputs[0].Type.Bits)
	o1w := calloc.Wires(outputs[1].Type.Bits)

	var outputWires []*circuits.Wire
	outputWires = append(outputWires, o0w...)
	outputWires = append(outputWires, o1w...)

	for idx := range outputWires {
		outputWires[idx].SetOutput(true)
	}

	cc, err := circuits.NewCompiler(utils.NewParams(), calloc, inputs, outputs,
		inputWires, outputWires)
	if err != nil {
		panic(err)
	}

	err = circuits.NewDivider(cc, i0w, i1w, o0w, o1w)
	if err != nil {
		panic(err)
	}

	circ := cc.Compile()

	obits, err := circ.Compute([]*big.Int{x.big(), y.big()})
	if err != nil {
		panic(err)
	}

	z.bits = int32(outputs[1].Type.Bits)
	z.values = obits[1]
	z.small = false

	return z
}

// Mul sets z to x*y and returns z.
func (z *Int) Mul(x, y *Int) *Int {
	if x.small && y.small {
		z.setShort(x.i64 * y.i64)
		return z
	}
	return z.bin(func(cc *circuits.Compiler, x, y, z []*circuits.Wire) error {
		return circuits.NewMultiplier(cc, 0, x, y, z)
	}, x, y)
}

// Or sets z to x|y and returns z.
func (z *Int) Or(x, y *Int) *Int {
	if x.small && y.small {
		z.setShort(x.i64 | y.i64)
	} else {
		z.values = big.NewInt(0).Or(x.big(), y.big())
		z.small = false
	}
	z.bits = max(x.bits, y.bits)
	return z
}

// Rsh sets z to x>>n and returns z.
func (z *Int) Rsh(x *Int, n uint) *Int {
	if x.small {
		z.setShort(x.i64 >> n)
		return z
	}
	if z != x {
		z.bits = x.bits
		z.values = big.NewInt(0).Set(x.values)
		z.small = false
	}
	z.values.Rsh(z.values, n)
	return z
}

// SetBig sets z to x and returns z.
func (z *Int) SetBig(x *big.Int) *Int {
	if x.IsInt64() {
		z.setShort(x.Int64())
		return z
	}
	z.bits = int32(x.BitLen())
	if z.bits > 0 && x.Sign() == 1 && x.Bit(int(z.bits-1)) == 1 {
		z.bits++
	}
	z.values = new(big.Int).Set(x)
	z.small = false
	return z
}

func (z *Int) setShort(x int64) {
	z.i64 = x
	z.values = nil
	z.small = true
}

// SetString sets z to s according to its ascii value. The argument
// base specifies how the argument string base is interpreted.
func (z *Int) SetString(s string, base int) (*Int, bool) {
	i, ok := new(big.Int).SetString(s, base)
	if !ok {
		return nil, false
	}
	z.SetBig(i)
	return z, true
}

// Sign returns -1, 0, 1 if z is negative, zero, or positive.
func (z *Int) Sign() int {
	if z.small {
		if z.i64 < 0 {
			return -1
		} else if z.i64 > 0 {
			return 1
		} else {
			return 0
		}
	}
	return z.values.Sign()
}

// Sub sets z to x-y and returns z.
func (z *Int) Sub(x, y *Int) *Int {
	if x.small && y.small {
		z.setShort(x.i64 - y.i64)
		return z
	}
	return z.bin(circuits.NewSubtractor, x, y)
}

// Xor sets z to x^y and returns z.
func (z *Int) Xor(x, y *Int) *Int {
	if x.small && y.small {
		z.setShort(x.i64 ^ y.i64)
	} else {
		z.values = big.NewInt(0).Xor(x.big(), y.big())
		z.small = false
	}
	z.bits = max(x.bits, y.bits)
	return z
}

type binaryOp func(cc *circuits.Compiler, x, y, z []*circuits.Wire) error

func (z *Int) bin(op binaryOp, x, y *Int) *Int {
	calloc := circuits.NewAllocator()
	inputs := circuit.IO{
		newIOArg("x", types.TInt, x.bits),
		newIOArg("y", types.TInt, y.bits),
	}
	outputs := circuit.IO{
		newIOArg("z", types.TInt, max(x.bits, y.bits)),
	}
	i0w := calloc.Wires(inputs[0].Type.Bits)
	i1w := calloc.Wires(inputs[1].Type.Bits)
	var inputWires []*circuits.Wire
	inputWires = append(inputWires, i0w...)
	inputWires = append(inputWires, i1w...)

	outputWires := calloc.Wires(outputs[0].Type.Bits)
	for idx := range outputWires {
		outputWires[idx].SetOutput(true)
	}

	cc, err := circuits.NewCompiler(utils.NewParams(), calloc, inputs, outputs,
		inputWires, outputWires)
	if err != nil {
		panic(err)
	}

	err = op(cc, i0w, i1w, outputWires)
	if err != nil {
		panic(err)
	}

	circ := cc.Compile()

	obits, err := circ.Compute([]*big.Int{x.big(), y.big()})
	if err != nil {
		panic(err)
	}

	z.bits = int32(outputs[0].Type.Bits)
	z.values = obits[0]
	z.small = false

	return z
}

func newIOArg(name string, t types.Type, size int32) circuit.IOArg {
	return circuit.IOArg{
		Name: name,
		Type: types.Info{
			Type:       t,
			IsConcrete: true,
			Bits:       types.Size(size),
		},
	}
}
