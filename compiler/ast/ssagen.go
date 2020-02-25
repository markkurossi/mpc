//
// ssagen.go
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
	*ssa.Block, error) {

	var err error

	for _, b := range ast {
		if block.Dead {
			ctx.logger.Warningf(b.Location(), "unreachable code")
			break
		}
		block, err = b.SSA(block, ctx, gen)
		if err != nil {
			return nil, err
		}
	}

	return block, nil
}

func (ast *Func) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, error) {

	// Define arguments.
	for idx, arg := range ast.Args {
		a, err := gen.NewVar(arg.Name, arg.Type, ctx.Scope())
		if err != nil {
			return nil, err
		}
		block.Bindings.Set(a)
		if ctx.Verbose {
			fmt.Printf("args[%d]=%s\n", idx, a)
		}
		ast.Bindings[arg.Name] = a
	}
	// Define return variables.
	for idx, ret := range ast.Return {
		if len(ret.Name) == 0 {
			ret.Name = fmt.Sprintf("%%ret%d", idx)
		}
		r, err := gen.NewVar(ret.Name, ret.Type, ctx.Scope())
		if err != nil {
			return nil, err
		}
		block.Bindings.Set(r)
		if ctx.Verbose {
			fmt.Printf("ret[%d]=%s\n", idx, r)
		}
		ast.Bindings[ret.Name] = r
	}

	block, err := ast.Body.SSA(block, ctx, gen)
	if err != nil {
		return nil, err
	}

	// Select return variables.
	var vars []ssa.Variable
	for _, ret := range ast.Return {
		v, err := ctx.Start.ReturnBinding(ret.Name, ctx.Return, gen)
		if err != nil {
			return nil, err
		}
		vars = append(vars, v)
	}
	ctx.Return.AddInstr(ssa.NewRetInstr(vars))

	return block, nil
}

func (ast *VariableDef) SSA(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (*ssa.Block, error) {

	for _, n := range ast.Names {
		v, err := gen.NewVar(n, ast.Type, ctx.Scope())
		if err != nil {
			return nil, err
		}
		block.Bindings.Set(v)
		if ctx.Verbose {
			fmt.Printf("var %s\n", v)
		}
	}
	return block, nil
}

func (ast *Assign) SSA(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (*ssa.Block, error) {

	b, err := block.Bindings.Get(ast.Name)
	if err != nil {
		return nil, ctx.logger.Errorf(ast.Loc, "%s", err.Error())
	}
	v, err := gen.NewVar(b.Name, b.Type, ctx.Scope())
	if err != nil {
		return nil, err
	}
	ctx.Push(v)
	block, err = ast.Expr.SSA(block, ctx, gen)
	if err != nil {
		return nil, err
	}
	v, err = ctx.Pop()
	if err != nil {
		return nil, err
	}
	block.Bindings.Set(v)

	return block, nil
}

func (ast *If) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, error) {

	ctx.Push(gen.AnonVar(types.BoolType()))
	block, err := ast.Expr.SSA(block, ctx, gen)
	if err != nil {
		return nil, err
	}
	e, err := ctx.Pop()
	if err != nil {
		return nil, err
	}

	branchBlock := gen.NextBlock(block)
	branchBlock.BranchCond = e
	block.AddInstr(ssa.NewJumpInstr(branchBlock))

	block = branchBlock

	// Branch.
	tBlock := gen.BranchBlock(block)
	block.AddInstr(ssa.NewIfInstr(e, tBlock))

	// True branch.
	tNext, err := ast.True.SSA(tBlock, ctx, gen)
	if err != nil {
		return nil, err
	}

	// False (else) branch.
	if len(ast.False) == 0 {
		// No else branch.
		if tNext.Dead {
			// True branch terminated.
			tNext = gen.NextBlock(block)
		} else {
			tNext.Bindings = tNext.Bindings.Merge(e, block.Bindings)
			block.SetNext(tNext)
		}
		block.AddInstr(ssa.NewJumpInstr(tNext))

		return tNext, nil
	}

	fBlock := gen.NextBlock(block)
	block.AddInstr(ssa.NewJumpInstr(fBlock))

	fNext, err := ast.False.SSA(fBlock, ctx, gen)
	if err != nil {
		return nil, err
	}

	if fNext.Dead && tNext.Dead {
		// Both branches terminate.
		next := gen.Block()
		next.Dead = true
		return next, nil
	} else if fNext.Dead {
		// False-branch terminates.
		return tNext, nil
	} else if tNext.Dead {
		// True-branch terminates.
		return fNext, nil
	}

	// Both branches continue.

	next := gen.Block()
	tNext.SetNext(next)
	tNext.AddInstr(ssa.NewJumpInstr(next))

	fNext.SetNext(next)
	fNext.AddInstr(ssa.NewJumpInstr(next))

	next.Bindings = tNext.Bindings.Merge(e, fNext.Bindings)

	return next, nil
}

func (ast *Call) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, error) {

	called, ok := ctx.Functions[ast.Name]
	if !ok {
		return nil, ctx.logger.Errorf(ast.Loc, "function '%s' not defined",
			ast.Name)
	}

	var args []ssa.Variable
	var err error

	for _, expr := range ast.Exprs {
		ctx.Push(gen.UndefVar())
		block, err = expr.SSA(block, ctx, gen)
		if err != nil {
			return nil, err
		}
		v, err := ctx.Pop()
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}
	fmt.Printf("call arguments: %v\n", args)

	t, err := ctx.Peek()
	if err != nil {
		return nil, err
	}
	fmt.Printf("Return variables: %v\n", t)

	// XXX stacked ctx, this is wrong.
	start := gen.Block()
	ctx.Return = gen.Block()

	_, err = called.SSA(start, ctx, gen)

	// XXX pop ctx

	block.AddCall(start)
	block.AddInstr(ssa.NewCallInstr(args, t, start))

	return block, nil
}

