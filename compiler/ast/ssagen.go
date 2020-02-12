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
		if block == nil {
			fmt.Printf("%s: unreachable code\n", b.Location())
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

	ctx.Func = ast
	defer func() {
		ctx.Func = nil
	}()

	// Define arguments.
	for idx, arg := range ast.Args {
		a, err := gen.Var(arg.Name, arg.Type, ctx.Scope())
		if err != nil {
			return nil, err
		}
		if ctx.Verbose {
			fmt.Printf("args[%d]=%s\n", idx, a)
		}
	}
	// Define return variables.
	for idx, ret := range ast.Return {
		if len(ret.Name) == 0 {
			ret.Name = fmt.Sprintf("$ret%d", idx)
		}
		r, err := gen.Var(ret.Name, ret.Type, ctx.Scope())
		if err != nil {
			return nil, err
		}
		if ctx.Verbose {
			fmt.Printf("ret[%d]=%s\n", idx, r)
		}
	}

	return ast.Body.SSA(block, ctx, gen)
}

func (ast *VariableDef) SSA(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (*ssa.Block, error) {

	for _, n := range ast.Names {
		v, err := gen.Var(n, ast.Type, ctx.Scope())
		if err != nil {
			return nil, err
		}
		if ctx.Verbose {
			fmt.Printf("var %s\n", v)
		}
	}
	return block, nil
}

func (ast *Assign) SSA(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (*ssa.Block, error) {

	v, err := gen.Lookup(ast.Name, ctx.Scope(), true)
	if err != nil {
		return nil, err
	}
	ctx.Push(v)
	block, err = ast.Expr.SSA(block, ctx, gen)
	if err != nil {
		return nil, err
	}
	_, err = ctx.Pop()
	if err != nil {
		return nil, err
	}
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

	branchBlock := gen.Block()
	block.AddInstr(ssa.NewJumpInstr(branchBlock))
	block.AddTo(branchBlock)

	block = branchBlock

	// Branch.
	tBlock := gen.Block()
	block.AddInstr(ssa.NewIfInstr(e, tBlock))
	block.AddTo(tBlock)

	// True branch.
	tNext, err := ast.True.SSA(tBlock, ctx, gen)
	if err != nil {
		return nil, err
	}

	// False (else) branch.
	if len(ast.False) == 0 {
		// No else branch.
		if tNext == nil {
			// True branch terminated.
			tNext = gen.Block()
		}
		block.AddInstr(ssa.NewJumpInstr(tNext))
		block.AddTo(tNext)

		return tNext, nil
	}

	fBlock := gen.Block()
	block.AddTo(fBlock)
	block.AddInstr(ssa.NewJumpInstr(fBlock))

	fNext, err := ast.False.SSA(fBlock, ctx, gen)
	if err != nil {
		return nil, err
	}

	if fNext == nil && tNext == nil {
		// Both branches terminate.
		return nil, nil
	} else if fNext == nil {
		// False-branch terminates.
		return tNext, nil
	} else if tNext == nil {
		// True-branch terminates.
		return fNext, nil
	}

	// Both branches continue.
	next := gen.Block()
	fNext.AddTo(next)
	fNext.AddInstr(ssa.NewJumpInstr(next))
	tNext.AddTo(next)
	tNext.AddInstr(ssa.NewJumpInstr(next))

	return next, nil
}

func (ast *Return) SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
	*ssa.Block, error) {

	if ctx.Func == nil {
		return nil, fmt.Errorf("%s: return outside function", ast.Loc)
	}
	if len(ctx.Func.Return) != len(ast.Exprs) {
		// TODO %s: too many arguments to return
		// TODO \thave (nil, error)
		// TODO \twant (error)

		// TODO %s: not enough arguments to return
		// TODO \thave ()
		// TODO \twant (error)

		return nil, fmt.Errorf("%s: invalid number of arguments to return",
			ast.Loc)
	}

	for idx, expr := range ast.Exprs {
		r := ctx.Func.Return[idx]
		v, err := gen.Lookup(r.Name, ctx.Scope(), false)
		if err != nil {
			return nil, err
		}

		ctx.Push(v)
		block, err = expr.SSA(block, ctx, gen)
		if err != nil {
			return nil, err
		}
		_, err = ctx.Pop()
		if err != nil {
			return nil, err
		}
	}

	block.AddInstr(ssa.NewJumpInstr(ctx.BlockTail))
	block.AddTo(ctx.BlockTail)

	return nil, nil
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

	// TODO: check that l and r are of same type

	t, err := ctx.Peek()
	if err != nil {
		return nil, err
	}

	// TODO: check that target is of correct type

	var instr ssa.Instr
	switch ast.Op {
	case BinaryPlus:
		instr, err = ssa.NewAddInstr(l.Type, l, r, t)
	case BinaryMinus:
		instr, err = ssa.NewSubInstr(l.Type, l, r, t)
	case BinaryLt:
		instr, err = ssa.NewLtInstr(l.Type, l, r, t)
	case BinaryGt:
		instr, err = ssa.NewGtInstr(l.Type, l, r, t)
	default:
		fmt.Printf("%s %s %s\n", l, ast.Op, r)
		return nil, fmt.Errorf("Binary.SSA '%s' not implemented yet", ast.Op)
	}
	if err != nil {
		return nil, err
	}

	block.AddInstr(instr)

	return block, nil
}

func (ast *VariableRef) SSA(block *ssa.Block, ctx *Codegen,
	gen *ssa.Generator) (*ssa.Block, error) {

	v, err := gen.Lookup(ast.Name, ctx.Scope(), false)
	if err != nil {
		return nil, err
	}
	t, err := ctx.Peek()
	if err != nil {
		return nil, err
	}
	if t.Type.Type == types.Undefined {
		// Replace undefined variable with referenced one.
		ctx.Pop()
		ctx.Push(v)
		return block, nil
	}
	// TODO: check assignement is valid.
	// Assing variable

	block.AddInstr(ssa.NewMovInstr(v, t))

	return block, nil
}
