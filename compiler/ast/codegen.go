//
// ast.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"

	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/utils"
)

type Codegen struct {
	logger    *utils.Logger
	Verbose   bool
	Functions map[string]*Func
	Func      *Func
	targets   []ssa.Variable
	Stack     []Compilation
}

func NewCodegen(logger *utils.Logger, functions map[string]*Func) *Codegen {
	return &Codegen{
		logger:    logger,
		Functions: functions,
	}
}

func (ctx *Codegen) Scope() int {
	if ctx.Func != nil {
		return 1
	}
	return 0
}

func (ctx *Codegen) Push(target ssa.Variable) {
	ctx.targets = append(ctx.targets, target)
}

func (ctx *Codegen) Pop() (ssa.Variable, error) {
	if len(ctx.targets) == 0 {
		return ssa.Variable{}, fmt.Errorf("target stack underflow")
	}
	ret := ctx.targets[len(ctx.targets)-1]
	ctx.targets = ctx.targets[:len(ctx.targets)-1]

	return ret, nil
}

func (ctx *Codegen) Peek() (ssa.Variable, error) {
	if len(ctx.targets) == 0 {
		return ssa.Variable{}, fmt.Errorf("target stack underflow")
	}
	return ctx.targets[len(ctx.targets)-1], nil
}

func (ctx *Codegen) NumTargets() int {
	return len(ctx.targets)
}

func (ctx *Codegen) Target(idx int) ssa.Variable {
	return ctx.targets[len(ctx.targets)-1-idx]
}

func (ctx *Codegen) SetTarget(idx int, v ssa.Variable) {
	ctx.targets[len(ctx.targets)-1-idx] = v
}

func (ctx *Codegen) PushCompilation(start, ret, caller *ssa.Block) {
	ctx.Stack = append(ctx.Stack, Compilation{
		Start:  start,
		Return: ret,
		Caller: caller,
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
}

func (ctx *Codegen) Cardinality(ast AST) int {
	switch a := ast.(type) {
	case *Call:
		f, ok := ctx.Functions[a.Name]
		if ok {
			return len(f.Return)
		}
	case *Binary, *VariableRef, *Constant:
		return 1
	}

	return 0
}
