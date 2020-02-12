//
// parser.go
//
// Copyright (c) 2019-2020 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"fmt"
	"os"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/ast"
	"github.com/markkurossi/mpc/compiler/ssa"
)

func (unit *Unit) Compile() (*circuit.Circuit, error) {
	main, ok := unit.Functions["main"]
	if !ok {
		return nil, fmt.Errorf("No main function")
	}

	output := ssa.NewGenerator()
	ctx := ast.NewCodegen()

	ctx.BlockHead = output.Block()
	ctx.BlockTail = output.Block()
	ctx.BlockTail.AddInstr(ssa.NewRetInstr())

	_, err := main.SSA(ctx.BlockHead, ctx, output)
	if err != nil {
		return nil, err
	}

	ssa.PP(os.Stdout, ctx.BlockHead)

	return nil, nil
}

func (unit *Unit) oldCompile() (*circuit.Circuit, error) {
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

	return nil, fmt.Errorf("Unit.Compile not implemented yet")
}
