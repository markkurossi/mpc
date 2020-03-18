//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"io"
	"math/big"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/types"
)

type Operand uint8

const (
	Iadd Operand = iota
	Uadd
	Fadd
	Isub
	Usub
	Fsub
	Bor
	Bxor
	Band
	Bclr
	Bts
	Btc
	Imult
	Umult
	Fmult
	Idiv
	Udiv
	Fdiv
	Imod
	Umod
	Fmod
	Lshift
	Rshift
	Slice
	Ilt
	Ult
	Flt
	Ile
	Ule
	Fle
	Igt
	Ugt
	Fgt
	Ige
	Uge
	Fge
	Eq
	Neq
	And
	Or
	If
	Jump
	Mov
	Phi
	Ret
	Circ
	Builtin
)

var operands = map[Operand]string{
	Iadd:    "iadd",
	Uadd:    "uadd",
	Fadd:    "fadd",
	Isub:    "isub",
	Usub:    "usub",
	Fsub:    "fsub",
	Band:    "band",
	Bor:     "bor",
	Bxor:    "bxor",
	Bclr:    "bclr",
	Bts:     "bts",
	Btc:     "btc",
	Imult:   "imult",
	Umult:   "umult",
	Fmult:   "fmult",
	Idiv:    "idiv",
	Udiv:    "udiv",
	Fdiv:    "fdiv",
	Imod:    "imod",
	Umod:    "umod",
	Fmod:    "fmod",
	Lshift:  "lshift",
	Rshift:  "rshift",
	Slice:   "slice",
	Ilt:     "ilt",
	Ult:     "ult",
	Flt:     "flt",
	Ile:     "ile",
	Ule:     "ule",
	Fle:     "fle",
	Igt:     "igt",
	Ugt:     "ugt",
	Fgt:     "fgt",
	Ige:     "ige",
	Uge:     "uge",
	Fge:     "fge",
	Eq:      "eq",
	Neq:     "neq",
	And:     "and",
	Or:      "or",
	If:      "if",
	Jump:    "jump",
	Mov:     "mov",
	Phi:     "phi",
	Ret:     "ret",
	Circ:    "circ",
	Builtin: "builtin",
}

var maxOperandLength int

func init() {
	for _, v := range operands {
		if len(v) > maxOperandLength {
			maxOperandLength = len(v)
		}
	}
}

func (op Operand) String() string {
	name, ok := operands[op]
	if ok {
		return name
	}
	return fmt.Sprintf("{Operand %d}", op)
}

type Instr struct {
	Op      Operand
	In      []Variable
	Out     *Variable
	Label   *Block
	Circ    *circuit.Circuit
	Builtin circuits.Builtin
	Ret     []Variable
}

