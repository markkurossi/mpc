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
)

var (
	_ AST = &List{}
	_ AST = &Func{}
	_ AST = &Return{}
	_ AST = &Binary{}
	_ AST = &Identifier{}
)

func indent(w io.Writer, indent int) {
	for i := 0; i < indent; i++ {
		fmt.Fprint(w, " ")
	}
}

type AST interface {
	Fprint(w io.Writer, indent int)
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

type Argument struct {
	Name string
	Type TypeInfo
}

type Func struct {
	Name   string
	Args   []Argument
	Return []TypeInfo
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
			fmt.Fprintf(w, "%s", ret)
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

type Return struct {
	Expr AST
}

func (ast *Return) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "return")
	if ast.Expr != nil {
		fmt.Fprintf(w, " ")
		ast.Expr.Fprint(w, 0)
	}
}

type BinaryType int

const (
	BinaryPlus BinaryType = iota
	BinaryMult
)

var binaryTypes = map[BinaryType]string{
	BinaryPlus: "+",
	BinaryMult: "*",
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

type Identifier struct {
	Name string
	// XXX Reference to variable
}

func (ast *Identifier) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "%s", ast.Name)
}
