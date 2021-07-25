//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"
	"math"

	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/types"
)

// Eval implements the compiler.ast.AST.Eval for list statements.
func (ast List) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {
	return ssa.Undefined, false, fmt.Errorf("List.Eval not implemented yet")
}

// Eval implements the compiler.ast.AST.Eval for function definitions.
func (ast *Func) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {
	return ssa.Undefined, false, nil
}

// Eval implements the compiler.ast.AST.Eval for constant definitions.
func (ast *ConstantDef) Eval(env *Env, ctx *Codegen,
	gen *ssa.Generator) (ssa.Value, bool, error) {
	return ssa.Undefined, false, nil
}

// Eval implements the compiler.ast.AST.Eval for variable definitions.
func (ast *VariableDef) Eval(env *Env, ctx *Codegen,
	gen *ssa.Generator) (ssa.Value, bool, error) {
	return ssa.Undefined, false, nil
}

// Eval implements the compiler.ast.AST.Eval for assignment expressions.
func (ast *Assign) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {

	var values []interface{}
	for _, expr := range ast.Exprs {
		val, ok, err := expr.Eval(env, ctx, gen)
		if err != nil || !ok {
			return ssa.Undefined, ok, err
		}
		// XXX multiple return values.
		values = append(values, val)
	}

	if len(ast.LValues) != len(values) {
		return ssa.Undefined, false, ctx.Errorf(ast,
			"assignment mismatch: %d variables but %d values",
			len(ast.LValues), len(values))
	}

	arrType := types.Info{
		Type: types.TArray,
	}

	for idx, lv := range ast.LValues {
		constVal, _, err := gen.Constant(values[idx], types.Undefined)
		if err != nil {
			return ssa.Undefined, false, err
		}
		gen.AddConstant(constVal)
		arrType.ElementType = &constVal.Type

		ref, ok := lv.(*VariableRef)
		if !ok {
			return ssa.Undefined, false,
				ctx.Errorf(ast, "cannot assign to %s", lv)
		}
		// XXX package.name below

		var lValue ssa.Value
		if ast.Define {
			lValue, err = gen.NewVal(ref.Name.Name, constVal.Type, ctx.Scope())
			if err != nil {
				return ssa.Undefined, false, err
			}
		} else {
			b, ok := env.Get(ref.Name.Name)
			if !ok {
				return ssa.Undefined, false,
					ctx.Errorf(ast, "undefined variable '%s'", ref.Name)
			}
			lValue, err = gen.NewVal(b.Name, b.Type, ctx.Scope())
			if err != nil {
				return ssa.Undefined, false, err
			}
		}
		env.Set(lValue, &constVal)
	}
	arrType.ArraySize = len(values)

	return gen.Constant(values, arrType)
}

// Eval implements the compiler.ast.AST.Eval for if statements.
func (ast *If) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {
	return ssa.Undefined, false, nil
}

// Eval implements the compiler.ast.AST.Eval for call expressions.
func (ast *Call) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {

	// Resolve called.
	var pkgName string
	if len(ast.Ref.Name.Package) > 0 {
		pkgName = ast.Ref.Name.Package
	} else {
		pkgName = ast.Ref.Name.Defined
	}
	pkg, ok := ctx.Packages[pkgName]
	if !ok {
		return ssa.Undefined, false,
			ctx.Errorf(ast, "package '%s' not found", pkgName)
	}
	_, ok = pkg.Functions[ast.Ref.Name.Name]
	if ok {
		return ssa.Undefined, false, nil
	}
	// Check builtin functions.
	for _, bi := range builtins {
		if bi.Name != ast.Ref.Name.Name {
			continue
		}
		if bi.Type != BuiltinFunc {
			return ssa.Undefined, false,
				fmt.Errorf("builtin %s used as function", bi.Name)
		}
		if bi.Eval == nil {
			return ssa.Undefined, false, nil
		}
		return bi.Eval(ast.Exprs, env, ctx, gen, ast.Location())
	}

	return ssa.Undefined, false, nil
}

// Eval implements the compiler.ast.AST.Eval for return statements.
func (ast *Return) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {
	return ssa.Undefined, false, nil
}

// Eval implements the compiler.ast.AST.Eval for for statements.
func (ast *For) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {
	return ssa.Undefined, false, nil
}

