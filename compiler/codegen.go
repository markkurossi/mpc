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
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/ast"
	"github.com/markkurossi/mpc/compiler/circuits"
)

func (unit *Unit) Compile() (*circuit.Circuit, error) {
	main, ok := unit.Functions["main"]
	if !ok {
		return nil, fmt.Errorf("No main function")
	}

	var args circuit.IO
	for _, arg := range main.Args {
		args = append(args, circuit.IOArg{
			Name: arg.Name,
			Type: arg.Type.String(),
			Size: arg.Type.Bits,
		})
	}

	// Split arguments into garbler and evaluator arguments.
	var separatorSeen bool
	var n1, n2 circuit.IO
	for _, a := range args {
		if !separatorSeen {
			if !strings.HasPrefix(a.Name, "e") {
				n1 = append(n1, a)
				continue
			}
			separatorSeen = true
		}
		n2 = append(n2, a)
	}
	if !separatorSeen {
		if len(args) != 2 {
			return nil, fmt.Errorf("Can't split arguments: %s", args)
		}
		n1 = args[0:1]
		n2 = args[1:]
	}

	var n3 circuit.IO
	for _, rt := range main.Return {
		n3 = append(n3, circuit.IOArg{
			Name: rt.Name,
			Type: rt.Type.String(),
			Size: rt.Type.Bits,
		})
	}

	c := circuits.NewCompiler(n1, n2, n3)

	var w int
	for idx, arg := range args {
		main.Args[idx].Wires = c.Inputs[w : w+arg.Size]
		w += arg.Size
	}

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
