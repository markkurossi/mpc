//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"

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
	Types       []*TypeInfo
	Constants   []*ConstantDef
	Functions   map[string]*Func
}

func NewPackage(name string) *Package {
	return &Package{
		Name:      name,
		Imports:   make(map[string]string),
		Functions: make(map[string]*Func),
	}
}

func (pkg *Package) Compile(packages map[string]*Package, logger *utils.Logger,
	params *utils.Params) (*circuit.Circuit, Annotations, error) {

	main, ok := pkg.Functions["main"]
	if !ok {
		return nil, nil, logger.Errorf(utils.Point{},
			"no main function defined")
	}

	gen := ssa.NewGenerator(params)
	ctx := NewCodegen(logger, pkg, packages, params.Verbose)

	// Init package.
	err := pkg.Init(packages, ctx, gen)
	if err != nil {
		return nil, nil, err
	}

	ctx.PushCompilation(gen.Block(), gen.Block(), nil, main)
	ctx.Start().Bindings = pkg.Bindings.Clone()

	// Arguments.
	var inputs circuit.IO
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

		arg := circuit.IOArg{
			Name: a.String(),
			Type: a.Type.String(),
			Size: a.Type.Bits,
		}
		if typeInfo.Type == types.Struct {
			arg.Combound = flattenStruct(typeInfo)
		}

		inputs = append(inputs, arg)
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

	// Return values
	var outputs circuit.IO
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
		outputs = append(outputs, circuit.IOArg{
			Name: v.String(),
			Type: v.Type.String(),
			Size: v.Type.Bits,
		})
	}

	cc, err := circuits.NewCompiler(inputs, outputs)
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
	if params.OptPruneGates {
		pruned := cc.Prune()
		if params.Verbose {
			fmt.Printf(" - Pruned %d gates\n", pruned)
		}
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

func flattenStruct(t types.Info) circuit.IO {
	var result circuit.IO
	if t.Type != types.Struct {
		return result
	}

	for _, f := range t.Struct {
		// XXX recursive structs.
		result = append(result, circuit.IOArg{
			Name: f.Name,
			Type: f.Type.String(),
			Size: f.Type.Bits,
		})
	}

	return result
}

func (pkg *Package) Init(packages map[string]*Package, ctx *Codegen,
	gen *ssa.Generator) error {

	if pkg.Initialized {
		return nil
	}
	pkg.Initialized = true
	if ctx.Verbose {
		fmt.Printf("Initializing %s\n", pkg.Name)
	}

	// Imported packages.
	for alias, name := range pkg.Imports {
		p, ok := packages[alias]
		if !ok {
			return fmt.Errorf("imported and not used: \"%s\"", name)
		}
		err := p.Init(packages, ctx, gen)
		if err != nil {
			return err
		}
	}

	// Define types.
	for _, typeDef := range pkg.Types {
		err := pkg.defineType(typeDef, ctx, gen)
		if err != nil {
			return err
		}
	}

	// Define constants.

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

func (pkg *Package) defineType(def *TypeInfo, ctx *Codegen,
	gen *ssa.Generator) error {

	fmt.Printf("Defining type %s\n", def)
	env := &Env{
		Bindings: pkg.Bindings,
	}
	switch def.Type {
	case TypeStruct:
		_, ok := pkg.Bindings.Get(def.TypeName)
		if ok {
			return fmt.Errorf("type %s already defined", def.TypeName)
		}
		// Construct compound type.
		var fields []types.StructField
		var bits int
		var minBits int
		var offset int
		for _, field := range def.StructFields {
			info, err := field.Type.Resolve(env, ctx, gen)
			if err != nil {
				return err
			}
			field := types.StructField{
				Name: field.Name,
				Type: info,
			}
			field.Type.Offset = offset
			fields = append(fields, field)

			bits += info.Bits
			minBits += info.MinBits
			offset += info.Bits
		}
		info := types.Info{
			Type:    types.Struct,
			Bits:    bits,
			MinBits: minBits,
			Struct:  fields,
		}

		v, err := ssa.Constant(info)
		if err != nil {
			return err
		}
		lval, err := gen.NewVar(def.TypeName, info, ctx.Scope())
		if err != nil {
			return err
		}
		pkg.Bindings.Set(lval, &v)
		return nil

	case TypeAlias:
		_, ok := pkg.Bindings.Get(def.TypeName)
		if ok {
			return fmt.Errorf("type %s already defined", def.TypeName)
		}
		info, err := def.AliasType.Resolve(env, ctx, gen)
		if err != nil {
			return err
		}
		v, err := ssa.Constant(info)
		if err != nil {
			return err
		}
		lval, err := gen.NewVar(def.TypeName, info, ctx.Scope())
		if err != nil {
			return err
		}
		pkg.Bindings.Set(lval, &v)
		return nil

	default:
		return fmt.Errorf("invalid type definition: %s", def)
	}
}
