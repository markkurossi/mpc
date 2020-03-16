//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"

	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/types"
)

func (ast List) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, []ssa.Variable, error) {

	var err error

	for _, b := range ast {
		if block.Dead {
			ctx.logger.Warningf(b.Location(), "unreachable code")
			break
		}
		block, _, err = b.SSA(block, ctx, gen)
		if err != nil {
			return nil, nil, err
		}
	}

	return block, nil, nil
}

func (ast *Func) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, []ssa.Variable, error) {

	ctx.Start().Name = fmt.Sprintf("%s#%d", ast.Name, ast.NumInstances)
	ctx.Return().Name = fmt.Sprintf("%s.ret#%d", ast.Name, ast.NumInstances)
	ast.NumInstances++

	// Define arguments.
	for _, arg := range ast.Args {
		a, err := gen.NewVar(arg.Name, arg.Type, ctx.Scope())
		if err != nil {
			return nil, nil, err
		}
		block.Bindings.Set(a, nil)
		ast.Bindings[arg.Name] = a
	}
	// Define return variables.
	for idx, ret := range ast.Return {
		if len(ret.Name) == 0 {
			ret.Name = fmt.Sprintf("%%ret%d", idx)
		}
		r, err := gen.NewVar(ret.Name, ret.Type, ctx.Scope())
		if err != nil {
			return nil, nil, err
		}
		block.Bindings.Set(r, nil)
		ast.Bindings[ret.Name] = r
	}

	block, _, err := ast.Body.SSA(block, ctx, gen)
	if err != nil {
		return nil, nil, err
	}

	// Select return variables.
	var vars []ssa.Variable
	for _, ret := range ast.Return {
		v, ok := ctx.Start().ReturnBinding(ret.Name, ctx.Return(), gen)
		if !ok {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"undefined variable '%s'", ret.Name)
		}
		vars = append(vars, v)
	}

	caller := ctx.Caller()
	if caller != nil {
		ctx.Return().AddInstr(ssa.NewJumpInstr(caller))
	} else {
		for idx, ret := range ast.Return {
			ast.Bindings[ret.Name] = vars[idx]
		}
		ctx.Return().AddInstr(ssa.NewRetInstr(vars))
	}

	return block, vars, nil
}

