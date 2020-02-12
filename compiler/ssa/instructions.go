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
	Igt
	Ugt
	Fgt
	If
	Mov
	Jump
	Ret
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
	Igt:   "Igt",
	Ugt:   "Ugt",
	Fgt:   "Fgt",
	If:    "if",
	Mov:   "mov",
	Jump:  "jump",
	Ret:   "ret",
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
		return Instr{}, fmt.Errorf("Invalid type %s for addition", t)
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
		return Instr{}, fmt.Errorf("Invalid type %s for < comparison", t)
	}
	return Instr{
		Op:  op,
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

func NewRetInstr() Instr {
	return Instr{
		Op: Ret,
	}
}

// iadd i{0,0}/int32 j{0,0}/int32 x{0,0}/int32
// if0 i{0,0}/int32 block0
func (i Instr) String() string {
	result := i.Op.String()
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
	Name    string
	Scope   int
	Version int
	Type    types.Info
}

func (v Variable) String() string {
	return fmt.Sprintf("%s@%d,%d/%s",
		v.Name, v.Scope, v.Version, v.Type)
}
