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

	"github.com/markkurossi/mpc/compiler/circuits"
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
	Visit(enter, exit func(ast AST) error) error
	Compile(compiler *circuits.Compiler, out []*circuits.Wire) (
		[]*circuits.Wire, error)
	// SSA generates SSA code from the AST node. The code is appended
	// into the basic block `block'. The function returns the next
	// sequential basic. The `ssa.Dead' is set to `true' if the code
	// terminates i.e. all following AST nodes are dead code.
	SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (*ssa.Block, error)
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

func (ast List) Visit(enter, exit func(ast AST) error) error {
	err := enter(ast)
	if err != nil {
		return err
	}
	for _, el := range ast {
		err = el.Visit(enter, exit)
		if err != nil {
			return err
		}
	}
	return exit(ast)
}

type Variable struct {
	Name  string
	Type  types.Info
	Wires []*circuits.Wire
}

type Func struct {
	Loc    utils.Point
	Name   string
	Args   []*Variable
	Return []*Variable
	Body   List
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

func (ast *Func) Visit(enter, exit func(ast AST) error) error {
	err := enter(ast)
	if err != nil {
		return err
	}
	for _, el := range ast.Body {
		err = el.Visit(enter, exit)
		if err != nil {
			return err
		}
	}
	return exit(ast)
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

func (ast *VariableDef) Visit(enter, exit func(ast AST) error) error {
	return fmt.Errorf("VariableDef.Visit not implemented")
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

func (ast *Assign) Visit(enter, exit func(ast AST) error) error {
	return fmt.Errorf("Assign.Visit not implemented")
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

func (ast *If) Visit(enter, exit func(ast AST) error) error {
	return fmt.Errorf("If.Visit not implemented yet")
}

type Return struct {
	Loc   utils.Point
	Exprs []AST

	// XXX to be removed
	Return []*Variable
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

func (ast *Return) Visit(enter, exit func(ast AST) error) error {
	err := enter(ast)
	if err != nil {
		return err
	}
	for _, expr := range ast.Exprs {
		err = expr.Visit(enter, exit)
		if err != nil {
			return err
		}
	}
	return exit(ast)
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

func (ast *Binary) Visit(enter, exit func(ast AST) error) error {
	err := enter(ast)
	if err != nil {
		return err
	}
	err = ast.Left.Visit(enter, exit)
	if err != nil {
		return err
	}
	err = ast.Right.Visit(enter, exit)
	if err != nil {
		return err
	}
	return exit(ast)
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

func (ast *VariableRef) Visit(enter, exit func(ast AST) error) error {
	err := enter(ast)
	if err != nil {
		return err
	}
	return exit(ast)
}

type Constant struct {
	Loc     utils.Point
	UintVal *uint64
}

func (ast *Constant) String() string {
	return fmt.Sprintf("$%d", *ast.UintVal)
}

func (ast *Constant) Variable() (ssa.Variable, error) {
	v := ssa.Variable{
		Const: true,
	}
	if ast.UintVal != nil {
		v.Name = fmt.Sprintf("$%d", *ast.UintVal)
		v.Type = types.Info{
			Type: types.Uint, // XXX min bits
		}
		v.ConstUint = ast.UintVal
	} else {
		return v, fmt.Errorf("constant %v not implemented yet", ast)
	}
	return v, nil
}

func (ast *Constant) Location() utils.Point {
	return ast.Loc
}

func (ast *Constant) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "%d", ast.UintVal)
}

func (ast *Constant) Visit(enter, exit func(ast AST) error) error {
	err := enter(ast)
	if err != nil {
		return err
	}
	return exit(ast)
}
