//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/utils"
)

type Package struct {
	Name       string
	Imports    map[string]string
	Bindings   ssa.Bindings
	Functions  map[string]*Func
	References map[string]string
}

func NewPackage(name string) *Package {
	return &Package{
		Name:       name,
		Imports:    make(map[string]string),
		Functions:  make(map[string]*Func),
		References: make(map[string]string),
	}
}

func (unit *Package) Compile(packages map[string]*Package, logger *utils.Logger,
	params *utils.Params) (*circuit.Circuit, Annotations, error) {

	main, ok := unit.Functions["main"]
	if !ok {
		return nil, nil, logger.Errorf(utils.Point{},
			"no main function defined")
	}

	gen := ssa.NewGenerator(params.Verbose)
	ctx := NewCodegen(logger, packages, params.Verbose)

	ctx.PushCompilation(gen.Block(), gen.Block(), nil, main)
	ctx.Start().Bindings = unit.Bindings.Clone()

	// Compile main.
	if params.Verbose {
		fmt.Printf("main.SSA()...\n")
	}
	_, returnVars, err := main.SSA(ctx.Start(), ctx, gen)
	if err != nil {
		return nil, nil, err
	}
	if params.Verbose {
		fmt.Printf("main.SSA() done\n")
	}

	if params.SSAOut != nil {
		ssa.PP(params.SSAOut, ctx.Start())
	}
	if params.SSADotOut != nil {
		ssa.Dot(params.SSADotOut, ctx.Start())
	}

	// Arguments.
	var args circuit.IO
	for _, arg := range main.Args {
		v, ok := main.Bindings[arg.Name]
		if !ok {
			return nil, nil, fmt.Errorf("argument %s not bound", arg.Name)
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
			return nil, nil, fmt.Errorf("can't split arguments: %s", args)
		}
		g = circuit.IO{args[0]}
		e = circuit.IO{args[1]}
	}

	// Return values
	var r circuit.IO
	for idx, rt := range main.Return {
		_, ok := main.Bindings[rt.Name]
		if !ok {
			return nil, nil, fmt.Errorf("return value %s not bound", rt.Name)
		}
		v := returnVars[idx]
		r = append(r, circuit.IOArg{
			Name: v.String(),
			Type: v.Type.String(),
			Size: v.Type.Bits,
		})
	}

	cc, err := circuits.NewCompiler(g, e, r)
	if err != nil {
		return nil, nil, err
	}

	err = gen.DefineConstants(cc)
	if err != nil {
		return nil, nil, err
	}

	err = ctx.Start().Circuit(gen, cc)
	if err != nil {
		return nil, nil, err
	}

	circ := cc.Compile()
	if params.CircOut != nil {
		circ.Marshal(params.CircOut)
	}
	if params.CircDotOut != nil {
		circ.Dot(params.CircDotOut)
	}

	return circ, main.Annotations, nil
}