func (ast *Return) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, error) {

	if ctx.Func == nil {
		return nil, ctx.logger.Errorf(ast.Loc, "return outside function")
	}
	if len(ast.Exprs) > len(ctx.Func.Return) {
		// TODO \thave (nil, error)
		// TODO \twant (error)
		return nil, ctx.logger.Errorf(ast.Loc, "too many aruments to return")
	} else if len(ast.Exprs) < len(ctx.Func.Return) {
		// TODO \thave ()
		// TODO \twant (error)
		return nil, ctx.logger.Errorf(ast.Loc, "not enough arguments to return")
	}

	for idx, expr := range ast.Exprs {
		r := ctx.Func.Return[idx]
		v, err := gen.NewVar(r.Name, r.Type, ctx.Scope())
		if err != nil {
			return nil, err
		}

		ctx.Push(v)
		block, err = expr.SSA(block, ctx, gen)
		if err != nil {
			return nil, err
		}
		v, err = ctx.Pop()
		if err != nil {
			return nil, err
		}
		block.Bindings.Set(v)
	}

	block.AddInstr(ssa.NewJumpInstr(ctx.Return))
	block.SetNext(ctx.Return)
	block.Dead = true

	return block, nil
}

func (ast *Binary) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, error) {

	ctx.Push(gen.UndefVar())
	block, err := ast.Left.SSA(block, ctx, gen)
	if err != nil {
		return nil, err
	}
	l, err := ctx.Pop()
	if err != nil {
		return nil, err
	}

	ctx.Push(gen.UndefVar())
	block, err = ast.Right.SSA(block, ctx, gen)
	if err != nil {
		return nil, err
	}
	r, err := ctx.Pop()
	if err != nil {
		return nil, err
	}

	// Check that l and r are of same type
	if !l.TypeCompatible(r) {
		return nil, ctx.logger.Errorf(ast.Loc, "invalid types: %s %s %s",
			l.Type, ast.Op, r.Type)
	}

	// Check that target is of correct type
	t, err := ctx.Peek()
	if err != nil {
		return nil, err
	}
	var resultType types.Info
	switch ast.Op {
	case BinaryPlus, BinaryMinus, BinaryMult:
		resultType = l.Type

	case BinaryLt, BinaryLe, BinaryGt, BinaryGe, BinaryEq, BinaryNeq,
		BinaryAnd, BinaryOr:
		resultType = types.BoolType()

	default:
		fmt.Printf("%s %s %s\n", l, ast.Op, r)
		return nil, ctx.logger.Errorf(ast.Loc,
			"Binary.SSA '%s' not implemented yet", ast.Op)
	}
	if t.Type.Undefined() {
		// Target undefined, use expression type.
		ctx.Pop()
		t.Type = resultType
		ctx.Push(t)
	} else if !t.Type.Equal(resultType) {
		return nil, ctx.logger.Errorf(ast.Loc,
			"cannot assign value of type %s to type %s", resultType, t.Type)
	}

	var instr ssa.Instr
	switch ast.Op {
	case BinaryPlus:
		instr, err = ssa.NewAddInstr(l.Type, l, r, t)
	case BinaryMinus:
		instr, err = ssa.NewSubInstr(l.Type, l, r, t)
	case BinaryMult:
		instr, err = ssa.NewMultInstr(l.Type, l, r, t)
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
	default:
		fmt.Printf("%s %s %s\n", l, ast.Op, r)
		return nil, ctx.logger.Errorf(ast.Loc,
			"Binary.SSA '%s' not implemented yet", ast.Op)
	}
	if err != nil {
		return nil, err
	}

	block.AddInstr(instr)

	return block, nil
}

func (ast *VariableRef) SSA(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (*ssa.Block, error) {

	b, err := block.Bindings.Get(ast.Name)
	if err != nil {
		return nil, ctx.logger.Errorf(ast.Loc, "%s", err.Error())
	}

	v := b.Value(block, gen)
	if err != nil {
		return nil, err
	}
	block.Bindings.Set(v)

	t, err := ctx.Peek()
	if err != nil {
		return nil, err
	}

	if t.Type.Undefined() {
		// Replace undefined variable with referenced one.
		ctx.Pop()
		ctx.Push(v)
		return block, nil
	}
	// TODO: check assignment is valid.
	block.AddInstr(ssa.NewMovInstr(v, t))
	return block, nil
}

func (ast *Constant) SSA(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (*ssa.Block, error) {

	v, err := ast.Variable()
	if err != nil {
		return nil, err
	}
	gen.AddConstant(v)

	t, err := ctx.Peek()
	if err != nil {
		return nil, err
	}
	if t.Type.Undefined() {
		// Replace undefined variable with constant.
		ctx.Pop()
		ctx.Push(v)
		return block, nil
	}
	// Check that the assignment is valid.
	if v.Type.Bits > t.Type.Bits {
		return nil, ctx.logger.Errorf(ast.Loc,
			"constant overflow when assigning %s to %s", v.Type, t.Type)
	}

	// XXX signed vs. unsigned
	block.AddInstr(ssa.NewMovInstr(v, t))
	return block, nil
}
