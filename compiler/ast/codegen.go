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

	"github.com/markkurossi/mpc/compiler/circuits"
)

func (ast List) Compile(compiler *circuits.Compiler,
	out []*circuits.Wire) error {
	return fmt.Errorf("List.Compile() not implemented yet")
}

func (f *Func) Compile(compiler *circuits.Compiler,
	out []*circuits.Wire) error {
	for _, ast := range f.Body {
		err := ast.Compile(compiler, out)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ast Return) Compile(compiler *circuits.Compiler,
	out []*circuits.Wire) error {
	return ast.Expr.Compile(compiler, out)
}

func (ast Binary) Compile(compiler *circuits.Compiler,
	out []*circuits.Wire) error {

	l, err := getWires(ast.Left)
	if err != nil {
		return err
	}
	r, err := getWires(ast.Right)
	if err != nil {
		return err
	}
	switch ast.Op {
	case BinaryPlus:

	case BinaryMult:
		err := circuits.NewMultiplier(compiler, l, r, out)
		if err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("Binary.Compile not implemented yet")
}

func getWires(ast AST) ([]*circuits.Wire, error) {
	switch a := ast.(type) {
	case *VariableRef:
		return a.Var.Wires, nil

	default:
		return nil, fmt.Errorf("getWires %T not implemented yet", a)
	}
}

func (ast VariableRef) Compile(compiler *circuits.Compiler,
	out []*circuits.Wire) error {
	return fmt.Errorf("VariableRef.Compile() not implemented yet")
}