// Eval implements the compiler.ast.AST.Eval for binary expressions.
func (ast *Binary) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {
	l, ok, err := ast.Left.Eval(env, ctx, gen)
	if err != nil || !ok {
		return ssa.Undefined, ok, err
	}
	r, ok, err := ast.Right.Eval(env, ctx, gen)
	if err != nil || !ok {
		return ssa.Undefined, ok, err
	}

	switch lval := l.ConstValue.(type) {
	case int32:
		var rval int32
		switch rv := r.ConstValue.(type) {
		case int32:
			rval = rv
		default:
			return ssa.Undefined, false, ctx.Errorf(ast.Right,
				"invalid r-value %T %s %T", lval, ast.Op, rv)
		}
		switch ast.Op {
		case BinaryMult:
			return gen.Constant(lval*rval, types.Int32)
		case BinaryDiv:
			if rval == 0 {
				return ssa.Undefined, false, ctx.Errorf(ast.Right,
					"integer divide by zero")
			}
			return gen.Constant(lval/rval, types.Int32)
		case BinaryMod:
			if rval == 0 {
				return ssa.Undefined, false, ctx.Errorf(ast.Right,
					"integer divide by zero")
			}
			return gen.Constant(lval%rval, types.Int32)
		case BinaryLshift:
			return gen.Constant(lval<<rval, types.Int32)
		case BinaryRshift:
			return gen.Constant(lval>>rval, types.Int32)

		case BinaryPlus:
			return gen.Constant(lval+rval, types.Int32)
		case BinaryMinus:
			return gen.Constant(lval-rval, types.Int32)

		case BinaryEq:
			return gen.Constant(lval == rval, types.Bool)
		case BinaryNeq:
			return gen.Constant(lval != rval, types.Bool)
		case BinaryLt:
			return gen.Constant(lval < rval, types.Bool)
		case BinaryLe:
			return gen.Constant(lval <= rval, types.Bool)
		case BinaryGt:
			return gen.Constant(lval > rval, types.Bool)
		case BinaryGe:
			return gen.Constant(lval >= rval, types.Bool)
		default:
			return ssa.Undefined, false, ctx.Errorf(ast.Right,
				"Binary.Eval: '%T %s %T' not implemented yet", l, ast.Op, r)
		}

	case uint64:
		var rval uint64
		switch rv := r.ConstValue.(type) {
		case uint64:
			rval = rv
		default:
			return ssa.Undefined, false, ctx.Errorf(ast.Right,
				"%T: invalid r-value %v (%T)", lval, rv, rv)
		}
		switch ast.Op {
		case BinaryMult:
			return gen.Constant(lval*rval, types.Uint64)
		case BinaryDiv:
			if rval == 0 {
				return ssa.Undefined, false, ctx.Errorf(ast.Right,
					"integer divide by zero")
			}
			return gen.Constant(lval/rval, types.Uint64)
		case BinaryMod:
			if rval == 0 {
				return ssa.Undefined, false, ctx.Errorf(ast.Right,
					"integer divide by zero")
			}
			return gen.Constant(lval%rval, types.Uint64)
		case BinaryLshift:
			return gen.Constant(lval<<rval, types.Uint64)
		case BinaryRshift:
			return gen.Constant(lval>>rval, types.Uint64)

		case BinaryPlus:
			return gen.Constant(lval+rval, types.Uint64)
		case BinaryMinus:
			return gen.Constant(lval-rval, types.Uint64)

		case BinaryEq:
			return gen.Constant(lval == rval, types.Bool)
		case BinaryNeq:
			return gen.Constant(lval != rval, types.Bool)
		case BinaryLt:
			return gen.Constant(lval < rval, types.Bool)
		case BinaryLe:
			return gen.Constant(lval <= rval, types.Bool)
		case BinaryGt:
			return gen.Constant(lval > rval, types.Bool)
		case BinaryGe:
			return gen.Constant(lval >= rval, types.Bool)
		default:
			return ssa.Undefined, false, ctx.Errorf(ast.Right,
				"Binary.Eval: '%T %s %T' not implemented yet", l, ast.Op, r)
		}

	default:
		return ssa.Undefined, false, ctx.Errorf(ast.Left,
			"invalid l-value %v (%T)", lval, lval)
	}
}

// Eval implements the compiler.ast.AST.Eval for unary expressions.
func (ast *Unary) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {
	expr, ok, err := ast.Expr.Eval(env, ctx, gen)
	if err != nil || !ok {
		return ssa.Undefined, ok, err
	}
	switch val := expr.ConstValue.(type) {
	case int32:
		switch ast.Type {
		case UnaryMinus:
			return gen.Constant(-val, types.Int32)
		default:
			return ssa.Undefined, false, ctx.Errorf(ast.Expr,
				"Unary.Eval: '%s%T' not implemented yet", ast.Type, val)
		}
	default:
		return ssa.Undefined, false, ctx.Errorf(ast.Expr,
			"invalid value %s%T", ast.Type, val)
	}
}

// Eval implements the compiler.ast.AST.Eval for slice expressions.
func (ast *Slice) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {

	expr, ok, err := ast.Expr.Eval(env, ctx, gen)
	if err != nil || !ok {
		return ssa.Undefined, ok, err
	}

	from := 0
	to := math.MaxInt32

	if ast.From != nil {
		val, ok, err := ast.From.Eval(env, ctx, gen)
		if err != nil || !ok {
			return ssa.Undefined, ok, err
		}
		from, err = intVal(val)
		if err != nil {
			return ssa.Undefined, false, ctx.Errorf(ast.From, err.Error())
		}
	}
	if ast.To != nil {
		val, ok, err := ast.To.Eval(env, ctx, gen)
		if err != nil || !ok {
			return ssa.Undefined, ok, err
		}
		to, err = intVal(val)
		if err != nil {
			return ssa.Undefined, false, ctx.Errorf(ast.To, err.Error())
		}
	}
	if to <= from {
		return ssa.Undefined, false, ctx.Errorf(ast.Expr,
			"invalid slice range %d:%d", from, to)
	}
	switch val := expr.ConstValue.(type) {
	case int32:
		if from >= 32 {
			return ssa.Undefined, false, ctx.Errorf(ast.From,
				"slice bounds out of range [%d:32]", from)
		}
		tmp := uint32(val)
		tmp >>= from
		tmp &^= 0xffffffff << (to - from)
		return gen.Constant(int32(tmp), types.Int32)

	default:
		return ssa.Undefined, false, ctx.Errorf(ast.Expr,
			"Slice.Eval: expr %T not implemented yet", val)
	}
}