func (ast *VariableDef) SSA(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (*ssa.Block, []ssa.Variable, error) {

	for _, n := range ast.Names {
		lValue, err := gen.NewVar(n, ast.Type, ctx.Scope())
		if err != nil {
			return nil, nil, err
		}
		block.Bindings.Set(lValue, nil)

		// XXX is init is nil, we must init the variable with type's zero.
		if ast.Init != nil {
			var v []ssa.Variable
			block, v, err = ast.Init.SSA(block, ctx, gen)
			if err != nil {
				return nil, nil, err
			}
			if len(v) != 1 {
				return nil, nil, ctx.logger.Errorf(ast.Loc,
					"multiple-value %s used in single-value context", ast.Init)
			}
			block.AddInstr(ssa.NewMovInstr(v[0], lValue))
		}

	}
	return block, nil, nil
}

func (ast *Assign) SSA(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (*ssa.Block, []ssa.Variable, error) {

	block, v, err := ast.Expr.SSA(block, ctx, gen)
	if err != nil {
		return nil, nil, err
	}
	// XXX 0, 1, n return values.
	if len(v) != 1 {
		return nil, nil, ctx.logger.Errorf(ast.Loc,
			"assignment mismatch: %d variables but %d value", 1, len(v))
	}

	var lValue ssa.Variable
	if ast.Define {
		lValue, err = gen.NewVar(ast.Name, v[0].Type, ctx.Scope())
		if err != nil {
			return nil, nil, err
		}
	} else {
		// XXX check ast.Define
		b, ok := block.Bindings.Get(ast.Name)
		if !ok {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"undefined varialbe '%s'", ast.Name)
		}
		lValue, err = gen.NewVar(b.Name, b.Type, ctx.Scope())
		if err != nil {
			return nil, nil, err
		}
	}

	block.AddInstr(ssa.NewMovInstr(v[0], lValue))
	block.Bindings.Set(lValue, nil)

	return block, v, nil
}

func (ast *If) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, []ssa.Variable, error) {

	block, e, err := ast.Expr.SSA(block, ctx, gen)
	if err != nil {
		return nil, nil, err
	}
	if len(e) == 0 {
		return nil, nil, ctx.logger.Errorf(ast.Expr.Location(),
			"%s used as value", ast.Expr)
	} else if len(e) > 1 {
		return nil, nil, ctx.logger.Errorf(ast.Expr.Location(),
			"multiple-value %s used in single-value context", ast.Expr)
	}
	if e[0].Type.Type != types.Bool {
		return nil, nil, ctx.logger.Errorf(ast.Expr.Location(),
			"non-bool %s (type %s) used as if condition", ast.Expr, e[0].Type)
	}

	branchBlock := gen.NextBlock(block)
	branchBlock.BranchCond = e[0]
	block.AddInstr(ssa.NewJumpInstr(branchBlock))

	block = branchBlock

	// Branch.
	tBlock := gen.BranchBlock(block)
	block.AddInstr(ssa.NewIfInstr(e[0], tBlock))

	// True branch.
	tNext, _, err := ast.True.SSA(tBlock, ctx, gen)
	if err != nil {
		return nil, nil, err
	}

	// False (else) branch.
	if len(ast.False) == 0 {
		// No else branch.
		if tNext.Dead {
			// True branch terminated.
			tNext = gen.NextBlock(block)
		} else {
			tNext.Bindings = tNext.Bindings.Merge(e[0], block.Bindings)
			block.SetNext(tNext)
		}
		block.AddInstr(ssa.NewJumpInstr(tNext))

		return tNext, nil, nil
	}

	fBlock := gen.NextBlock(block)
	block.AddInstr(ssa.NewJumpInstr(fBlock))

	fNext, _, err := ast.False.SSA(fBlock, ctx, gen)
	if err != nil {
		return nil, nil, err
	}

	if fNext.Dead && tNext.Dead {
		// Both branches terminate.
		next := gen.Block()
		next.Dead = true
		return next, nil, nil
	} else if fNext.Dead {
		// False-branch terminates.
		return tNext, nil, nil
	} else if tNext.Dead {
		// True-branch terminates.
		return fNext, nil, nil
	}

	// Both branches continue.

	next := gen.Block()
	tNext.SetNext(next)
	tNext.AddInstr(ssa.NewJumpInstr(next))

	fNext.SetNext(next)
	fNext.AddInstr(ssa.NewJumpInstr(next))

	next.Bindings = tNext.Bindings.Merge(e[0], fNext.Bindings)

	return next, nil, nil
}

