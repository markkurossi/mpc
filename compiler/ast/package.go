//
// Copyright (c) 2020-2021 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
)

// Package implements a MPCL package.
type Package struct {
	Name        string
	Initialized bool
	Imports     map[string]string
	Bindings    ssa.Bindings
	Types       []*TypeInfo
	Constants   []*ConstantDef
	Variables   []*VariableDef
	Functions   map[string]*Func
}

// NewPackage creates a new package.
func NewPackage(name string) *Package {
	return &Package{
		Name:      name,
		Imports:   make(map[string]string),
		Functions: make(map[string]*Func),
	}
}

// Compile compiles the package.
func (pkg *Package) Compile(packages map[string]*Package, logger *utils.Logger,
	params *utils.Params) (*ssa.Program, Annotations, error) {

	main, ok := pkg.Functions["main"]
	if !ok {
		return nil, nil, logger.Errorf(utils.Point{},
			"no main function defined")
	}

	gen := ssa.NewGenerator(params)
	ctx := NewCodegen(logger, pkg, packages, params.Verbose)

	// Init is the program start point.
	init := gen.Block()

	// Init package.
	block, err := pkg.Init(packages, init, ctx, gen)
	if err != nil {
		return nil, nil, err
	}

	// Main block derives package's bindings from block with NextBlock().
	ctx.PushCompilation(gen.NextBlock(block), gen.Block(), nil, main)

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
			arg.Compound = flattenStruct(typeInfo)
		}

		inputs = append(inputs, arg)
	}

	// Compile main.
	_, returnVars, err := main.SSA(ctx.Start(), ctx, gen)
	if err != nil {
		return nil, nil, err
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

	steps := init.Serialize()

	program, err := ssa.NewProgram(params, inputs, outputs, gen.Constants(),
		steps)
	if err != nil {
		return nil, nil, err
	}
	err = program.Peephole()
	if err != nil {
		return nil, nil, err
	}
	program.GC()

	if params.SSAOut != nil {
		program.PP(params.SSAOut)
	}
	if params.SSADotOut != nil {
		ssa.Dot(params.SSADotOut, init)
	}

	return program, main.Annotations, nil
}

func flattenStruct(t types.Info) circuit.IO {
	var result circuit.IO
	if t.Type != types.Struct {
		return result
	}

	for _, f := range t.Struct {
		if f.Type.Type == types.Struct {
			ios := flattenStruct(f.Type)
			result = append(result, ios...)
		} else {
			result = append(result, circuit.IOArg{
				Name: f.Name,
				Type: f.Type.String(),
				Size: f.Type.Bits,
			})
		}
	}

	return result
}

// Init initializes the package.
func (pkg *Package) Init(packages map[string]*Package, block *ssa.Block,
	ctx *Codegen, gen *ssa.Generator) (*ssa.Block, error) {

	if pkg.Initialized {
		return block, nil
	}
	pkg.Initialized = true
	if ctx.Verbose {
		fmt.Printf("Initializing %s\n", pkg.Name)
	}

	// Imported packages.
	for alias, name := range pkg.Imports {
		p, ok := packages[alias]
		if !ok {
			return nil, fmt.Errorf("imported and not used: \"%s\"", name)
		}
		var err error
		block, err = p.Init(packages, block, ctx, gen)
		if err != nil {
			return nil, err
		}
	}

	// Define types.
	for _, typeDef := range pkg.Types {
		err := pkg.defineType(typeDef, ctx, gen)
		if err != nil {
			return nil, err
		}
	}

	// Package initializer block.
	block = gen.NextBlock(block)
	block.Name = fmt.Sprintf(".%s", pkg.Name)

	// Package sees only its bindings.
	block.Bindings = pkg.Bindings.Clone()

	var err error

	// Define constants.
	for _, def := range pkg.Constants {
		block, _, err = def.SSA(block, ctx, gen)
		if err != nil {
			return nil, err
		}
	}

	// Define variables.
	for _, def := range pkg.Variables {
		block, _, err = def.SSA(block, ctx, gen)
		if err != nil {
			return nil, err
		}
	}

	pkg.Bindings = block.Bindings

	return block, nil
}

func (pkg *Package) defineType(def *TypeInfo, ctx *Codegen,
	gen *ssa.Generator) error {

	_, ok := pkg.Bindings.Get(def.TypeName)
	if ok {
		return ctx.logger.Errorf(def.Loc, "type %s already defined",
			def.TypeName)
	}
	env := &Env{
		Bindings: pkg.Bindings,
	}
	var info types.Info
	var err error

	switch def.Type {
	case TypeStruct:
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
		info = types.Info{
			Type:    types.Struct,
			Bits:    bits,
			MinBits: minBits,
			Struct:  fields,
		}

	case TypeArray:
		constSize, ok, err := def.ArrayLength.Eval(env, ctx, gen)
		if err != nil {
			return err
		}
		if !ok {
			return ctx.logger.Errorf(def.ArrayLength.Location(),
				"array size is not constant: %s", def.ArrayLength)
		}
		var size int
		switch s := constSize.(type) {
		case int:
			size = s
		case int32:
			size = int(s)
		case uint64:
			size = int(s)
		default:
			return ctx.logger.Errorf(def.ArrayLength.Location(),
				"invalid array size: %s", def.ArrayLength)
		}
		elInfo, err := def.ElementType.Resolve(env, ctx, gen)
		if err != nil {
			return err
		}
		info = types.Info{
			Type:    types.Array,
			Bits:    size * elInfo.Bits,
			MinBits: size * elInfo.MinBits,
			Element: &elInfo,
		}

	case TypeAlias:
		info, err = def.AliasType.Resolve(env, ctx, gen)
		if err != nil {
			return err
		}

	default:
		return ctx.logger.Errorf(def.Loc, "invalid type definition: %s", def)
	}

	v, err := ssa.Constant(gen, info)
	if err != nil {
		return err
	}
	lval, err := gen.NewVar(def.TypeName, info, ctx.Scope())
	if err != nil {
		return err
	}
	pkg.Bindings.Set(lval, &v)
	return nil
}
