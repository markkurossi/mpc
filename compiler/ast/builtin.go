//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"
	"path"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
)

// Builtin implements MPCL builtin elements.
type Builtin struct {
	Name string
	Type BuiltinType
	SSA  SSA
	Eval Eval
}

// BuiltinType specifies the different MPCL builtin elements.
type BuiltinType int

// Builtin types.
const (
	BuiltinFunc BuiltinType = iota
)

// SSA implements the builtin SSA generation.
type SSA func(block *ssa.Block, ctx *Codegen, gen *ssa.Generator,
	args []ssa.Variable, loc utils.Point) (*ssa.Block, []ssa.Variable, error)

// Eval implements the builtin evaluation in constant folding.
type Eval func(args []AST, env *Env, ctx *Codegen, gen *ssa.Generator,
	loc utils.Point) (ssa.Variable, bool, error)

// Predeclared identifiers.
var builtins = []Builtin{
	{
		Name: "copy",
		Type: BuiltinFunc,
		SSA:  copySSA,
	},
	{
		Name: "len",
		Type: BuiltinFunc,
		SSA:  lenSSA,
		Eval: lenEval,
	},
	{
		Name: "make",
		Type: BuiltinFunc,
		Eval: makeEval,
	},
	{
		Name: "native",
		Type: BuiltinFunc,
		SSA:  nativeSSA,
	},
	{
		Name: "size",
		Type: BuiltinFunc,
		SSA:  sizeSSA,
		Eval: sizeEval,
	},
}

func copySSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator,
	args []ssa.Variable, loc utils.Point) (*ssa.Block, []ssa.Variable, error) {

	if len(args) != 2 {
		return nil, nil, ctx.Errorf(loc,
			"invalid amount of arguments in call to copy")
	}
	dst := args[0]
	src := args[1]

	if dst.Type.Type != types.TArray {
		return nil, nil, ctx.Errorf(loc,
			"arguments to copy must be slices; have %s, %s",
			dst.Type.Type, src.Type.Type)
	}
	if src.Type.Type != types.TArray {
		return nil, nil, ctx.Errorf(loc,
			"second argument to copy should be slice or array")
	}
	if !dst.Type.ArrayElement.Equal(*src.Type.ArrayElement) {
		return nil, nil, ctx.Errorf(loc,
			"arguments to copy have different element types: %s and %s",
			dst.Type.ArrayElement, src.Type.ArrayElement)
	}

	lValue, err := gen.NewVar(dst.Name, dst.Type, ctx.Scope())
	if err != nil {
		return nil, nil, err
	}

	// If len(dst) > len(src): 	  amov  src dst 0 len(src)
	//    len(dst) < len(src): 	  slice src 0 len(dst) dst
	//    len(dst) = len(src): 	  move  src dst
	var copied int
	if dst.Type.ArraySize > src.Type.ArraySize {
		copied = src.Type.ArraySize
		fromConst, _, err := gen.Constant(int32(0), types.Uint32)
		if err != nil {
			return nil, nil, err
		}
		toConst, _, err := gen.Constant(int32(src.Type.Bits), types.Uint32)
		if err != nil {
			return nil, nil, err
		}
		block.AddInstr(ssa.NewAmovInstr(src, dst, fromConst, toConst, lValue))
	} else if dst.Type.ArraySize < src.Type.ArraySize {
		copied = dst.Type.ArraySize
		fromConst, _, err := gen.Constant(int32(0), types.Uint32)
		if err != nil {
			return nil, nil, err
		}
		toConst, _, err := gen.Constant(int32(dst.Type.Bits), types.Uint32)
		if err != nil {
			return nil, nil, err
		}
		block.AddInstr(ssa.NewSliceInstr(src, fromConst, toConst, lValue))
	} else {
		copied = dst.Type.ArraySize
		block.AddInstr(ssa.NewMovInstr(src, lValue))
	}
	block.Bindings.Set(lValue, nil)

	v, _, err := gen.Constant(int32(copied), types.Int32)
	if err != nil {
		return nil, nil, err
	}
	gen.AddConstant(v)

	return block, []ssa.Variable{v}, nil
}

func lenSSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator,
	args []ssa.Variable, loc utils.Point) (*ssa.Block, []ssa.Variable, error) {

	if len(args) != 1 {
		return nil, nil, ctx.Errorf(loc,
			"invalid amount of arguments in call to len")
	}

	var val int
	switch args[0].Type.Type {
	case types.TString:
		val = args[0].Type.Bits / types.ByteBits

	case types.TArray:
		val = args[0].Type.ArraySize

	default:
		return nil, nil, ctx.Errorf(loc, "invalid argument 1 (type %s) for len",
			args[0].Type)
	}

	v, _, err := gen.Constant(int32(val), types.Int32)
	if err != nil {
		return nil, nil, err
	}
	gen.AddConstant(v)

	return block, []ssa.Variable{v}, nil
}

func lenEval(args []AST, env *Env, ctx *Codegen, gen *ssa.Generator,
	loc utils.Point) (ssa.Variable, bool, error) {

	if len(args) != 1 {
		return ssa.Undefined, false, ctx.Errorf(loc,
			"invalid amount of arguments in call to len")
	}

	switch arg := args[0].(type) {
	case *VariableRef:
		var b ssa.Binding
		var ok bool

		if len(arg.Name.Package) > 0 {
			var pkg *Package
			pkg, ok = ctx.Packages[arg.Name.Package]
			if !ok {
				return ssa.Undefined, false, ctx.Errorf(loc,
					"package '%s' not found", arg.Name.Package)
			}
			b, ok = pkg.Bindings.Get(arg.Name.Name)
		} else {
			b, ok = env.Get(arg.Name.Name)
		}
		if !ok {
			return ssa.Undefined, false, ctx.Errorf(loc,
				"undefined variable '%s'", arg.Name.String())
		}

		switch b.Type.Type {
		case types.TString:
			return gen.Constant(int32(b.Type.Bits/types.ByteBits), types.Int32)

		case types.TArray:
			return gen.Constant(int32(b.Type.ArraySize), types.Int32)

		default:
			return ssa.Undefined, false, ctx.Errorf(loc,
				"invalid argument 1 (type %s) for len", b.Type)
		}

	default:
		return ssa.Undefined, false, ctx.Errorf(loc,
			"len(%v/%T) is not constant", arg, arg)
	}
}

func makeEval(args []AST, env *Env, ctx *Codegen, gen *ssa.Generator,
	loc utils.Point) (ssa.Variable, bool, error) {

	if len(args) != 2 {
		return ssa.Undefined, false, ctx.Errorf(loc,
			"invalid amount of argument in call to make")
	}
	ref, ok := args[0].(*VariableRef)
	if !ok {
		return ssa.Undefined, false, ctx.Errorf(args[0], "%s is not a type",
			args[0])
	}
	ti := TypeInfo{
		Type: TypeName,
		Name: ref.Name,
	}
	typeInfo, err := ti.Resolve(env, ctx, gen)
	if err != nil {
		return ssa.Undefined, false, ctx.Errorf(args[0], "%s is not a type",
			args[0])
	}
	if typeInfo.Bits != 0 {
		return ssa.Undefined, false, ctx.Errorf(args[0],
			"can't make specified type %s", typeInfo)
	}

	constVal, _, err := args[1].Eval(env, ctx, gen)
	if err != nil {
		return ssa.Undefined, false, ctx.Errorf(args[1], "%s", err)
	}

	var bits int
	switch val := constVal.ConstValue.(type) {
	case int32:
		bits = int(val)

	default:
		return ssa.Undefined, false, ctx.Errorf(loc,
			"non-integer (%T) len argument in make(%s)", val, typeInfo)
	}
	typeInfo.Bits = bits

	return gen.Constant(typeInfo, types.Undefined)
}

func nativeSSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator,
	args []ssa.Variable, loc utils.Point) (*ssa.Block, []ssa.Variable, error) {

	if len(args) < 1 {
		return nil, nil, ctx.Errorf(loc,
			"not enought argument in call to native")
	}
	name, ok := args[0].ConstValue.(string)
	if !args[0].Const || !ok {
		return nil, nil, ctx.Errorf(loc,
			"not enought argument in call to native")
	}
	// Our native name constant is not needed in the implementation.
	gen.RemoveConstant(args[0])
	args = args[1:]

	switch name {
	case "hamming":
		if len(args) != 2 {
			return nil, nil, ctx.Errorf(loc,
				"invalid amount of arguments in call to '%s'", name)
		}

		var typeInfo types.Info
		for _, arg := range args {
			if arg.Type.Bits > typeInfo.Bits {
				typeInfo = arg.Type
			}
		}

		v := gen.AnonVar(typeInfo)
		block.AddInstr(ssa.NewBuiltinInstr(circuits.Hamming, args[0], args[1],
			v))

		return block, []ssa.Variable{v}, nil

	default:
		if strings.HasSuffix(name, ".circ") {
			return nativeCircuit(name, block, ctx, gen, args, loc)
		}
		return nil, nil, ctx.Errorf(loc, "unknown native '%s'", name)
	}
}

func nativeCircuit(name string, block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator, args []ssa.Variable, loc utils.Point) (
	*ssa.Block, []ssa.Variable, error) {

	dir := path.Dir(loc.Source)
	fp := path.Join(dir, name)
	circ, err := circuit.Parse(fp)
	if err != nil {
		return nil, nil, ctx.Errorf(loc, "failed to parse circuit: %s", err)
	}

	if len(circ.Inputs) < len(args) {
		return nil, nil, ctx.Errorf(loc,
			"not enought argument in call to native")
	} else if len(circ.Inputs) < len(args) {
		return nil, nil, ctx.Errorf(loc, "too many argument in call to native")
	}
	// Check that the argument types match.
	for idx, io := range circ.Inputs {
		arg := args[idx]
		if io.Size < arg.Type.Bits || io.Size > arg.Type.Bits && !arg.Const {
			return nil, nil, ctx.Errorf(loc,
				"invalid argument %d for native circuit: got %s, need %d",
				idx, arg.Type, io.Size)
		}
	}

	if ctx.Verbose {
		fmt.Printf(" - native %s: %v\n", name, circ)
	}

	var result []ssa.Variable

	for _, io := range circ.Outputs {
		result = append(result, gen.AnonVar(types.Info{
			Type: types.TUndefined,
			Bits: io.Size,
		}))
	}

	block.AddInstr(ssa.NewCircInstr(args, circ, result))

	return block, result, nil
}

func sizeSSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator,
	args []ssa.Variable, loc utils.Point) (*ssa.Block, []ssa.Variable, error) {

	if len(args) != 1 {
		return nil, nil, ctx.Errorf(loc,
			"invalid amount of arguments in call to size")
	}

	v, _, err := gen.Constant(int32(args[0].Type.Bits), types.Int32)
	if err != nil {
		return nil, nil, err
	}
	gen.AddConstant(v)

	return block, []ssa.Variable{v}, nil
}

func sizeEval(args []AST, env *Env, ctx *Codegen, gen *ssa.Generator,
	loc utils.Point) (ssa.Variable, bool, error) {

	if len(args) != 1 {
		return ssa.Undefined, false, ctx.Errorf(loc,
			"invalid amount of arguments in call to size")
	}

	switch arg := args[0].(type) {
	case *VariableRef:
		var b ssa.Binding
		var ok bool

		if len(arg.Name.Package) > 0 {
			var pkg *Package
			pkg, ok = ctx.Packages[arg.Name.Package]
			if !ok {
				return ssa.Undefined, false, ctx.Errorf(loc,
					"package '%s' not found", arg.Name.Package)
			}
			b, ok = pkg.Bindings.Get(arg.Name.Name)
		} else {
			b, ok = env.Get(arg.Name.Name)
		}
		if !ok {
			return ssa.Undefined, false, ctx.Errorf(loc,
				"undefined variable '%s'", arg.Name.String())
		}
		return gen.Constant(int32(b.Type.Bits), types.Int32)

	default:
		return ssa.Undefined, false, ctx.Errorf(loc,
			"size(%v/%T) is not constant", arg, arg)
	}
}