func (ast *Call) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, []ssa.Variable, error) {

	// Generate call values.

	var callValues [][]ssa.Variable
	var v []ssa.Variable
	var err error

	for _, expr := range ast.Exprs {
		block, v, err = expr.SSA(block, ctx, gen)
		if err != nil {
			return nil, nil, err
		}
		callValues = append(callValues, v)
	}

	// Resolve called.
	pkg, ok := ctx.Packages[ast.Name.Package]
	if !ok {
		return nil, nil, ctx.logger.Errorf(ast.Loc, "package '%s' not found",
			ast.Name.Package)
	}
	called, ok := pkg.Functions[ast.Name.Name]
	if !ok {
		// Check builtin functions.
		for _, bi := range builtins {
			if bi.Name != ast.Name.Name {
				continue
			}
			if bi.Type != BuiltinFunc {
				return nil, nil, ctx.logger.Errorf(ast.Loc,
					"builtin %s used as function", bi.Name)
			}
			// Flatten arguments.
			var args []ssa.Variable
			for _, arg := range callValues {
				args = append(args, arg...)
			}
			return bi.SSA(block, ctx, gen, args, ast.Loc)
		}
		return nil, nil, ctx.logger.Errorf(ast.Loc, "function '%s' not defined",
			ast.Name)
	}

	var args []ssa.Variable

	if len(callValues) == 0 {
		if len(called.Args) != 0 {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"not enough arguments in call to %s", ast.Name)
			// TODO \thave ()
			// TODO \twant (int, int)
		}
	} else if len(callValues) == 1 {
		if len(callValues[0]) < len(called.Args) {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"not enough arguments in call to %s", ast.Name)
			// TODO \thave ()
			// TODO \twant (int, int)
		} else if len(callValues[0]) > len(called.Args) {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"too many arguments in call to %s", ast.Name)
			// TODO \thave (int, int)
			// TODO \twant ()
		}
		args = callValues[0]
	} else {
		if len(callValues) < len(called.Args) {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"not enough arguments in call to %s", ast.Name)
			// TODO \thave ()
			// TODO \twant (int, int)
		} else if len(callValues) > len(called.Args) {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"too many arguments in call to %s", ast.Name)
			// TODO \thave (int, int)
			// TODO \twant ()
		} else {
			for idx, ca := range callValues {
				expr := ast.Exprs[idx]
				if len(ca) == 0 {
					return nil, nil,
						ctx.logger.Errorf(expr.Location(),
							"%s used as value", expr)
				} else if len(ca) > 1 {
					return nil, nil,
						ctx.logger.Errorf(expr.Location(),
							"multiple-value %s in single-value context", expr)
				}
				args = append(args, ca[0])
			}
		}
	}

	// Return block.
	rblock := gen.Block()
	rblock.Bindings = block.Bindings.Clone()

	ctx.PushCompilation(gen.Block(), gen.Block(), rblock, called)
	_, returnValues, err := called.SSA(ctx.Start(), ctx, gen)
	if err != nil {
		return nil, nil, err
	}

	for idx, arg := range called.Args {
		b, ok := ctx.Start().Bindings.Get(arg.Name)
		if !ok {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"undefined variable '%s'", arg.Name)
		}
		bv := b.Value(block, gen)
		block.AddInstr(ssa.NewMovInstr(args[idx], bv))
	}

	block.SetNext(ctx.Start())
	block.AddInstr(ssa.NewJumpInstr(ctx.Start()))

	ctx.Return().SetNext(rblock)
	block = rblock

	ctx.PopCompilation()

	return block, returnValues, nil
}

func (ast *Return) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, []ssa.Variable, error) {

	if ctx.Func() == nil {
		return nil, nil, ctx.logger.Errorf(ast.Loc, "return outside function")
	}

	var rValues [][]ssa.Variable
	var result []ssa.Variable
	var v []ssa.Variable
	var err error

	for _, expr := range ast.Exprs {
		block, v, err = expr.SSA(block, ctx, gen)
		if err != nil {
			return nil, nil, err
		}
		rValues = append(rValues, v)
	}
	if len(rValues) == 0 {
		if len(ctx.Func().Return) != 0 {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"not enough arguments to return")
			// TODO \thave ()
			// TODO \twant (error)
		}
	} else if len(rValues) == 1 {
		if len(rValues[0]) < len(ctx.Func().Return) {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"not enough arguments to return")
			// TODO \thave ()
			// TODO \twant (error)
		} else if len(rValues[0]) > len(ctx.Func().Return) {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"too many aruments to return")
			// TODO \thave (nil, error)
			// TODO \twant (error)
		}
		result = rValues[0]
	} else {
		if len(rValues) < len(ctx.Func().Return) {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"not enough arguments to return")
			// TODO \thave ()
			// TODO \twant (error)
		} else if len(rValues) > len(ctx.Func().Return) {
			return nil, nil, ctx.logger.Errorf(ast.Loc,
				"too many aruments to return")
			// TODO \thave (nil, error)
			// TODO \twant (error)
		} else {
			for idx, rv := range rValues {
				expr := ast.Exprs[idx]
				if len(rv) == 0 {
					return nil, nil,
						ctx.logger.Errorf(expr.Location(),
							"%s used as value", expr)
				} else if len(rv) > 1 {
					return nil, nil,
						ctx.logger.Errorf(expr.Location(),
							"multiple-value %s in single-value context", expr)
				}
				result = append(result, rv[0])
			}
		}
	}

	for idx, r := range ctx.Func().Return {
		v, err := gen.NewVar(r.Name, r.Type, ctx.Scope())
		if err != nil {
			return nil, nil, err
		}
		block.AddInstr(ssa.NewMovInstr(result[idx], v))
		block.Bindings.Set(v, nil)
	}

	block.AddInstr(ssa.NewJumpInstr(ctx.Return()))
	block.SetNext(ctx.Return())
	block.Dead = true

	return block, nil, nil
}

