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
	Fprint(w io.Writer, indent int)
	Visit(enter, exit func(ast AST) error) error
	Compile(compiler *circuits.Compiler, out []*circuits.Wire) (
		[]*circuits.Wire, error)
}

type List []AST

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

type Type int

func (t Type) String() string {
	for k, v := range Types {
		if v == t {
			return k
		}
	}
	return fmt.Sprintf("{Type %d}", t)
}

const (
	TypeUndefined Type = iota
	TypeInt
	TypeFloat
)

var Types = map[string]Type{
	"<Untyped>": TypeUndefined,
	"int":       TypeInt,
	"float":     TypeFloat,
}

type TypeInfo struct {
	Type Type
	Bits int
}

func (t TypeInfo) String() string {
	if t.Bits == 0 {
		return t.Type.String()
	}
	return fmt.Sprintf("%s%d", t.Type, t.Bits)
}

type Variable struct {
	Name  string
	Type  TypeInfo
	Wires []*circuits.Wire
}

type Func struct {
	Loc    Point
	Name   string
	Args   []*Variable
	Return []*Variable
	Body   List
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
	Expr  AST
	True  List
	False List
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
	Exprs  []AST
	Return []*Variable
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
	Left  AST
	Op    BinaryType
	Right AST
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
	Name string
	Var  *Variable
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
