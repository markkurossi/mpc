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

func (a List) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "{\n")
	for _, el := range a {
		el.Fprint(w, ind+2)
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
	TypeInt Type = iota
	TypeFloat
)

var Types = map[string]Type{
	"int":   TypeInt,
	"float": TypeFloat,
}

type TypeInfo struct {
	Type Type
	Bits int
}

func (t TypeInfo) String() string {
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

func (a *Func) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "func %s(", a.Name)
	for idx, arg := range a.Args {
		if idx > 0 {
			fmt.Fprintf(w, ", ")
		}
		fmt.Fprintf(w, "%s %s", arg.Name, arg.Type)
	}

	fmt.Fprintf(w, ")")

	if len(a.Return) > 0 {
		fmt.Fprintf(w, " ")
		if len(a.Return) > 1 {
			fmt.Fprintf(w, "(")
		}
		for idx, ret := range a.Return {
			if idx > 0 {
				fmt.Fprintf(w, ", ")
			}
			fmt.Fprintf(w, "%s", ret)
		}
		if len(a.Return) > 1 {
			fmt.Fprintf(w, ")")
		}
	}

	if len(a.Body) > 0 {
		fmt.Fprintf(w, " {\n")
		fmt.Fprintf(w, "}")
	} else {
		fmt.Fprintf(w, " {}")
	}
}