func NewAddInstr(t types.Info, l, r, o Variable) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.Int:
		op = Iadd
	case types.Uint:
		op = Uadd
	case types.Float:
		op = Fadd
	default:
		fmt.Printf("%v + %v (%v)\n", l, r, t)
		return Instr{}, fmt.Errorf("Invalid type %s for addition", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewSubInstr(t types.Info, l, r, o Variable) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.Int:
		op = Isub
	case types.Uint:
		op = Usub
	case types.Float:
		op = Fsub
	default:
		return Instr{}, fmt.Errorf("Invalid type %s for subtraction", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewMultInstr(t types.Info, l, r, o Variable) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.Int:
		op = Imult
	case types.Uint:
		op = Umult
	case types.Float:
		op = Fmult
	default:
		return Instr{}, fmt.Errorf("Invalid type %s for multiplication", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewDivInstr(t types.Info, l, r, o Variable) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.Int:
		op = Idiv
	case types.Uint:
		op = Udiv
	case types.Float:
		op = Fdiv
	default:
		return Instr{}, fmt.Errorf("Invalid type %s for multiplication", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewModInstr(t types.Info, l, r, o Variable) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.Int:
		op = Imod
	case types.Uint:
		op = Umod
	case types.Float:
		op = Fmod
	default:
		return Instr{}, fmt.Errorf("Invalid type %s for multiplication", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewLshiftInstr(l, r, o Variable) Instr {
	return Instr{
		Op:  Lshift,
		In:  []Variable{l, r},
		Out: &o,
	}
}

func NewRshiftInstr(l, r, o Variable) Instr {
	return Instr{
		Op:  Rshift,
		In:  []Variable{l, r},
		Out: &o,
	}
}

func NewSliceInstr(v, from, to, o Variable) Instr {
	return Instr{
		Op:  Slice,
		In:  []Variable{v, from, to},
		Out: &o,
	}
}

func NewLtInstr(t types.Info, l, r, o Variable) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.Int:
		op = Ilt
	case types.Uint:
		op = Ult
	case types.Float:
		op = Flt
	default:
		return Instr{}, fmt.Errorf("Invalid type %s for < comparison", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewLeInstr(t types.Info, l, r, o Variable) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.Int:
		op = Ile
	case types.Uint:
		op = Ule
	case types.Float:
		op = Fle
	default:
		return Instr{}, fmt.Errorf("Invalid type %s for <= comparison", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewGtInstr(t types.Info, l, r, o Variable) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.Int:
		op = Igt
	case types.Uint:
		op = Ugt
	case types.Float:
		op = Fgt
	default:
		return Instr{}, fmt.Errorf("Invalid type %s for > comparison", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewGeInstr(t types.Info, l, r, o Variable) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.Int:
		op = Ige
	case types.Uint:
		op = Uge
	case types.Float:
		op = Fge
	default:
		return Instr{}, fmt.Errorf("Invalid type %s for >= comparison", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewEqInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Eq,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewNeqInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Neq,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewAndInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  And,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewOrInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Or,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewBandInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Band,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewBclrInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Bclr,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewBorInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Bor,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewBxorInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Bxor,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

func NewIfInstr(c Variable, t *Block) Instr {
	return Instr{
		Op:    If,
		In:    []Variable{c},
		Label: t,
	}
}

func NewMovInstr(from, to Variable) Instr {
	return Instr{
		Op:  Mov,
		In:  []Variable{from},
		Out: &to,
	}
}

func NewJumpInstr(label *Block) Instr {
	return Instr{
		Op:    Jump,
		Label: label,
	}
}

func NewPhiInstr(cond, l, r, v Variable) Instr {
	return Instr{
		Op:  Phi,
		In:  []Variable{cond, l, r},
		Out: &v,
	}
}

func NewRetInstr(ret []Variable) Instr {
	return Instr{
		Op: Ret,
		In: ret,
	}
}

func NewCircInstr(args []Variable, circ *circuit.Circuit,
	ret []Variable) Instr {
	return Instr{
		Op:   Circ,
		In:   args,
		Circ: circ,
		Ret:  ret,
	}
}

func NewBuiltinInstr(builtin circuits.Builtin, a, b, r Variable) Instr {
	return Instr{
		Op:      Builtin,
		In:      []Variable{a, b},
		Out:     &r,
		Builtin: builtin,
	}
}

func (i Instr) String() string {
	result := i.Op.String()

	if len(i.In) == 0 && i.Out == nil && i.Label == nil {
		return result
	}

	for len(result) < maxOperandLength+1 {
		result += " "
	}
	for _, i := range i.In {
		result += " "
		result += i.String()
	}
	if i.Out != nil {
		result += " "
		result += i.Out.String()
	}
	if i.Label != nil {
		result += " "
		result += i.Label.ID
	}
	if i.Circ != nil {
		result += fmt.Sprintf(" {G=%d, W=%d}", i.Circ.NumGates, i.Circ.NumWires)
	}
	for _, r := range i.Ret {
		result += " "
		result += r.String()
	}
	return result
}

func (i Instr) PP(out io.Writer) {
	fmt.Fprintf(out, "\t%s\n", i)
}

type Variable struct {
	Name       string
	Scope      int
	Version    int
	Type       types.Info
	Const      bool
	ConstValue interface{}
}

func (v Variable) String() string {
	if v.Const {
		return v.Name
	}
	var version string
	if v.Version >= 0 {
		version = fmt.Sprintf("%d", v.Version)
	} else {
		version = "?"
	}
	return fmt.Sprintf("%s{%d,%s}%s",
		v.Name, v.Scope, version, v.Type.ShortString())
}

func (v *Variable) Equal(other BindingValue) bool {
	o, ok := other.(*Variable)
	if !ok {
		return false
	}
	return o.Name == v.Name && o.Scope == v.Scope && o.Version == v.Version
}

func (v *Variable) Value(block *Block, gen *Generator) Variable {
	return *v
}

func (v *Variable) Bit(bit int) bool {
	switch val := v.ConstValue.(type) {
	case bool:
		if bit == 0 {
			return val
		}
		return false

	case int32:
		return (val & (1 << bit)) != 0

	case uint64:
		return (val & (1 << bit)) != 0

	case *big.Int:
		if bit > val.BitLen() {
			return false
		}
		return val.Bit(bit) != 0

	case string:
		bytes := []byte(val)
		idx := bit / 8
		mod := bit % 8
		if idx >= len(bytes) {
			return false
		}
		return bytes[idx]&(1<<mod) != 0

	default:
		panic(fmt.Sprintf("Variable.Bit called for non const %v (%T)", v, val))
	}
}

// LValueFor checks if the value `o` can be assigned for lvalue of type `l`.
func LValueFor(l types.Info, o Variable) bool {
	if o.Const {
		return l.CanAssignConst(o.Type)
	} else {
		return l.Equal(o.Type)
	}
}

func (v Variable) TypeCompatible(o Variable) bool {
	if v.Const && o.Const {
		return v.Type.Type == o.Type.Type
	} else if v.Const {
		return o.Type.CanAssignConst(v.Type)
	} else if o.Const {
		return v.Type.CanAssignConst(o.Type)
	} else {
		return v.Type.Equal(o.Type)
	}
}

func Constant(value interface{}) (Variable, error) {
	v := Variable{
		Const:      true,
		ConstValue: value,
	}
	switch val := value.(type) {
	case int32:
		var minBits int
		// Count minimum bits needed to represent the value.
		for minBits = 1; minBits < 32; minBits++ {
			if (0xffffffff<<minBits)&uint64(val) == 0 {
				break
			}
		}

		v.Name = fmt.Sprintf("$%d", val)
		v.Type = types.Info{
			Type:    types.Uint,
			Bits:    32,
			MinBits: minBits,
		}

	case uint64:
		var minBits int
		// Count minimum bits needed to represent the value.
		for minBits = 1; minBits < 64; minBits++ {
			if (0xffffffffffffffff<<minBits)&val == 0 {
				break
			}
		}

		var bits int
		if minBits > 32 {
			bits = 64
		} else {
			bits = 32
		}

		v.Name = fmt.Sprintf("$%d", val)
		v.Type = types.Info{
			Type:    types.Uint,
			Bits:    bits,
			MinBits: minBits,
		}

	case *big.Int:
		v.Name = fmt.Sprintf("$%s", val.String())
		if val.Sign() == -1 {
			v.Type = types.Info{
				Type: types.Int,
			}
		} else {
			v.Type = types.Info{
				Type: types.Uint,
			}
		}
		minBits := val.BitLen()
		var bits int
		if minBits > 64 {
			bits = minBits
		} else if minBits > 32 {
			bits = 64
		} else {
			bits = 32
		}

		v.Type.Bits = bits
		v.Type.MinBits = minBits

	case bool:
		v.Name = fmt.Sprintf("$%v", val)
		v.Type = types.Info{
			Type:    types.Bool,
			Bits:    1,
			MinBits: 1,
		}

	case string:
		v.Name = fmt.Sprintf("$%q", val)
		bits := len([]byte(val)) * 8

		v.Type = types.Info{
			Type:    types.String,
			Bits:    bits,
			MinBits: bits,
		}

	default:
		return v, fmt.Errorf("Constant: %v (%T) not implemented yet", val, val)
	}
	return v, nil
}