func intVal(val interface{}) (int, error) {
	switch v := val.(type) {
	case int32:
		return int(v), nil

	case ssa.Value:
		if !v.Const {
			return 0, fmt.Errorf("non-const slice index: %v", v)
		}
		return intVal(v.ConstValue)

	default:
		return 0, fmt.Errorf("invalid slice index: %T", v)
	}
}

// Eval implements the compiler.ast.AST.Eval() for index expressions.
func (ast *Index) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {

	expr, ok, err := ast.Expr.Eval(env, ctx, gen)
	if err != nil || !ok {
		return ssa.Undefined, ok, err
	}

	val, ok, err := ast.Index.Eval(env, ctx, gen)
	if err != nil || !ok {
		return ssa.Undefined, ok, err
	}
	index, err := intVal(val)
	if err != nil {
		return ssa.Undefined, false, ctx.Errorf(ast.Index, err.Error())
	}

	switch val := expr.ConstValue.(type) {
	case string:
		bytes := []byte(val)
		if index < 0 || index >= len(bytes) {
			return ssa.Undefined, false, ctx.Errorf(ast.Index,
				"invalid array index %d (out of bounds for %d-element string)",
				index, len(bytes))
		}
		return gen.Constant(bytes[index], types.Int32)

	default:
		return ssa.Undefined, false, ctx.Errorf(ast.Expr,
			"Index.Eval: expr %T not implemented yet", val)
	}
}

// Eval implements the compiler.ast.AST.Eval for variable references.
func (ast *VariableRef) Eval(env *Env, ctx *Codegen,
	gen *ssa.Generator) (ssa.Value, bool, error) {

	var b ssa.Binding
	var ok bool

	// Check if package name is bound to variable.
	b, ok = env.Get(ast.Name.Package)
	if ok {
		// Bound. We are selecting value from its value.
		val, ok := b.Bound.(*ssa.Value)
		if !ok || !val.Const {
			return ssa.Undefined, false, nil
		}
		return ssa.Undefined, false, ctx.Errorf(ast,
			"VariableRef.Eval: select not implemented yet")
	}

	if len(ast.Name.Package) > 0 {
		var pkg *Package
		pkg, ok = ctx.Packages[ast.Name.Package]
		if !ok {
			return ssa.Undefined, false, ctx.Errorf(ast,
				"package '%s' not found", ast.Name.Package)
		}
		b, ok = pkg.Bindings.Get(ast.Name.Name)
	} else {
		// First check env bindings.
		b, ok = env.Get(ast.Name.Name)
		if !ok {
			// Check names in the current package.
			b, ok = ctx.Package.Bindings.Get(ast.Name.Name)
		}
	}
	if !ok {
		return ssa.Undefined, false, ctx.Errorf(ast, "undefined variable '%s'",
			ast.Name.String())
	}

	val, ok := b.Bound.(*ssa.Value)
	if !ok || !val.Const {
		return ssa.Undefined, false, nil
	}

	return *val, true, nil
}

// Eval implements the compiler.ast.AST.Eval for constant values.
func (ast *BasicLit) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {
	return gen.Constant(ast.Value, types.Undefined)
}

// Eval implements the compiler.ast.AST.Eval for constant values.
func (ast *CompositeLit) Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
	ssa.Value, bool, error) {

	typeInfo, err := ast.Type.Resolve(env, ctx, gen)
	if err != nil {
		return ssa.Undefined, false, err
	}
	switch typeInfo.Type {
	case types.TStruct:
		// Check if all elements are constants.
		var values []interface{}
		for _, el := range ast.Value {
			// XXX check if el.Key is specified

			v, ok, err := el.Element.Eval(env, ctx, gen)
			if err != nil || !ok {
				return ssa.Undefined, ok, err
			}
			// XXX chck that v is assignment compatible with typeInfo.Struct[i]
			values = append(values, v)
		}
		return gen.Constant(values, typeInfo)

	case types.TArray:
		// Check if all elements are constants.
		var values []interface{}
		for _, el := range ast.Value {
			// XXX check if el.Key is specified

			v, ok, err := el.Element.Eval(env, ctx, gen)
			if err != nil || !ok {
				return ssa.Undefined, ok, err
			}
			// XXX check that v is assignment compatible with array.
			values = append(values, v)
		}
		return gen.Constant(values, typeInfo)

	default:
		fmt.Printf("CompositeLit.Eval: not implemented yet: %v, Value: %v\n",
			typeInfo, ast.Value)
		return ssa.Undefined, false, nil
	}
}
