//
// Copyright (c) 2020-2021 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/types"
)

// Operand specifies SSA assembly operand
type Operand uint8

// SSA assembly operands.
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
	Mov
	Amov
	Phi
	Ret
	Circ
	Builtin
	GC
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
	Mov:     "mov",
	Amov:    "amov",
	Phi:     "phi",
	Ret:     "ret",
	Circ:    "circ",
	Builtin: "builtin",
	GC:      "gc",
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

// Instr implements SSA assembly instruction.
type Instr struct {
	Op      Operand
	In      []Variable
	Out     *Variable
	Label   *Block
	Circ    *circuit.Circuit
	Builtin circuits.Builtin
	GC      string
	Ret     []Variable
}

// NewAddInstr creates a new addition instruction based on the type t.
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
		return Instr{}, fmt.Errorf("invalid type %s for addition", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewSubInstr creates a new subtraction instruction based on the type
// t.
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
		return Instr{}, fmt.Errorf("invalid type %s for subtraction", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewMultInstr creates a new multiplication instruction based on the
// type t.
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
		return Instr{}, fmt.Errorf("invalid type %s for multiplication", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewDivInstr creates a new division instruction based on the type t.
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
		return Instr{}, fmt.Errorf("invalid type %s for division", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewModInstr creates a new modulo instruction based on the type t.
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
		return Instr{}, fmt.Errorf("invalid type %s for modulo", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewLshiftInstr creates a new Lshift instruction.
func NewLshiftInstr(l, r, o Variable) Instr {
	return Instr{
		Op:  Lshift,
		In:  []Variable{l, r},
		Out: &o,
	}
}

// NewRshiftInstr creates a new Rshift instruction.
func NewRshiftInstr(l, r, o Variable) Instr {
	return Instr{
		Op:  Rshift,
		In:  []Variable{l, r},
		Out: &o,
	}
}

// NewSliceInstr creates a new Slice instruction.
func NewSliceInstr(v, from, to, o Variable) Instr {
	return Instr{
		Op:  Slice,
		In:  []Variable{v, from, to},
		Out: &o,
	}
}

// NewLtInstr creates a new less-than instruction based on the type t.
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
		return Instr{}, fmt.Errorf("invalid type %s for < comparison", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewLeInstr creates a new less-equal instruction based on the type
// t.
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
		return Instr{}, fmt.Errorf("invalid type %s for <= comparison", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewGtInstr creates a new greater-than instruction based on the type
// t.
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
		return Instr{}, fmt.Errorf("invalid type %s for > comparison", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewGeInstr creates a new greater-equal instruction based on the
// type t.
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
		return Instr{}, fmt.Errorf("invalid type %s for >= comparison", t)
	}
	return Instr{
		Op:  op,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewEqInstr creates a new Eq instruction.
func NewEqInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Eq,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewNeqInstr creates a new Neq instruction.
func NewNeqInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Neq,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewAndInstr creates a new And instruction.
func NewAndInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  And,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewOrInstr creates a new Or instruction.
func NewOrInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Or,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewBandInstr creates a new Band instruction.
func NewBandInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Band,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewBclrInstr creates a new Bclr instruction.
func NewBclrInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Bclr,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewBorInstr creates a new Bor instruction.
func NewBorInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Bor,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewBxorInstr creates a new Bxor instruction.
func NewBxorInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  Bxor,
		In:  []Variable{l, r},
		Out: &o,
	}, nil
}

// NewMovInstr creates a new Mov instruction.
func NewMovInstr(from, to Variable) Instr {
	return Instr{
		Op:  Mov,
		In:  []Variable{from},
		Out: &to,
	}
}

// NewAmovInstr creates a new Amov instruction.
func NewAmovInstr(v, arr, from, to, o Variable) Instr {
	return Instr{
		Op:  Amov,
		In:  []Variable{v, arr, from, to},
		Out: &o,
	}
}

// NewPhiInstr creates a new Phi instruction.
func NewPhiInstr(cond, l, r, v Variable) Instr {
	return Instr{
		Op:  Phi,
		In:  []Variable{cond, l, r},
		Out: &v,
	}
}

// NewRetInstr creates a new Ret instruction.
func NewRetInstr(ret []Variable) Instr {
	return Instr{
		Op: Ret,
		In: ret,
	}
}

// NewCircInstr creates a new Circ instruction.
func NewCircInstr(args []Variable, circ *circuit.Circuit,
	ret []Variable) Instr {
	return Instr{
		Op:   Circ,
		In:   args,
		Circ: circ,
		Ret:  ret,
	}
}

// NewBuiltinInstr creates a new Builtin instruction.
func NewBuiltinInstr(builtin circuits.Builtin, a, b, r Variable) Instr {
	return Instr{
		Op:      Builtin,
		In:      []Variable{a, b},
		Out:     &r,
		Builtin: builtin,
	}
}

// NewGCInstr creates a new GC instruction.
func NewGCInstr(v string) Instr {
	return Instr{
		Op: GC,
		GC: v,
	}
}

func (i Instr) String() string {
	return i.string(maxOperandLength, false)
}

// StringTyped returns a typed string representation of the instruction.
func (i Instr) StringTyped() string {
	return i.string(0, true)
}

func (i Instr) string(maxLen int, typesOnly bool) string {
	result := i.Op.String()

	if len(i.In) == 0 && i.Out == nil && i.Label == nil && len(i.GC) == 0 {
		return result
	}

	for len(result) < maxLen {
		result += " "
	}
	for _, i := range i.In {
		result += " "
		if typesOnly {
			result += i.Type.String()
		} else {
			result += i.String()
		}
	}
	if i.Out != nil {
		result += " "
		if typesOnly {
			result += i.Out.Type.String()
		} else {
			result += i.Out.String()
		}
	}
	if i.Label != nil {
		result += " "
		result += i.Label.ID
	}
	if i.Circ != nil {
		result += fmt.Sprintf(" {G=%d, W=%d}", i.Circ.NumGates, i.Circ.NumWires)
	}
	if len(i.GC) > 0 {
		result += " "
		result += i.GC
	}
	for _, r := range i.Ret {
		result += " "
		result += r.String()
	}
	return result
}

// PP pretty-prints instruction to the specified io.Writer.
func (i Instr) PP(out io.Writer) {
	fmt.Fprintf(out, "\t%s\n", i)
}

// Variable implements SSA variable binding.
type Variable struct {
	ID         VariableID
	Name       string
	Scope      int
	Version    int
	Type       types.Info
	TypeRef    bool
	Const      bool
	ConstValue interface{}
}

// VariableID defines unique variable IDs.
type VariableID uint32

func (v Variable) String() string {
	if v.Const {
		return v.Name
	}
	if v.TypeRef {
		return v.Type.String()
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

// Equal tests if this variable is equal to the argument binding value.
func (v *Variable) Equal(other BindingValue) bool {
	o, ok := other.(*Variable)
	if !ok {
		return false
	}
	return o.Name == v.Name && o.Scope == v.Scope && o.Version == v.Version
}

// Value returns the variables value.
func (v *Variable) Value(block *Block, gen *Generator) Variable {
	return *v
}

// Bit tests if the argument bit is set in the variable.
func (v *Variable) Bit(bit int) bool {
	arr, ok := v.ConstValue.([]interface{})
	if ok {
		length := len(arr)
		elBits := v.Type.Bits / length
		idx := bit / elBits
		ofs := bit % elBits
		if idx >= length {
			return false
		}
		return isSet(arr[idx], ofs)
	}

	return isSet(v.ConstValue, bit)
}

func isSet(v interface{}, bit int) bool {
	switch val := v.(type) {
	case bool:
		if bit == 0 {
			return val
		}
		return false

	case int32:
		return (val & (1 << bit)) != 0

	case uint32:
		return (val & (1 << bit)) != 0

	case int64:
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
		idx := bit / types.ByteBits
		mod := bit % types.ByteBits
		if idx >= len(bytes) {
			return false
		}
		return bytes[idx]&(1<<mod) != 0

	default:
		panic(fmt.Sprintf("isSet called for non const %v (%T)", v, val))
	}
}

// LValueFor checks if the value `o` can be assigned for lvalue of type `l`.
func LValueFor(l types.Info, o Variable) bool {
	if o.Const {
		return l.CanAssignConst(o.Type)
	}
	return l.Equal(o.Type)
}

// TypeCompatible tests if the argument variable is type compatible
// with this variable.
func (v Variable) TypeCompatible(o Variable) bool {
	if v.Const && o.Const {
		return v.Type.Type == o.Type.Type
	} else if v.Const {
		return o.Type.CanAssignConst(v.Type)
	} else if o.Const {
		return v.Type.CanAssignConst(o.Type)
	}
	return v.Type.Equal(o.Type)
}

// Constant creates a constant variable for the argument value. Type
// info is optional. If it is undefined, the type info will be
// resolved from the constant value.
func Constant(gen *Generator, value interface{}, ti types.Info) (
	Variable, error) {

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

	case int64:
		var minBits int
		// Count minimum bits needed to represent the value.
		for minBits = 1; minBits < 64; minBits++ {
			if (0xffffffffffffffff<<minBits)&uint64(val) == 0 {
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
		bits := len([]byte(val)) * types.ByteBits

		v.Type = types.Info{
			Type:    types.String,
			Bits:    bits,
			MinBits: bits,
		}

	case []interface{}:
		var bits int
		var length string
		var name string

		if len(val) > 0 {
			ev, err := Constant(gen, val[0], types.UndefinedInfo)
			if err != nil {
				return v, err
			}
			bits = ev.Type.Bits * len(val)
			name = ev.Type.String()
			length = fmt.Sprintf("%d", len(val))
		} else {
			name = "interface{}"
		}

		v.Name = fmt.Sprintf("$[%s]%s{%v}", length, name, arrayString(val))
		if ti.Undefined() {
			v.Type = types.Info{
				Type:    types.Array,
				Bits:    bits,
				MinBits: bits,
			}
		} else {
			v.Type = ti
			v.Type.Bits = len(val) * ti.ArrayElement.Bits
			v.Type.MinBits = ti.Bits
		}

	case types.Info:
		v.Name = fmt.Sprintf("$%s", val)
		v.Type = val
		v.TypeRef = true

	default:
		return v, fmt.Errorf("ssa.Constant: %v (%T) not implemented yet",
			val, val)
	}
	v.ID = gen.nextVariableID()

	return v, nil
}

func arrayString(arr []interface{}) string {
	var parts []string
	for _, part := range arr {
		parts = append(parts, fmt.Sprintf("%v", part))
	}
	return strings.Join(parts, ",")
}
