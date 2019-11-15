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

func MakeWires(size int) []*circuits.Wire {
	result := make([]*circuits.Wire, size)
	for i := 0; i < size; i++ {
		result[i] = circuits.NewWire()
	}
	return result
}

func (ast List) Compile(compiler *circuits.Compiler,
	out []*circuits.Wire) ([]*circuits.Wire, error) {
	return nil, fmt.Errorf("List.Compile() not implemented yet")
}

func (f *Func) Compile(compiler *circuits.Compiler,
	out []*circuits.Wire) ([]*circuits.Wire, error) {

	var outputs []*circuits.Wire
	var err error

	for _, ast := range f.Body {
		outputs, err = ast.Compile(compiler, out)
		if err != nil {
			return nil, err
		}
	}
	return outputs, nil
}

func (ast Return) Compile(compiler *circuits.Compiler,
	out []*circuits.Wire) ([]*circuits.Wire, error) {
	return ast.Expr.Compile(compiler, out)
}

func (ast Binary) Compile(compiler *circuits.Compiler,
	out []*circuits.Wire) ([]*circuits.Wire, error) {

	l, err := ast.Left.Compile(compiler, nil)
	if err != nil {
		return nil, err
	}
	r, err := ast.Right.Compile(compiler, nil)
	if err != nil {
		return nil, err
	}

	switch ast.Op {
	case BinaryPlus:

	case BinaryMult:
		if out == nil {
			var size int
			if len(l) > len(r) {
				size = len(l) * 2
			} else {
				size = len(r) * 2
			}
			out = MakeWires(size)
		}
		err := circuits.NewMultiplier(compiler, l, r, out)
		if err != nil {
			return nil, err
		}
		return out, nil
	}

	return nil, fmt.Errorf("Binary.Compile not implemented yet")
}

func (ast VariableRef) Compile(compiler *circuits.Compiler,
	out []*circuits.Wire) ([]*circuits.Wire, error) {
	return ast.Var.Wires, nil
}
