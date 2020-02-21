//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/ast"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/utils"
)

func Compile(data string) (*circuit.Circuit, error) {
	return compile("{data}", strings.NewReader(data))
}

func CompileFile(file string) (*circuit.Circuit, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	return compile(file, f)
}

func compileCircuit(name string, in io.Reader) (*circuit.Circuit, error) {
	logger := utils.NewLogger(name, os.Stdout)
	parser := NewParser(logger, in)
	_, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("not implemented yet")
}

func compile(name string, in io.Reader) (*circuit.Circuit, error) {
	logger := utils.NewLogger(name, os.Stdout)
	parser := NewParser(logger, in)
	unit, err := parser.Parse()
	if err != nil {
		return nil, err
	}
	return unit.Compile(logger)
}

func (unit *Unit) Compile(logger *utils.Logger) (*circuit.Circuit, error) {
	main, ok := unit.Functions["main"]
	if !ok {
		return nil, logger.Errorf(utils.Point{}, "no main function defined\n")
	}

	gen := ssa.NewGenerator()
	ctx := ast.NewCodegen(logger)

	ctx.BlockHead = gen.Block()
	ctx.BlockTail = gen.Block()

	_, err := main.SSA(ctx.BlockHead, ctx, gen)
	if err != nil {
		return nil, err
	}

	if false {
		ssa.PP(os.Stdout, ctx.BlockHead)
		ssa.Dot(os.Stdout, ctx.BlockHead)
	}

	// Arguments.
	var args circuit.IO
	for _, arg := range main.Args {
		v, ok := main.Bindings[arg.Name]
		if !ok {
			return nil, fmt.Errorf("argument %s not bound", arg.Name)
		}
		args = append(args, circuit.IOArg{
			Name: v.String(),
			Type: v.Type.String(),
			Size: v.Type.Bits,
		})
	}

	// Split arguments into garbler and evaluator arguments.
	var separatorSeen bool
	var g, e circuit.IO
	for _, a := range args {
		if !separatorSeen {
			if !strings.HasPrefix(a.Name, "e") {
				g = append(g, a)
				continue
			}
			separatorSeen = true
		}
		e = append(e, a)
	}
	if !separatorSeen {
		if len(args) != 2 {
			return nil, fmt.Errorf("can't split arguments: %s", args)
		}
		g = circuit.IO{args[0]}
		e = circuit.IO{args[1]}
	}

	// Return values
	var r circuit.IO
	for _, rt := range main.Return {
		v, ok := main.Bindings[rt.Name]
		if !ok {
			return nil, fmt.Errorf("return value %s not bound", rt.Name)
		}
		r = append(r, circuit.IOArg{
			Name: v.String(),
			Type: v.Type.String(),
			Size: v.Type.Bits,
		})
	}

	cc := circuits.NewCompiler(g, e, r)

	err = gen.DefineConstants(cc)
	if err != nil {
		return nil, err
	}

	err = ctx.BlockHead.Circuit(gen, cc)
	if err != nil {
		return nil, err
	}

	circ := cc.Compile()
	if false {
		fmt.Printf(" => %s\n", circ)
		circ.Marshal(os.Stdout)
	}

	return circ, nil
}
