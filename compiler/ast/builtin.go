//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"
	"os"
	"path"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
)

type Builtin struct {
	Name string
	Type BuiltinType
	SSA  SSA
	Eval Eval
}

type BuiltinType int

const (
	BuiltinFunc BuiltinType = iota
)

type SSA func(block *ssa.Block, ctx *Codegen, gen *ssa.Generator,
	args []ssa.Variable, loc utils.Point) (*ssa.Block, []ssa.Variable, error)

type Eval func(args []AST, block *ssa.Block, ctx *Codegen, gen *ssa.Generator,
	loc utils.Point) (interface{}, bool, error)

// Predeclared identifiers.
var builtins = []Builtin{
	Builtin{
		Name: "native",
		Type: BuiltinFunc,
		SSA:  nativeSSA,
	},
	Builtin{
		Name: "size",
		Type: BuiltinFunc,
		SSA:  sizeSSA,
		Eval: sizeEval,
	},
	Builtin{
		Name: "hamming",
		Type: BuiltinFunc,
		SSA:  hammingSSA,
	},
}

func nativeSSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator,
	args []ssa.Variable, loc utils.Point) (*ssa.Block, []ssa.Variable, error) {

	if len(args) < 1 {
		return nil, nil, ctx.logger.Errorf(loc,
			"not enought argument in call to native")
	}
	name, ok := args[0].ConstValue.(string)
	if !args[0].Const || !ok {
		return nil, nil, ctx.logger.Errorf(loc,
			"not enought argument in call to native")
	}
	// Our circuit name constant is not needed in the circuit
	// computation, it is here only as a reference to the circuit
	// file.
	gen.RemoveConstant(args[0])

	dir := path.Dir(loc.Source)
	fp := path.Join(dir, name)
	file, err := os.Open(fp)
	if err != nil {
		return nil, nil, ctx.logger.Errorf(loc,
			"failed to open circuit: %s", err)
	}
	defer file.Close()
	circ, err := circuit.Parse(file)
	if err != nil {
		return nil, nil, ctx.logger.Errorf(loc,
			"failed to parse circuit: %s", err)
	}

	var inputs circuit.IO
	inputs = append(inputs, circ.N1...)
	inputs = append(inputs, circ.N2...)

	if len(inputs) < len(args)-1 {
		return nil, nil, ctx.logger.Errorf(loc,
			"not enought argument in call to native")
	} else if len(inputs) < len(args)-1 {
		return nil, nil, ctx.logger.Errorf(loc,
			"too many argument in call to native")
	}
	// Check that the argument types match.
	for idx, io := range inputs {
		arg := args[idx+1]
		if io.Size != arg.Type.Bits {
			// Check if arg is const and smaller, can type convert.
			return nil, nil, ctx.logger.Errorf(loc,
				"invalid argument %d for native circuit: got %s, need %d",
				idx, arg.Type, io.Size)
		}
	}

	if ctx.Verbose {
		fmt.Printf(" - native %s: %v\n", name, circ)
	}

	var result []ssa.Variable

	for _, io := range circ.N3 {
		result = append(result, gen.AnonVar(types.Info{
			Type: types.Undefined,
			Bits: io.Size,
		}))
	}

	block.AddInstr(ssa.NewCircInstr(args[1:], circ, result))

	return block, result, nil
}

func sizeSSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator,
	args []ssa.Variable, loc utils.Point) (*ssa.Block, []ssa.Variable, error) {

	if len(args) != 1 {
		return nil, nil, ctx.logger.Errorf(loc,
			"invalid amount of arguments in call to size")
	}

	val := args[0].Type.Bits
	v := ssa.Variable{
		Name: ConstantName(val),
		Type: types.Info{
			Type: types.Int,
			Bits: val,
		},
		Const:      true,
		ConstValue: val,
	}
	gen.AddConstant(v)

	return block, []ssa.Variable{v}, nil
}

func sizeEval(args []AST, block *ssa.Block, ctx *Codegen, gen *ssa.Generator,
	loc utils.Point) (interface{}, bool, error) {

	if len(args) != 1 {
		return nil, false, ctx.logger.Errorf(loc,
			"invalid amount of arguments in call to size")
	}

	switch arg := args[0].(type) {
	case *VariableRef:
		var b ssa.Binding
		var ok bool

		if len(arg.Name.Package) > 0 {
			pkg, ok := ctx.Packages[arg.Name.Package]
			if !ok {
				return nil, false, ctx.logger.Errorf(loc,
					"package '%s' not found", arg.Name.Package)
			}
			b, ok = pkg.Bindings.Get(arg.Name.Name)
		} else {
			b, ok = block.Bindings.Get(arg.Name.Name)
		}
		if !ok {
			return nil, false, ctx.logger.Errorf(loc,
				"undefined variable '%s'", arg.Name.String())
		}
		return int32(b.Type.Bits), true, nil

	default:
		return nil, false, ctx.logger.Errorf(loc,
			"size(%v/%T) is not constant", arg, arg)
	}
}

func hammingSSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator,
	args []ssa.Variable, loc utils.Point) (*ssa.Block, []ssa.Variable, error) {

	if len(args) != 2 {
		return nil, nil, ctx.logger.Errorf(loc,
			"invalid amount of arguments in call to hamming")
	}

	// XXX Check the largest type.
	v := gen.AnonVar(args[0].Type)
	block.AddInstr(ssa.NewBuiltinInstr(circuits.Hamming, args[0], args[1], v))

	return block, []ssa.Variable{v}, nil
}
