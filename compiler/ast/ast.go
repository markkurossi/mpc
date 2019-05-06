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

type AST interface {
	Fprint(w io.Writer, indent int)
}

type List struct {
	Elements []AST
}

func (a *List) Fprint(w io.Writer, indent int) {
	fmt.Fprintf(w, "[\n")
	for _, el := range a.Elements {
		el.Fprint(w, indent+2)
		fmt.Fprintf(w, ",\n")
	}
	fmt.Fprintf(w, "]")
}

type Func struct {
	Name string
	Body []AST
}

func (a *Func) Fprint(w io.Writer, indent int) {
	fmt.Fprintf(w, "func %s()", a.Name)
}
