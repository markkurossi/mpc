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
)

var (
	_ AST = &List{}
	_ AST = &Func{}
	_ AST = &If{}
	_ AST = &Return{}
	_ AST = &Binary{}
	_ AST = &VariableRef{}
)

type Point struct {
	Line int // 1-based
	Col  int // 0-based
}

func (s Point) String() string {
	return fmt.Sprintf("%d:%d", s.Line, s.Col)
}

func indent(w io.Writer, indent int) {
	for i := 0; i < indent; i++ {
		fmt.Fprint(w, " ")
	}
}

type AST interface {
	Location() Point
	Fprint(w io.Writer, indent int)
	Visit(enter, exit func(ast AST) error) error
	Compile(compiler *circuits.Compiler, out []*circuits.Wire) (
		[]*circuits.Wire, error)
	// SSA generates SSA code from the AST node. The code is appended
	// into the basic block `block'. The function returns the next
	// sequential basic block or nil if the AST node terminates the
	// control flow, i.e. any potentially following code will be dead.
	SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (*ssa.Block, error)
}

type List []AST

func (ast List) Location() Point {
	if len(ast) > 0 {
		return ast[0].Location()
	}
	return Point{}
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
	Loc    Point
	Name   string
	Args   []*Variable
	Return []*Variable
	Body   List
}

func (ast *Func) Location() Point {
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

type If struct {
	Loc   Point
	Expr  AST
	True  List
	False List
}

func (ast *If) Location() Point {
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
	Loc   Point
	Exprs []AST

	// XXX to be removed
	Return []*Variable
}

func (ast *Return) Location() Point {
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
	Loc   Point
	Left  AST
	Op    BinaryType
	Right AST
}

func (ast *Binary) Location() Point {
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
	Loc  Point
	Name string
	Var  *Variable
}

func (ast *VariableRef) Location() Point {
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
