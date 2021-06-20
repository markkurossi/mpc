//
// ast.go
//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/utils"
)

// Codegen implements compilation stack.
type Codegen struct {
	logger   *utils.Logger
	Verbose  bool
	Package  *Package
	Packages map[string]*Package
	Stack    []Compilation
}

// NewCodegen creates a new compilation.
func NewCodegen(logger *utils.Logger, pkg *Package,
	packages map[string]*Package, verbose bool) *Codegen {
	return &Codegen{
		logger:   logger,
		Package:  pkg,
		Packages: packages,
		Verbose:  verbose,
	}
}

// Errorf logs an error message.
func (ctx *Codegen) Errorf(locator utils.Locator, format string,
	a ...interface{}) error {
	return ctx.logger.Errorf(locator.Location(), format, a...)
}

// Func returns the current function in the current compilation.
func (ctx *Codegen) Func() *Func {
	if len(ctx.Stack) == 0 {
		return nil
	}
	return ctx.Stack[len(ctx.Stack)-1].Called
}

// Scope returns the value scope in the current compilation.
func (ctx *Codegen) Scope() int {
	if ctx.Func() != nil {
		return 1
	}
	return 0
}

// PushCompilation pushes a new compilation to the compilation stack.
func (ctx *Codegen) PushCompilation(start, ret, caller *ssa.Block,
	called *Func) {

	ctx.Stack = append(ctx.Stack, Compilation{
		Start:  start,
		Return: ret,
		Caller: caller,
		Called: called,
	})
}

// PopCompilation pops the topmost compilation from the compilation
// stack.
func (ctx *Codegen) PopCompilation() {
	if len(ctx.Stack) == 0 {
		panic("compilation stack underflow")
	}
	ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
}

// Start returns the start block of the current compilation.
func (ctx *Codegen) Start() *ssa.Block {
	return ctx.Stack[len(ctx.Stack)-1].Start
}

// Return returns the return block of the current compilation.
func (ctx *Codegen) Return() *ssa.Block {
	return ctx.Stack[len(ctx.Stack)-1].Return
}

// Caller returns the caller block of the current compilation.
func (ctx *Codegen) Caller() *ssa.Block {
	return ctx.Stack[len(ctx.Stack)-1].Caller
}

// Compilation contains information about a function call compilation.
type Compilation struct {
	Start  *ssa.Block
	Return *ssa.Block
	Caller *ssa.Block
	Called *Func
}
