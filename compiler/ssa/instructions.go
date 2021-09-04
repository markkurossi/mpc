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
	Srshift
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
	Smov
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
	Srshift: "srshift",
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
	Smov:    "smov",
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
	In      []Value
	Out     *Value
	Label   *Block
	Circ    *circuit.Circuit
	Builtin circuits.Builtin
	GC      string
	Ret     []Value
}

// Check verifies that the instruction values are properly set. If any
// unspecified values are found, the Check function panics.
func (i Instr) Check() {
	for idx, in := range i.In {
		if !in.Check() {
			panic(fmt.Sprintf("invalid input %d: %s (%#v)", idx, in, in))
		}
	}
}

// NewAddInstr creates a new addition instruction based on the type t.
func NewAddInstr(t types.Info, l, r, o Value) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.TInt:
		op = Iadd
	case types.TUint:
		op = Uadd
	case types.TFloat:
		op = Fadd
	default:
		fmt.Printf("%v + %v (%v)\n", l, r, t)
		return Instr{}, fmt.Errorf("invalid type %s for addition", t)
	}
	return Instr{
		Op:  op,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewSubInstr creates a new subtraction instruction based on the type
// t.
func NewSubInstr(t types.Info, l, r, o Value) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.TInt:
		op = Isub
	case types.TUint:
		op = Usub
	case types.TFloat:
		op = Fsub
	default:
		return Instr{}, fmt.Errorf("invalid type %s for subtraction", t)
	}
	return Instr{
		Op:  op,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewMultInstr creates a new multiplication instruction based on the
// type t.
func NewMultInstr(t types.Info, l, r, o Value) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.TInt:
		op = Imult
	case types.TUint:
		op = Umult
	case types.TFloat:
		op = Fmult
	default:
		return Instr{}, fmt.Errorf("invalid type %s for multiplication", t)
	}
	return Instr{
		Op:  op,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewDivInstr creates a new division instruction based on the type t.
func NewDivInstr(t types.Info, l, r, o Value) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.TInt:
		op = Idiv
	case types.TUint:
		op = Udiv
	case types.TFloat:
		op = Fdiv
	default:
		return Instr{}, fmt.Errorf("invalid type %s for division", t)
	}
	return Instr{
		Op:  op,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewModInstr creates a new modulo instruction based on the type t.
func NewModInstr(t types.Info, l, r, o Value) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.TInt:
		op = Imod
	case types.TUint:
		op = Umod
	case types.TFloat:
		op = Fmod
	default:
		return Instr{}, fmt.Errorf("invalid type %s for modulo", t)
	}
	return Instr{
		Op:  op,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewLshiftInstr creates a new Lshift instruction.
func NewLshiftInstr(l, r, o Value) Instr {
	return Instr{
		Op:  Lshift,
		In:  []Value{l, r},
		Out: &o,
	}
}

// NewRshiftInstr creates a new Rshift instruction.
func NewRshiftInstr(l, r, o Value) Instr {
	return Instr{
		Op:  Rshift,
		In:  []Value{l, r},
		Out: &o,
	}
}

// NewSrshiftInstr creates a new Srshift instruction.
func NewSrshiftInstr(l, r, o Value) Instr {
	return Instr{
		Op:  Srshift,
		In:  []Value{l, r},
		Out: &o,
	}
}

// NewSliceInstr creates a new Slice instruction.
func NewSliceInstr(v, from, to, o Value) Instr {
	return Instr{
		Op:  Slice,
		In:  []Value{v, from, to},
		Out: &o,
	}
}

// NewLtInstr creates a new less-than instruction based on the type t.
func NewLtInstr(t types.Info, l, r, o Value) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.TInt:
		op = Ilt
	case types.TUint:
		op = Ult
	case types.TFloat:
		op = Flt
	default:
		return Instr{}, fmt.Errorf("invalid type %s for < comparison", t)
	}
	return Instr{
		Op:  op,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewLeInstr creates a new less-equal instruction based on the type
// t.
func NewLeInstr(t types.Info, l, r, o Value) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.TInt:
		op = Ile
	case types.TUint:
		op = Ule
	case types.TFloat:
		op = Fle
	default:
		return Instr{}, fmt.Errorf("invalid type %s for <= comparison", t)
	}
	return Instr{
		Op:  op,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewGtInstr creates a new greater-than instruction based on the type
// t.
func NewGtInstr(t types.Info, l, r, o Value) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.TInt:
		op = Igt
	case types.TUint:
		op = Ugt
	case types.TFloat:
		op = Fgt
	default:
		return Instr{}, fmt.Errorf("invalid type %s for > comparison", t)
	}
	return Instr{
		Op:  op,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewGeInstr creates a new greater-equal instruction based on the
// type t.
func NewGeInstr(t types.Info, l, r, o Value) (Instr, error) {
	var op Operand
	switch t.Type {
	case types.TInt:
		op = Ige
	case types.TUint:
		op = Uge
	case types.TFloat:
		op = Fge
	default:
		return Instr{}, fmt.Errorf("invalid type %s for >= comparison", t)
	}
	return Instr{
		Op:  op,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewEqInstr creates a new Eq instruction.
func NewEqInstr(l, r, o Value) (Instr, error) {
	return Instr{
		Op:  Eq,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewNeqInstr creates a new Neq instruction.
func NewNeqInstr(l, r, o Value) (Instr, error) {
	return Instr{
		Op:  Neq,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewAndInstr creates a new And instruction.
func NewAndInstr(l, r, o Value) (Instr, error) {
	return Instr{
		Op:  And,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewOrInstr creates a new Or instruction.
func NewOrInstr(l, r, o Value) (Instr, error) {
	return Instr{
		Op:  Or,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewBandInstr creates a new Band instruction.
func NewBandInstr(l, r, o Value) (Instr, error) {
	return Instr{
		Op:  Band,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewBclrInstr creates a new Bclr instruction.
func NewBclrInstr(l, r, o Value) (Instr, error) {
	return Instr{
		Op:  Bclr,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewBtsInstr creates a new Bts instruction.
func NewBtsInstr(l, r, o Value) Instr {
	return Instr{
		Op:  Bts,
		In:  []Value{l, r},
		Out: &o,
	}
}

// NewBorInstr creates a new Bor instruction.
func NewBorInstr(l, r, o Value) (Instr, error) {
	return Instr{
		Op:  Bor,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewBxorInstr creates a new Bxor instruction.
func NewBxorInstr(l, r, o Value) (Instr, error) {
	return Instr{
		Op:  Bxor,
		In:  []Value{l, r},
		Out: &o,
	}, nil
}

// NewMovInstr creates a new Mov instruction.
func NewMovInstr(from, to Value) Instr {
	return Instr{
		Op:  Mov,
		In:  []Value{from},
		Out: &to,
	}
}

// NewSmovInstr creates a new Mov instruction.
func NewSmovInstr(from, to Value) Instr {
	return Instr{
		Op:  Smov,
		In:  []Value{from},
		Out: &to,
	}
}

// NewAmovInstr creates a new Amov instruction.
func NewAmovInstr(v, arr, from, to, o Value) Instr {
	return Instr{
		Op:  Amov,
		In:  []Value{v, arr, from, to},
		Out: &o,
	}
}

// NewPhiInstr creates a new Phi instruction.
func NewPhiInstr(cond, t, f, v Value) Instr {
	return Instr{
		Op:  Phi,
		In:  []Value{cond, t, f},
		Out: &v,
	}
}

// NewRetInstr creates a new Ret instruction.
func NewRetInstr(ret []Value) Instr {
	return Instr{
		Op: Ret,
		In: ret,
	}
}

// NewCircInstr creates a new Circ instruction.
func NewCircInstr(args []Value, circ *circuit.Circuit,
	ret []Value) Instr {
	return Instr{
		Op:   Circ,
		In:   args,
		Circ: circ,
		Ret:  ret,
	}
}

// NewBuiltinInstr creates a new Builtin instruction.
func NewBuiltinInstr(builtin circuits.Builtin, a, b, r Value) Instr {
	return Instr{
		Op:      Builtin,
		In:      []Value{a, b},
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

// Scope defines variable scope (max 256 levels of nested blocks).
type Scope int16

// Value implements SSA value binding.
type Value struct {
	Name       string
	ID         ValueID
	TypeRef    bool
	Const      bool
	Scope      Scope
	Version    int32
	Type       types.Info
	PtrInfo    *PtrInfo
	ConstValue interface{}
}

// PtrInfo defines context information for pointer values.
type PtrInfo struct {
	Name          string
	Bindings      *Bindings
	Scope         Scope
	Offset        types.Size
	ContainerType types.Info
}

func (ptr PtrInfo) String() string {
	return fmt.Sprintf("*%s@%d", ptr.Name, ptr.Scope)
}

// Undefined defines an undefined value.
var Undefined Value

// ValueID defines unique value IDs.
type ValueID uint32

// Check tests that the value type is properly set.
func (v Value) Check() bool {
	return v.Type.Type != types.TUndefined && v.Type.Bits != 0
}

// ElementType returns the pointer element type of the value. For
// non-pointer values, this returns the value type itself.
func (v Value) ElementType() types.Info {
	if v.Type.Type == types.TPtr {
		return *v.Type.ElementType
	}
	return v.Type
}

// ContainerType returs the pointer container type of the value. For
// non-pointer values, this returns the value type itself.
func (v Value) ContainerType() types.Info {
	if v.Type.Type == types.TPtr {
		return v.PtrInfo.ContainerType
	}
	return v.Type
}

func (v Value) String() string {
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

	// XXX Value should have type, now we have flags and Type.Type
	if v.Type.Type == types.TPtr {
		return fmt.Sprintf("%s{%d,%s}%s{%s{%d}%s[%d-%d]}",
			v.Name, v.Scope, version, v.Type.ShortString(),
			v.PtrInfo.Name, v.PtrInfo.Scope, v.PtrInfo.ContainerType,
			v.PtrInfo.Offset, v.PtrInfo.Offset+v.Type.Bits)
	}
	return fmt.Sprintf("%s{%d,%s}%s",
		v.Name, v.Scope, version, v.Type.ShortString())
}

// ConstInt returns the value as const integer.
func (v *Value) ConstInt() (types.Size, error) {
	if !v.Const {
		return 0, fmt.Errorf("value is not constant")
	}
	switch val := v.ConstValue.(type) {
	case int:
		return types.Size(val), nil
	case int32:
		return types.Size(val), nil
	case int64:
		return types.Size(val), nil
	case uint64:
		return types.Size(val), nil

	default:
		return 0, fmt.Errorf("cannot use %v as integer", val)
	}
}

// Equal implements BindingValue.Equal.
func (v *Value) Equal(other BindingValue) bool {
	o, ok := other.(*Value)
	if !ok {
		return false
	}
	return o.Name == v.Name && o.Scope == v.Scope && o.Version == v.Version
}

// Value implements BindingValue.Value.
func (v *Value) Value(block *Block, gen *Generator) Value {
	return *v
}

// Bit tests if the argument bit is set in the value.
func (v *Value) Bit(bit types.Size) bool {
	arr, ok := v.ConstValue.([]interface{})
	if ok {
		length := types.Size(len(arr))
		elBits := v.Type.Bits / length
		idx := bit / elBits
		ofs := bit % elBits
		if idx >= length {
			return false
		}
		return isSet(arr[idx], v.Type, ofs)
	}

	return isSet(v.ConstValue, v.Type, bit)
}

func isSet(v interface{}, vt types.Info, bit types.Size) bool {
	switch val := v.(type) {
	case bool:
		if bit == 0 {
			return val
		}
		return false

	case int8:
		return (val & (1 << bit)) != 0

	case uint8:
		return (val & (1 << bit)) != 0

	case int32:
		return (val & (1 << bit)) != 0

	case uint32:
		return (val & (1 << bit)) != 0

	case int64:
		return (val & (1 << bit)) != 0

	case uint64:
		return (val & (1 << bit)) != 0

	case *big.Int:
		if bit > types.Size(val.BitLen()) {
			return false
		}
		return val.Bit(int(bit)) != 0

	case string:
		bytes := []byte(val)
		idx := bit / types.ByteBits
		mod := bit % types.ByteBits
		if idx >= types.Size(len(bytes)) {
			return false
		}
		return bytes[idx]&(1<<mod) != 0

	case Value:
		switch val.Type.Type {
		case types.TBool, types.TInt, types.TUint, types.TFloat, types.TString:
			return isSet(val.ConstValue, val.Type, bit)

		case types.TArray:
			elType := val.Type.ElementType
			idx := bit / elType.Bits
			mod := bit % elType.Bits
			if idx >= val.Type.ArraySize {
				return false
			}
			arr := val.ConstValue.([]interface{})
			return isSet(arr[idx], *elType, mod)

		case types.TStruct:
			fieldValues := val.ConstValue.([]interface{})
			for idx, f := range val.Type.Struct {
				if bit < f.Type.Bits {
					return isSet(fieldValues[idx], f.Type, bit)
				}
				bit -= f.Type.Bits
			}
			fallthrough

		default:
			panic(fmt.Sprintf("ssa.isSet: invalid Value %v (%v)",
				val, val.Type))
		}

	case types.Info:
		return false

	case []interface{}:
		switch vt.Type {
		case types.TStruct:
			for idx, f := range vt.Struct {
				if bit < f.Type.Bits {
					return isSet(val[idx], f.Type, bit)
				}
				bit -= f.Type.Bits
			}
			panic(fmt.Sprintf("ssa.isSet: bit overflow for %v", vt))

		case types.TArray:
			elType := vt.ElementType
			idx := bit / elType.Bits
			mod := bit % elType.Bits
			if idx >= vt.ArraySize {
				return false
			}
			return isSet(val[idx], *elType, mod)

		default:
			panic(fmt.Sprintf("ssa.isSet: type %v not supported", vt))
		}

	default:
		panic(fmt.Sprintf("ssa.isSet: non const %v (%T)", v, val))
	}
}

// LValueFor checks if the value `o` can be assigned for lvalue of type `l`.
func LValueFor(l types.Info, o Value) bool {
	if o.Const {
		return l.CanAssignConst(o.Type)
	}
	return l.Equal(o.Type)
}

// TypeCompatible tests if the argument value is type compatible with
// this value.
func (v Value) TypeCompatible(o Value) *types.Info {
	if v.Const && o.Const {
		if v.Type.Type == o.Type.Type {
			return &v.Type
		}
	} else if v.Const {
		if o.Type.CanAssignConst(v.Type) {
			return &o.Type
		}
	} else if o.Const {
		if v.Type.CanAssignConst(o.Type) {
			return &v.Type
		}
	}
	if v.Type.Equal(o.Type) {
		return &v.Type
	}
	return nil
}

// IntegerLike tests if the value is an integer.
func (v Value) IntegerLike() bool {
	switch v.Type.Type {
	case types.TInt, types.TUint:
		return true
	default:
		return false
	}
}
