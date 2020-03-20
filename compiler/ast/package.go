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
	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
)

type Package struct {
	Name        string
	Initialized bool
	Imports     map[string]string
	Bindings    ssa.Bindings
	Constants   []*ConstantDef
	Functions   map[string]*Func
	References  map[string]string
}

func NewPackage(name string) *Package {
	return &Package{
		Name:       name,
		Imports:    make(map[string]string),
		Functions:  make(map[string]*Func),
		References: make(map[string]string),
	}
}

func (pkg *Package) Compile(packages map[string]*Package, logger *utils.Logger,
	params *utils.Params) (*circuit.Circuit, Annotations, error) {

	main, ok := pkg.Functions["main"]
	if !ok {
		return nil, nil, logger.Errorf(utils.Point{},
			"no main function defined")
	}

	gen := ssa.NewGenerator(params.Verbose)
	ctx := NewCodegen(logger, pkg, packages, params.Verbose)

	// Init package.
	err := pkg.Init(packages, ctx, gen)
	if err != nil {
		return nil, nil, err
	}

	ctx.PushCompilation(gen.Block(), gen.Block(), nil, main)
	ctx.Start().Bindings = pkg.Bindings.Clone()

	// Arguments.
	var args circuit.IO
	for _, arg := range main.Args {
		typeInfo, err := arg.Type.Resolve(NewEnv(ctx.Start()), ctx, gen)
		if err != nil {
			return nil, nil, ctx.logger.Errorf(arg.Loc,
				"invalid argument type: %s", err)
		}
		if typeInfo.Bits == 0 {
			return nil, nil,
				fmt.Errorf("argument %s of %s has unspecified type",
					arg.Name, main)
		}
		// Define argument in block.
		a, err := gen.NewVar(arg.Name, typeInfo, ctx.Scope())
		if err != nil {
			return nil, nil, err
		}
		ctx.Start().Bindings.Set(a, nil)

		args = append(args, circuit.IOArg{
			Name: a.String(),
			Type: a.Type.String(),
			Size: a.Type.Bits,
		})
	}

	// Compile main.
	_, returnVars, err := main.SSA(ctx.Start(), ctx, gen)
	if err != nil {
		return nil, nil, err
	}
	err = ssa.Peephole(ctx.Start())
	if err != nil {
		return nil, nil, err
	}

	if params.SSAOut != nil {
		ssa.PP(params.SSAOut, ctx.Start())
	}
	if params.SSADotOut != nil {
		ssa.Dot(params.SSADotOut, ctx.Start())
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
		if idx >= len(returnVars) {
			return nil, nil, fmt.Errorf("too few values for %s", main)
		}
		typeInfo, err := rt.Type.Resolve(NewEnv(ctx.Start()), ctx, gen)
		if err != nil {
			return nil, nil, ctx.logger.Errorf(rt.Loc,
				"invalid return type: %s", err)
		}
		if typeInfo.Bits == 0 {
			typeInfo.Bits = returnVars[idx].Type.Bits
		}
		if returnVars[idx].Type.Type == types.Undefined {
			returnVars[idx].Type.Type = typeInfo.Type
		}
		if !ssa.LValueFor(typeInfo, returnVars[idx]) {
			return nil, nil,
				fmt.Errorf("invalid value %v for return value %d of %s",
					returnVars[idx].Type, idx, main)
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

	if params.Verbose {
		fmt.Printf("Creating circuit...\n")
	}
	err = ctx.Start().Circuit(gen, cc)
	if err != nil {
		return nil, nil, err
	}

	if params.Verbose {
		fmt.Printf("Compiling circuit...\n")
	}
	circ := cc.Compile()
	if params.CircOut != nil {
		if params.Verbose {
			fmt.Printf("Serializing circuit...\n")
		}
		circ.Marshal(params.CircOut)
	}
	if params.CircDotOut != nil {
		circ.Dot(params.CircDotOut)
	}

	return circ, main.Annotations, nil
}

func (pkg *Package) Init(packages map[string]*Package, ctx *Codegen,
	gen *ssa.Generator) error {

	if pkg.Initialized {
		return nil
	}
	pkg.Initialized = true
	fmt.Printf("Initializing %s\n", pkg.Name)

	// Imported packages.
	for alias, name := range pkg.Imports {
		p, ok := packages[alias]
		if !ok {
			return fmt.Errorf("unknown package '%s'", name)
		}
		err := p.Init(packages, ctx, gen)
		if err != nil {
			return err
		}
	}

	block := gen.Block()
	block.Bindings = pkg.Bindings.Clone()

	for _, def := range pkg.Constants {
		var err error
		block, _, err = def.SSA(block, ctx, gen)
		if err != nil {
			return err
		}
	}
	pkg.Bindings = block.Bindings

	return nil
}
