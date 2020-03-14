//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"
	"math/big"

	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/utils"
)

func (ast List) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	return nil, false, fmt.Errorf("List.Eval not implemented yet")
}

func (ast *Func) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	return nil, false, nil
}

func (ast *VariableDef) Eval(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (interface{}, bool, error) {
	return nil, false, nil
}

func (ast *Assign) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	val, ok, err := ast.Expr.Eval(block, ctx, gen)
	if err != nil || !ok {
		return nil, ok, err
	}

	constVal, err := ssa.Constant(val)
	gen.AddConstant(constVal)

	var lValue ssa.Variable
	if ast.Define {
		lValue, err = gen.NewVar(ast.Name, constVal.Type, ctx.Scope())
		if err != nil {
			return nil, false, err
		}
	} else {
		b, ok := block.Bindings.Get(ast.Name)
		if !ok {
			return nil, false, ctx.logger.Errorf(ast.Loc,
				"undefined variable '%s'", ast.Name)
		}
		lValue, err = gen.NewVar(b.Name, b.Type, ctx.Scope())
		if err != nil {
			return nil, false, err
		}
	}
	block.Bindings.Set(lValue, &constVal)

	return constVal.ConstValue, true, nil
}

func (ast *If) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	return nil, false, nil
}

func (ast *Call) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {

	// Resolve called.
	pkg, ok := ctx.Packages[ast.Name.Package]
	if !ok {
		return nil, false, ctx.logger.Errorf(ast.Loc, "package '%s' not found",
			ast.Name.Package)
	}
	_, ok = pkg.Functions[ast.Name.Name]
	if ok {
		return nil, false, nil
	}
	// Check builtin functions.
	for _, bi := range builtins {
		if bi.Name != ast.Name.Name {
			continue
		}
		if bi.Type != BuiltinFunc {
			return nil, false, fmt.Errorf("builtin %s used as function",
				bi.Name)
		}
		if bi.Eval == nil {
			return nil, false, nil
		}
		return bi.Eval(ast.Exprs, block, ctx, gen, ast.Location())
	}

	return nil, false, nil
}

func (ast *Return) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	return nil, false, nil
}

func (ast *For) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	return nil, false, nil
}

func (ast *Binary) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	l, ok, err := ast.Left.Eval(block, ctx, gen)
	if err != nil || !ok {
		return nil, ok, err
	}
	r, ok, err := ast.Right.Eval(block, ctx, gen)
	if err != nil || !ok {
		return nil, ok, err
	}

	switch lval := l.(type) {
	case int32:
		var rval int32
		switch rv := r.(type) {
		case int32:
			rval = rv
		default:
			return nil, false, ctx.logger.Errorf(ast.Right.Location(),
				"invalid r-value %T %s %T", lval, ast.Op, rv)
		}
		switch ast.Op {
		case BinaryMult:
			return lval * rval, true, nil
		case BinaryDiv:
			return lval / rval, true, nil
		case BinaryMod:
			return lval % rval, true, nil
		case BinaryLshift:
			return lval << rval, true, nil
		case BinaryRshift:
			return lval >> rval, true, nil

		case BinaryPlus:
			return lval + rval, true, nil
		case BinaryMinus:
			return lval - rval, true, nil

		case BinaryEq:
			return lval == rval, true, nil
		case BinaryNeq:
			return lval != rval, true, nil
		case BinaryLt:
			return lval < rval, true, nil
		case BinaryLe:
			return lval <= rval, true, nil
		case BinaryGt:
			return lval > rval, true, nil
		case BinaryGe:
			return lval >= rval, true, nil
		default:
			return nil, false, ctx.logger.Errorf(ast.Right.Location(),
				"Binary.Eval '%T %s %T' not implemented yet", l, ast.Op, r)
		}

	case uint64:
		var rval uint64
		switch rv := r.(type) {
		case uint64:
			rval = rv
		default:
			return nil, false, ctx.logger.Errorf(ast.Right.Location(),
				"%T: invalid r-value %v (%T)", lval, rv, rv)
		}
		switch ast.Op {
		case BinaryMult:
			return lval * rval, true, nil
		case BinaryDiv:
			return lval / rval, true, nil
		case BinaryMod:
			return lval % rval, true, nil
		case BinaryLshift:
			return lval << rval, true, nil
		case BinaryRshift:
			return lval >> rval, true, nil

		case BinaryPlus:
			return lval + rval, true, nil
		case BinaryMinus:
			return lval - rval, true, nil

		case BinaryEq:
			return lval == rval, true, nil
		case BinaryNeq:
			return lval != rval, true, nil
		case BinaryLt:
			return lval < rval, true, nil
		case BinaryLe:
			return lval <= rval, true, nil
		case BinaryGt:
			return lval > rval, true, nil
		case BinaryGe:
			return lval >= rval, true, nil
		default:
			return nil, false, ctx.logger.Errorf(ast.Right.Location(),
				"Binary.Eval '%T %s %T' not implemented yet", l, ast.Op, r)
		}

	default:
		return nil, false, ctx.logger.Errorf(ast.Left.Location(),
			"invalid l-value %v (%T)", lval, lval)
	}
}

func bigInt(i interface{}, ctx *Codegen, loc utils.Point) (*big.Int, error) {
	switch val := i.(type) {
	case int:
		return big.NewInt(int64(val)), nil

	default:
		return nil, ctx.logger.Errorf(loc,
			"invalid value %v (%T)", val, val)
	}
}

func (ast *VariableRef) Eval(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (interface{}, bool, error) {

	var b ssa.Binding
	var ok bool

	if len(ast.Name.Package) > 0 {
		pkg, ok := ctx.Packages[ast.Name.Package]
		if !ok {
			return nil, false, ctx.logger.Errorf(ast.Loc,
				"package '%s' not found", ast.Name.Package)
		}
		b, ok = pkg.Bindings.Get(ast.Name.Name)
	} else {
		b, ok = block.Bindings.Get(ast.Name.Name)
	}
	if !ok {
		return nil, false, ctx.logger.Errorf(ast.Loc,
			"undefined variable '%s'", ast.Name.String())
	}

	val, ok := b.Bound.(*ssa.Variable)
	if !ok || !val.Const {
		return nil, false, nil
	}

	return val.ConstValue, true, nil
}

func (ast *Constant) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	return ast.Value, true, nil
}