func (ast *For) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, []ssa.Variable, error) {

	// Init loop.
	_, ok, err := ast.Init.Eval(block, ctx, gen)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, ctx.logger.Errorf(ast.Init.Location(),
			"init statement is not compile-time constant: %s", err)
	}

	// Expand body as long as condition is true.
	for {
		constVal, ok, err := ast.Cond.Eval(block, ctx, gen)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, ctx.logger.Errorf(ast.Cond.Location(),
				"condition is not compile-time constant: %s", err)
		}
		val, ok := constVal.(bool)
		if !ok {
			return nil, nil, ctx.logger.Errorf(ast.Cond.Location(),
				"condition is not a boolean expression")
		}
		if !val {
			// Loop completed.
			break
		}

		// Expand block.
		block, _, err = ast.Body.SSA(block, ctx, gen)
		if err != nil {
			return nil, nil, err
		}

		// Increment.
		_, ok, err = ast.Inc.Eval(block, ctx, gen)
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			return nil, nil, ctx.logger.Errorf(ast.Init.Location(),
				"increment statement is not compile-time constant: %s", err)
		}
	}

	return block, nil, nil
}

func (ast *Binary) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, []ssa.Variable, error) {

	// Check constant folding.
	constVal, ok, err := ast.Eval(block, ctx, gen)
	if err != nil {
		return nil, nil, err
	}
	if ok {
		if ctx.Verbose {
			fmt.Printf("ConstFold: %v %s %v => %v\n",
				ast.Left, ast.Op, ast.Right, constVal)
		}
		v, err := ssa.Constant(constVal)
		if err != nil {
			return nil, nil, err
		}
		gen.AddConstant(v)
		return block, []ssa.Variable{v}, nil
	}

	block, lArr, err := ast.Left.SSA(block, ctx, gen)
	if err != nil {
		return nil, nil, err
	}

	block, rArr, err := ast.Right.SSA(block, ctx, gen)
	if err != nil {
		return nil, nil, err
	}

	// Check that l and r are of same type.
	if len(lArr) == 0 {
		return nil, nil, ctx.logger.Errorf(ast.Left.Location(),
			"%s used as value", ast.Left)
	}
	if len(lArr) > 1 {
		return nil, nil, ctx.logger.Errorf(ast.Left.Location(),
			"multiple-value %s in single-value context", ast.Left)
	}
	if len(rArr) == 0 {
		return nil, nil, ctx.logger.Errorf(ast.Right.Location(),
			"%s used as value", ast.Right)
	}
	if len(rArr) > 1 {
		return nil, nil, ctx.logger.Errorf(ast.Right.Location(),
			"multiple-value %s in single-value context", ast.Right)
	}
	l := lArr[0]
	r := rArr[0]

	if !l.TypeCompatible(r) {
		return nil, nil,
			ctx.logger.Errorf(ast.Loc, "invalid types: %s %s %s",
				l.Type, ast.Op, r.Type)
	}

	// Resolve target type.
	var resultType types.Info
	switch ast.Op {
	case BinaryMult, BinaryDiv, BinaryMod, BinaryLshift, BinaryRshift,
		BinaryBand, BinaryBclear,
		BinaryPlus, BinaryMinus, BinaryBor, BinaryBxor:
		resultType = l.Type

	case BinaryLt, BinaryLe, BinaryGt, BinaryGe, BinaryEq, BinaryNeq,
		BinaryAnd, BinaryOr:
		resultType = types.BoolType()

	default:
		fmt.Printf("%s %s %s\n", l, ast.Op, r)
		return nil, nil, ctx.logger.Errorf(ast.Loc,
			"Binary.SSA '%s' not implemented yet", ast.Op)
	}
	t := gen.AnonVar(resultType)

	var instr ssa.Instr
	switch ast.Op {
	case BinaryMult:
		instr, err = ssa.NewMultInstr(l.Type, l, r, t)
	case BinaryDiv:
		instr, err = ssa.NewDivInstr(l.Type, l, r, t)
	case BinaryMod:
		instr, err = ssa.NewModInstr(l.Type, l, r, t)
	case BinaryLshift:
		instr, err = ssa.NewLshiftInstr(l, r, t)
	case BinaryRshift:
		instr, err = ssa.NewRshiftInstr(l, r, t)
	case BinaryBand:
		instr, err = ssa.NewBandInstr(l, r, t)
	case BinaryBclear:
		instr, err = ssa.NewBclrInstr(l, r, t)
	case BinaryPlus:
		instr, err = ssa.NewAddInstr(l.Type, l, r, t)
	case BinaryMinus:
		instr, err = ssa.NewSubInstr(l.Type, l, r, t)
	case BinaryBor:
		instr, err = ssa.NewBorInstr(l, r, t)
	case BinaryBxor:
		instr, err = ssa.NewBxorInstr(l, r, t)
	case BinaryEq:
		instr, err = ssa.NewEqInstr(l, r, t)
	case BinaryNeq:
		instr, err = ssa.NewNeqInstr(l, r, t)
	case BinaryLt:
		instr, err = ssa.NewLtInstr(l.Type, l, r, t)
	case BinaryLe:
		instr, err = ssa.NewLeInstr(l.Type, l, r, t)
	case BinaryGt:
		instr, err = ssa.NewGtInstr(l.Type, l, r, t)
	case BinaryGe:
		instr, err = ssa.NewGeInstr(l.Type, l, r, t)
	case BinaryAnd:
		instr, err = ssa.NewAndInstr(l, r, t)
	case BinaryOr:
		instr, err = ssa.NewOrInstr(l, r, t)
	default:
		fmt.Printf("%s %s %s\n", l, ast.Op, r)
		return nil, nil, ctx.logger.Errorf(ast.Loc,
			"Binary.SSA '%s' not implemented yet", ast.Op)
	}
	if err != nil {
		return nil, nil, err
	}

	block.AddInstr(instr)

	return block, []ssa.Variable{t}, nil
}

