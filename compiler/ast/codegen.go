//
// ast.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/utils"
)

type Codegen struct {
	logger   *utils.Logger
	Verbose  bool
	Packages map[string]*Package
	Stack    []Compilation
}

func NewCodegen(logger *utils.Logger, packages map[string]*Package,
	verbose bool) *Codegen {
	return &Codegen{
		logger:   logger,
		Packages: packages,
		Verbose:  verbose,
	}
}

func (ctx *Codegen) Func() *Func {
	if len(ctx.Stack) == 0 {
		return nil
	}
	return ctx.Stack[len(ctx.Stack)-1].Called
}

func (ctx *Codegen) Scope() int {
	if ctx.Func() != nil {
		return 1
	}
	return 0
}

func (ctx *Codegen) PushCompilation(start, ret, caller *ssa.Block,
	called *Func) {

	ctx.Stack = append(ctx.Stack, Compilation{
		Start:  start,
		Return: ret,
		Caller: caller,
		Called: called,
	})
}

func (ctx *Codegen) PopCompilation() {
	if len(ctx.Stack) == 0 {
		panic("compilation stack underflow")
	}
	ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
}

func (ctx *Codegen) Start() *ssa.Block {
	return ctx.Stack[len(ctx.Stack)-1].Start
}

func (ctx *Codegen) Return() *ssa.Block {
	return ctx.Stack[len(ctx.Stack)-1].Return
}

func (ctx *Codegen) Caller() *ssa.Block {
	return ctx.Stack[len(ctx.Stack)-1].Caller
}

type Compilation struct {
	Start  *ssa.Block
	Return *ssa.Block
	Caller *ssa.Block
	Called *Func
}
