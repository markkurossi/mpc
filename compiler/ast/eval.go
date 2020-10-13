//
// Copyright (c) 2019-2020 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"
	"math"
	"math/big"

	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/utils"
)

// Eval implements the compiler.ast.AST.Eval for list statements.
func (ast List) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	return nil, false, fmt.Errorf("List.Eval not implemented yet")
}

// Eval implements the compiler.ast.AST.Eval for function definitions.
func (ast *Func) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	return nil, false, nil
}

// Eval implements the compiler.ast.AST.Eval for constant definitions.
func (ast *ConstantDef) Eval(env *Env, ctx *Codegen,
	gen *ssa.Generator) (interface{}, bool, error) {
	return nil, false, nil
}

// Eval implements the compiler.ast.AST.Eval for variable definitions.
func (ast *VariableDef) Eval(env *Env, ctx *Codegen,
	gen *ssa.Generator) (interface{}, bool, error) {
	return nil, false, nil
}

// Eval implements the compiler.ast.AST.Eval for assignment expressions.
func (ast *Assign) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {

	var values []interface{}
	for _, expr := range ast.Exprs {
		val, ok, err := expr.Eval(env, ctx, gen)
		if err != nil || !ok {
			return nil, ok, err
		}
		// XXX multiple return values.
		values = append(values, val)
	}

	if len(ast.LValues) != len(values) {
		return nil, false, ctx.logger.Errorf(ast.Loc,
			"assignment mismatch: %d variables but %d values",
			len(ast.LValues), len(values))
	}

	for idx, lv := range ast.LValues {

		constVal, err := ssa.Constant(gen, values[idx])
		if err != nil {
			return nil, false, err
		}
		gen.AddConstant(constVal)

		ref, ok := lv.(*VariableRef)
		if !ok {
			return nil, false, ctx.logger.Errorf(ast.Loc,
				"cannot assign to %s", lv)
		}
		// XXX package.name below

		var lValue ssa.Variable
		if ast.Define {
			lValue, err = gen.NewVar(ref.Name.Name, constVal.Type, ctx.Scope())
			if err != nil {
				return nil, false, err
			}
		} else {
			b, ok := env.Get(ref.Name.Name)
			if !ok {
				return nil, false, ctx.logger.Errorf(ast.Loc,
					"undefined variable '%s'", ref.Name)
			}
			lValue, err = gen.NewVar(b.Name, b.Type, ctx.Scope())
			if err != nil {
				return nil, false, err
			}
		}
		env.Set(lValue, &constVal)
	}

	return values, true, nil
}

// Eval implements the compiler.ast.AST.Eval for if statements.
func (ast *If) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	return nil, false, nil
}

// Eval implements the compiler.ast.AST.Eval for call expressions.
func (ast *Call) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {

	// Resolve called.
	var pkgName string
	if len(ast.Ref.Name.Package) > 0 {
		pkgName = ast.Ref.Name.Package
	} else {
		pkgName = ast.Ref.Name.Defined
	}
	pkg, ok := ctx.Packages[pkgName]
	if !ok {
		return nil, false,
			ctx.logger.Errorf(ast.Loc, "package '%s' not found", pkgName)
	}
	_, ok = pkg.Functions[ast.Ref.Name.Name]
	if ok {
		return nil, false, nil
	}
	// Check builtin functions.
	for _, bi := range builtins {
		if bi.Name != ast.Ref.Name.Name {
			continue
		}
		if bi.Type != BuiltinFunc {
			return nil, false, fmt.Errorf("builtin %s used as function",
				bi.Name)
		}
		if bi.Eval == nil {
			return nil, false, nil
		}
		return bi.Eval(ast.Exprs, env, ctx, gen, ast.Location())
	}

	return nil, false, nil
}

// Eval implements the compiler.ast.AST.Eval for return statements.
func (ast *Return) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	return nil, false, nil
}

// Eval implements the compiler.ast.AST.Eval for for statements.
func (ast *For) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	return nil, false, nil
}

// Eval implements the compiler.ast.AST.Eval for binary expressions.
func (ast *Binary) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	l, ok, err := ast.Left.Eval(env, ctx, gen)
	if err != nil || !ok {
		return nil, ok, err
	}
	r, ok, err := ast.Right.Eval(env, ctx, gen)
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

// Eval implements the compiler.ast.AST.Eval for slice expressions.
func (ast *Slice) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {

	expr, ok, err := ast.Expr.Eval(env, ctx, gen)
	if err != nil || !ok {
		return nil, ok, err
	}

	from := 0
	to := math.MaxInt32

	if ast.From != nil {
		val, ok, err := ast.From.Eval(env, ctx, gen)
		if err != nil || !ok {
			return nil, ok, err
		}
		from, err = intVal(val)
		if err != nil {
			return nil, false,
				ctx.logger.Errorf(ast.From.Location(), err.Error())
		}
	}
	if ast.To != nil {
		val, ok, err := ast.To.Eval(env, ctx, gen)
		if err != nil || !ok {
			return nil, ok, err
		}
		to, err = intVal(val)
		if err != nil {
			return nil, false,
				ctx.logger.Errorf(ast.To.Location(), err.Error())
		}
	}
	if to <= from {
		return nil, false, ctx.logger.Errorf(ast.Expr.Location(),
			"invalid slice range %d:%d", from, to)
	}
	switch val := expr.(type) {
	case int32:
		if from >= 32 {
			return nil, false,
				ctx.logger.Errorf(ast.From.Location(),
					"slice bounds out of range [%d:32]", from)
		}
		tmp := uint32(val)
		tmp >>= from
		tmp &^= 0xffffffff << (to - from)
		return int32(tmp), ok, nil

	default:
		return nil, false, ctx.logger.Errorf(ast.Expr.Location(),
			"Slice.Eval: expr %T not implemented yet", val)
	}
}

func intVal(val interface{}) (int, error) {
	switch v := val.(type) {
	case int32:
		return int(v), nil
	default:
		return 0, fmt.Errorf("invalid slice index: %T", v)
	}
}

// Eval implements the compiler.ast.AST.Eval for variable references.
func (ast *VariableRef) Eval(env *Env, ctx *Codegen,
	gen *ssa.Generator) (interface{}, bool, error) {

	var b ssa.Binding
	var ok bool

	// Check if package name is bound to variable.
	b, ok = env.Get(ast.Name.Package)
	if ok {
		// Bound. We are selecting value from its value.
		val, ok := b.Bound.(*ssa.Variable)
		if !ok || !val.Const {
			return nil, false, nil
		}
		return nil, false, ctx.logger.Errorf(ast.Loc,
			"select not implemented yet")
	}

	if len(ast.Name.Package) > 0 {
		var pkg *Package
		pkg, ok = ctx.Packages[ast.Name.Package]
		if !ok {
			return nil, false, ctx.logger.Errorf(ast.Loc,
				"package '%s' not found", ast.Name.Package)
		}
		b, ok = pkg.Bindings.Get(ast.Name.Name)
	} else {
		b, ok = env.Get(ast.Name.Name)
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

// Eval implements the compiler.ast.AST.Eval for constant values.
func (ast *Constant) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	interface{}, bool, error) {
	return ast.Value, true, nil
}
