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
	interface{}, error) {
	return nil, fmt.Errorf("List.Eval not implemented yet")
}

func (ast *Func) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, error) {
	return nil, fmt.Errorf("Func is not constant")
}

func (ast *VariableDef) Eval(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (interface{}, error) {
	return nil, fmt.Errorf("VariableDef is not constant")
}

func (ast *Assign) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, error) {
	val, err := ast.Expr.Eval(block, ctx, gen)
	if err != nil {
		return nil, err
	}

	constVal, err := ssa.Constant(val)
	gen.AddConstant(constVal)

	var lValue ssa.Variable
	if ast.Define {
		lValue, err = gen.NewVar(ast.Name, constVal.Type, ctx.Scope())
		if err != nil {
			return nil, err
		}
	} else {
		b, err := block.Bindings.Get(ast.Name)
		if err != nil {
			return nil, err
		}
		lValue, err = gen.NewVar(b.Name, b.Type, ctx.Scope())
		if err != nil {
			return nil, err
		}
	}
	block.Bindings.Set(lValue, &constVal)

	return constVal.ConstValue, nil
}

func (ast *If) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, error) {
	return nil, fmt.Errorf("If is not constant")
}

func (ast *Call) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, error) {

	// Resolve called.
	pkg, ok := ctx.Packages[ast.Name.Package]
	if !ok {
		return nil, ctx.logger.Errorf(ast.Loc, "package '%s' not found",
			ast.Name.Package)
	}
	called, ok := pkg.Functions[ast.Name.Name]
	if ok {
		return nil, ctx.logger.Errorf(ast.Loc, "%s is not constant", called)
	}
	// Check builtin functions.
	for _, bi := range builtins {
		if bi.Name != ast.Name.Name {
			continue
		}
		if bi.Type != BuiltinFunc {
			return nil, fmt.Errorf("builtin %s used as function", bi.Name)
		}
		if bi.Eval == nil {
			return nil, ctx.logger.Errorf(ast.Loc, "%s() is not constant",
				bi.Name)
		}
		return bi.Eval(ast.Exprs, block, ctx, gen, ast.Location())
	}

	return nil, fmt.Errorf("%s() is not constant", ast.Name.Name)
}

func (ast *Return) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, error) {
	return nil, fmt.Errorf("Return is not constant")
}

func (ast *For) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, error) {
	return nil, fmt.Errorf("For is not constant")
}

func (ast *Binary) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, error) {
	l, err := ast.Left.Eval(block, ctx, gen)
	if err != nil {
		return nil, err
	}
	r, err := ast.Right.Eval(block, ctx, gen)
	if err != nil {
		return nil, err
	}

	switch lval := l.(type) {
	case int32:
		var rval int32
		switch rv := r.(type) {
		case int32:
			rval = rv
		default:
			return nil, ctx.logger.Errorf(ast.Right.Location(),
				"invalid r-value %T %s %T", lval, ast.Op, rv)
		}
		switch ast.Op {
		case BinaryMult:
			return lval * rval, nil
		case BinaryDiv:
			return lval / rval, nil
		case BinaryMod:
			return lval % rval, nil
		case BinaryLshift:
			if rval > 31 {
				return nil, fmt.Errorf("int32 overflow in %T %s %v",
					lval, ast.Op, rval)
			}
			return lval << rval, nil
		case BinaryRshift:
			return lval << rval, nil

		case BinaryPlus:
			return lval + rval, nil
		case BinaryMinus:
			return lval - rval, nil

		case BinaryEq:
			return lval == rval, nil
		case BinaryNeq:
			return lval != rval, nil
		case BinaryLt:
			return lval < rval, nil
		case BinaryLe:
			return lval <= rval, nil
		case BinaryGt:
			return lval > rval, nil
		case BinaryGe:
			return lval >= rval, nil
		default:
			return nil, ctx.logger.Errorf(ast.Right.Location(),
				"Binary.Eval '%T %s %T' not implemented yet", l, ast.Op, r)
		}

	case uint64:
		var rval uint64
		switch rv := r.(type) {
		case uint64:
			rval = rv
		default:
			return nil, ctx.logger.Errorf(ast.Right.Location(),
				"%T: invalid r-value %v (%T)", lval, rv, rv)
		}
		switch ast.Op {
		case BinaryMult:
			return lval * rval, nil
		case BinaryDiv:
			return lval / rval, nil
		case BinaryMod:
			return lval % rval, nil
		case BinaryLshift:
			return lval << rval, nil
		case BinaryRshift:
			return lval << rval, nil

		case BinaryPlus:
			return lval + rval, nil
		case BinaryMinus:
			return lval - rval, nil

		case BinaryEq:
			return lval == rval, nil
		case BinaryNeq:
			return lval != rval, nil
		case BinaryLt:
			return lval < rval, nil
		case BinaryLe:
			return lval <= rval, nil
		case BinaryGt:
			return lval > rval, nil
		case BinaryGe:
			return lval >= rval, nil
		default:
			return nil, ctx.logger.Errorf(ast.Right.Location(),
				"Binary.Eval '%T %s %T' not implemented yet", l, ast.Op, r)
		}

	default:
		return nil, ctx.logger.Errorf(ast.Left.Location(),
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
	gen *ssa.Generator) (interface{}, error) {

	var b ssa.Binding
	var err error

	if len(ast.Name.Package) > 0 {
		pkg, ok := ctx.Packages[ast.Name.Package]
		if !ok {
			return nil, ctx.logger.Errorf(ast.Loc, "package '%s' not found",
				ast.Name.Package)
		}
		b, err = pkg.Bindings.Get(ast.Name.Name)
	} else {
		b, err = block.Bindings.Get(ast.Name.Name)
	}
	if err != nil {
		return nil, err
	}

	val, ok := b.Bound.(*ssa.Variable)
	if !ok || !val.Const {
		return nil, fmt.Errorf("value %v is not constant", b.Bound)
	}

	return val.ConstValue, nil
}

func (ast *Constant) Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	interface{}, error) {
	return ast.Value, nil
}