func (ast *VariableRef) SSA(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (*ssa.Block, []ssa.Variable, error) {

	var b ssa.Binding
	var ok bool

	if len(ast.Name.Package) > 0 {
		var pkg *Package
		pkg, ok = ctx.Packages[ast.Name.Package]
		if !ok {
			return nil, nil,
				ctx.logger.Errorf(ast.Loc, "package '%s' not found",
					ast.Name.Package)
		}
		b, ok = pkg.Bindings.Get(ast.Name.Name)
	} else {
		b, ok = block.Bindings.Get(ast.Name.Name)
	}
	if !ok {
		return nil, nil, ctx.logger.Errorf(ast.Loc, "undefined variable '%s'",
			ast.Name.String())
	}

	value := b.Value(block, gen)

	// Bind variable with the name it was referenced.
	lValue := value
	lValue.Name = ast.Name.String()
	block.Bindings.Set(lValue, &value)

	if value.Const {
		gen.AddConstant(value)
	}

	return block, []ssa.Variable{value}, nil
}

func (ast *Constant) SSA(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (*ssa.Block, []ssa.Variable, error) {

	v, err := ssa.Constant(ast.Value)
	if err != nil {
		return nil, nil, err
	}
	gen.AddConstant(v)

	return block, []ssa.Variable{v}, nil
}

func (ast *Conversion) SSA(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (*ssa.Block, []ssa.Variable, error) {

	block, val, err := ast.Expr.SSA(block, ctx, gen)
	if err != nil {
		return nil, nil, err
	}
	if len(val) == 0 {
		return nil, nil, ctx.logger.Errorf(ast.Expr.Location(),
			"%s used as value", ast.Expr)
	}
	if len(val) > 1 {
		return nil, nil, ctx.logger.Errorf(ast.Expr.Location(),
			"multiple-value %s in single-value context", ast.Expr)
	}

	t := gen.AnonVar(ast.Type)

	block.AddInstr(ssa.NewMovInstr(val[0], t))

	return block, []ssa.Variable{t}, nil
}
