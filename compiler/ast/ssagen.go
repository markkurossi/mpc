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

func (ast List) SSA(ctx *Codegen, gen *ssa.Generator) error {
	return fmt.Errorf("List.SSA not implemented yet")
}

func (ast *Func) SSA(ctx *Codegen, gen *ssa.Generator) error {
	ctx.Func = ast
	defer func() {
		ctx.Func = nil
	}()

	// Define arguments.
	for idx, arg := range ast.Args {
		a := gen.Var(arg.Name, arg.Type, ctx.Scope())
		fmt.Printf("args[%d]=%s\n", idx, a)
	}
	// Define return variables.
	for idx, ret := range ast.Return {
		if len(ret.Name) == 0 {
			ret.Name = fmt.Sprintf("$ret%d", idx)
		}
		r := gen.Var(ret.Name, ret.Type, ctx.Scope())
		fmt.Printf("ret[%d]=%s\n", idx, r)
	}

	for _, b := range ast.Body {
		err := b.SSA(ctx, gen)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ast *If) SSA(ctx *Codegen, gen *ssa.Generator) error {
	return fmt.Errorf("If.SSA not implemented yet")
}

func (ast *Return) SSA(ctx *Codegen, gen *ssa.Generator) error {
	if ctx.Func == nil {
		return fmt.Errorf("%s: return outside function", ast.Loc)
	}
	if len(ctx.Func.Return) != len(ast.Exprs) {
		// TODO %s: too many arguments to return
		// TODO \thave (nil, error)
		// TODO \twant (error)

		// TODO %s: not enough arguments to return
		// TODO \thave ()
		// TODO \twant (error)

		return fmt.Errorf("%s: invalid number of arguments to return", ast.Loc)
	}

	for idx, expr := range ast.Exprs {
		r := ctx.Func.Return[idx]
		v, err := gen.Lookup(r.Name, ctx.Scope())
		if err != nil {
			return err
		}

		ctx.Push(v)
		err = expr.SSA(ctx, gen)
		if err != nil {
			return err
		}
		_, err = ctx.Pop()
		if err != nil {
			return err
		}
	}

	ctx.BlockCurr.AddInstr(ssa.NewJumpInstr(ctx.BlockTail))
	ctx.BlockCurr.AddTo(ctx.BlockTail)

	ctx.AddBlock(gen.Block())

	return nil
}

func (ast *Binary) SSA(ctx *Codegen, gen *ssa.Generator) error {
	ctx.Push(gen.UndefVar())
	err := ast.Left.SSA(ctx, gen)
	if err != nil {
		return err
	}
	l, err := ctx.Pop()
	if err != nil {
		return err
	}

	ctx.Push(gen.UndefVar())
	err = ast.Right.SSA(ctx, gen)
	if err != nil {
		return err
	}
	r, err := ctx.Pop()
	if err != nil {
		return err
	}

	// TODO: check that l and r are of same type

	t, err := ctx.Peek()
	if err != nil {
		return err
	}

	// TODO: check that target is of correct type

	var instr ssa.Instr
	switch ast.Op {
	case BinaryPlus:
		instr, err = ssa.NewAddInstr(t.Type, l, r, t)
		if err != nil {
			return err
		}

	case BinaryMinus:
		instr, err = ssa.NewSubInstr(t.Type, l, r, t)
		if err != nil {
			return err
		}

	default:
		fmt.Printf("%s %s %s\n", l, ast.Op, r)
		return fmt.Errorf("Binary.SSA '%s' not implemented yet", ast.Op)
	}

	ctx.BlockCurr.AddInstr(instr)

	return nil
}

func (ast *VariableRef) SSA(ctx *Codegen, gen *ssa.Generator) error {
	v, err := gen.Lookup(ast.Name, ctx.Scope())
	if err != nil {
		return err
	}
	t, err := ctx.Peek()
	if err != nil {
		return err
	}
	if t.Type.Type == types.Undefined {
		// Replace undefined variable with referenced one.
		ctx.Pop()
		ctx.Push(v)
		return nil
	}
	// TODO: check assignement is valid.
	// Assing variable
	return fmt.Errorf("VariableRef.SSA: variable assignment")
}
