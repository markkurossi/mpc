//
// parser.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"fmt"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/ast"
	"github.com/markkurossi/mpc/compiler/circuits"
)

func (unit *Unit) Compile() (*circuit.Circuit, error) {
	main, ok := unit.Functions["main"]
	if !ok {
		return nil, fmt.Errorf("No main function")
	}
	if len(main.Args) != 2 {
		return nil, fmt.Errorf("Only 2 argument main() supported")
	}
	var returnBits int
	for _, rt := range main.Return {
		returnBits += rt.Type.Bits
	}

	n1 := main.Args[0].Type.Bits
	n2 := main.Args[1].Type.Bits

	fmt.Printf("n1=%d, n2=%d, n3=%d\n", n1, n2, returnBits)

	c := circuits.NewCompiler(n1, n2, returnBits)

	main.Args[0].Wires = c.Inputs[0:n1]
	main.Args[1].Wires = c.Inputs[n1:]

	var index int
	for _, rt := range main.Return {
		rt.Wires = c.Outputs[index : index+rt.Type.Bits]
		index += rt.Type.Bits
	}

	// Resolve variables
	for _, f := range unit.Functions {
		vars := make(map[string]*ast.Variable)
		for _, arg := range f.Args {
			vars[arg.Name] = arg
		}
		err := f.Visit(func(a ast.AST) error {
			ref, ok := a.(*ast.VariableRef)
			if !ok {
				return nil
			}
			v, ok := vars[ref.Name]
			if !ok {
				return fmt.Errorf("Unknown variable %s", ref.Name)
			}
			ref.Var = v
			return nil
		}, func(a ast.AST) error { return nil })
		if err != nil {
			return nil, err
		}
	}

	_, err := main.Compile(c, c.Outputs)
	if err != nil {
		return nil, err
	}

	return c.Compile(), nil
}
