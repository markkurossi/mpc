//
// ast.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"
	"io"

	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
)

var (
	_ AST = &List{}
	_ AST = &Func{}
	_ AST = &VariableDef{}
	_ AST = &Assign{}
	_ AST = &If{}
	_ AST = &Call{}
	_ AST = &Return{}
	_ AST = &Binary{}
	_ AST = &VariableRef{}
	_ AST = &Constant{}
)

func indent(w io.Writer, indent int) {
	for i := 0; i < indent; i++ {
		fmt.Fprint(w, " ")
	}
}

type AST interface {
	Location() utils.Point
	Fprint(w io.Writer, indent int)
	// SSA generates SSA code from the AST node. The code is appended
	// into the basic block `block'. The function returns the next
	// sequential basic. The `ssa.Dead' is set to `true' if the code
	// terminates i.e. all following AST nodes are dead code.
	SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
		*ssa.Block, []ssa.Variable, error)
}

type List []AST

func (ast List) Location() utils.Point {
	if len(ast) > 0 {
		return ast[0].Location()
	}
	return utils.Point{}
}

func (ast List) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "{\n")
	for _, el := range ast {
		el.Fprint(w, ind+4)
		fmt.Fprintf(w, ",\n")
	}
	indent(w, ind)
	fmt.Fprintf(w, "}")
}

type Variable struct {
	Name string
	Type types.Info
}

type Func struct {
	Loc          utils.Point
	Name         string
	Args         []*Variable
	Return       []*Variable
	Body         List
	Bindings     map[string]ssa.Variable
	NumInstances int
}

func NewFunc(loc utils.Point, name string, args []*Variable, ret []*Variable,
	body List) *Func {
	return &Func{
		Loc:      loc,
		Name:     name,
		Args:     args,
		Return:   ret,
		Body:     body,
		Bindings: make(map[string]ssa.Variable),
	}
}

func (ast *Func) Location() utils.Point {
	return ast.Loc
}

func (ast *Func) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "func %s(", ast.Name)
	for idx, arg := range ast.Args {
		if idx > 0 {
			fmt.Fprintf(w, ", ")
		}
		fmt.Fprintf(w, "%s %s", arg.Name, arg.Type)
	}

	fmt.Fprintf(w, ")")

	if len(ast.Return) > 0 {
		fmt.Fprintf(w, " ")
		if len(ast.Return) > 1 {
			fmt.Fprintf(w, "(")
		}
		for idx, ret := range ast.Return {
			if idx > 0 {
				fmt.Fprintf(w, ", ")
			}
			fmt.Fprintf(w, "%s", ret.Type)
		}
		if len(ast.Return) > 1 {
			fmt.Fprintf(w, ")")
		}
	}

	if len(ast.Body) > 0 {
		fmt.Fprintf(w, " {\n")
		for _, b := range ast.Body {
			b.Fprint(w, ind+4)
			fmt.Fprintf(w, "\n")
		}
		indent(w, ind)
		fmt.Fprintf(w, "}")
	} else {
		fmt.Fprintf(w, " {}")
	}
}

type VariableDef struct {
	Loc   utils.Point
	Names []string
	Type  types.Info
}

func (ast *VariableDef) Location() utils.Point {
	return ast.Loc
}

func (ast *VariableDef) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "var ")
	for idx, name := range ast.Names {
		if idx > 0 {
			fmt.Fprintf(w, ", ")
			fmt.Fprintf(w, "%s", name)
		}
	}
	fmt.Fprintf(w, " %s", ast.Type)
}

type Assign struct {
	Loc  utils.Point
	Name string
	Expr AST
}

func (ast *Assign) Location() utils.Point {
	return ast.Loc
}

func (ast *Assign) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "%s = ", ast.Name)
	ast.Expr.Fprint(w, ind)
}

type If struct {
	Loc   utils.Point
	Expr  AST
	True  List
	False List
}

func (ast *If) Location() utils.Point {
	return ast.Loc
}

func (ast *If) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "if ")
	ast.Expr.Fprint(w, 0)
	ast.True.Fprint(w, ind)
	if ast.False != nil {
		fmt.Fprintf(w, "else ")
		ast.False.Fprint(w, ind)
	}
}

