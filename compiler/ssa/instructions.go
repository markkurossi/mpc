//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"io"

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
	Imult
	Umult
	Fmult
	Idiv
	Udiv
	Fdiv
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
	And
	Or
	If
	Jump
	Mov
	Phi
	Ret
	Call
)

var operands = map[Operand]string{
	Iadd:  "iadd",
	Uadd:  "uadd",
	Fadd:  "fadd",
	Isub:  "isub",
	Usub:  "usub",
	Fsub:  "fsub",
	Bor:   "bor",
	Bxor:  "bxor",
	Imult: "imult",
	Umult: "umult",
	Fmult: "fmult",
	Idiv:  "idiv",
	Udiv:  "udiv",
	Fdiv:  "fdiv",
	Ilt:   "ilt",
	Ult:   "ult",
	Flt:   "flt",
	Ile:   "ile",
	Ule:   "ule",
	Fle:   "fle",
	Igt:   "igt",
	Ugt:   "ugt",
	Fgt:   "fgt",
	Ige:   "ige",
	Uge:   "uge",
	Fge:   "fge",
	And:   "and",
	Or:    "or",
	If:    "if",
	Jump:  "jump",
	Mov:   "mov",
	Phi:   "phi",
	Ret:   "ret",
	Call:  "call",
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
	Op    Operand
	In    []Variable
	Out   *Variable
	Label *Block
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

func NewAndInstr(l, r, o Variable) (Instr, error) {
	return Instr{
		Op:  And,
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

func NewCallInstr(args []Variable, ret Variable, f *Block) Instr {
	return Instr{
		Op:    Call,
		In:    args,
		Out:   &ret,
		Label: f,
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

	case uint64:
		return (val & (1 << bit)) != 0

	default:
		panic(fmt.Sprintf("Variable.Bit called for a non variable %v", v))
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