type Call struct {
	Loc   utils.Point
	Name  string
	Exprs []AST
}

func (ast *Call) Location() utils.Point {
	return ast.Loc
}

func (ast *Call) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "%s(", ast.Name)
	for idx, arg := range ast.Exprs {
		if idx > 0 {
			fmt.Fprint(w, ", ")
		}
		arg.Fprint(w, ind)
	}
	fmt.Fprint(w, ")")
}

type Return struct {
	Loc   utils.Point
	Exprs []AST
}

func (ast *Return) Location() utils.Point {
	return ast.Loc
}

func (ast *Return) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "return")
	if len(ast.Exprs) > 0 {
		fmt.Fprintf(w, " ")
		for idx, expr := range ast.Exprs {
			if idx > 0 {
				fmt.Fprintf(w, ", ")
			}
			expr.Fprint(w, 0)
		}
	}
}

type BinaryType int

const (
	BinaryPlus BinaryType = iota
	BinaryMinus
	BinaryMult
	BinaryDiv
	BinaryLt
	BinaryLe
	BinaryGt
	BinaryGe
	BinaryEq
	BinaryNeq
	BinaryAnd
	BinaryOr
)

var binaryTypes = map[BinaryType]string{
	BinaryPlus:  "+",
	BinaryMinus: "-",
	BinaryMult:  "*",
	BinaryDiv:   "/",
	BinaryLt:    "<",
	BinaryLe:    "<=",
	BinaryGt:    ">",
	BinaryGe:    ">=",
	BinaryEq:    "==",
	BinaryNeq:   "!=",
	BinaryAnd:   "&&",
	BinaryOr:    "||",
}

func (t BinaryType) String() string {
	name, ok := binaryTypes[t]
	if ok {
		return name
	}
	return fmt.Sprintf("{BinaryType %d}", t)
}

type Binary struct {
	Loc   utils.Point
	Left  AST
	Op    BinaryType
	Right AST
}

func (ast *Binary) Location() utils.Point {
	return ast.Loc
}

func (ast *Binary) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	ast.Left.Fprint(w, ind)
	fmt.Fprintf(w, " %s ", ast.Op)
	ast.Right.Fprint(w, ind)
}

func (ast *Binary) FprintDebug(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "(%s ", ast.Op)
	ast.Left.Fprint(w, ind)
	fmt.Fprintf(w, " ")
	ast.Right.Fprint(w, ind)
	fmt.Fprintf(w, ")")
}

type VariableRef struct {
	Loc  utils.Point
	Name string
	Var  *Variable
}

func (ast *VariableRef) Location() utils.Point {
	return ast.Loc
}

func (ast *VariableRef) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "%s", ast.Name)
}

type Constant struct {
	Loc   utils.Point
	Value interface{}
}

func (ast *Constant) String() string {
	switch val := ast.Value.(type) {
	case uint64:
		return fmt.Sprintf("$%d", val)
	case bool:
		return fmt.Sprintf("$%v", val)
	default:
		return fmt.Sprintf("{undefined constant %v}", ast.Value)
	}
}

func (ast *Constant) Variable() (ssa.Variable, error) {
	v := ssa.Variable{
		Const:      true,
		ConstValue: ast.Value,
	}
	switch val := ast.Value.(type) {
	case uint64:
		var bits int
		// Count minimum bits needed to represent the value.
		for bits = 1; bits < 64; bits++ {
			if (0xffffffffffffffff<<bits)&val == 0 {
				break
			}
		}
		v.Name = fmt.Sprintf("$%d", val)
		v.Type = types.Info{
			Type: types.Uint,
			Bits: bits,
		}

	case bool:
		v.Name = fmt.Sprintf("$%v", val)
		v.Type = types.Info{
			Type: types.Bool,
			Bits: 1,
		}

	default:
		return v, fmt.Errorf("ast.Constant.Variable(): %v not implemented yet",
			ast)
	}
	return v, nil
}

func (ast *Constant) Location() utils.Point {
	return ast.Loc
}

func (ast *Constant) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "%v", ast.Value)
}
