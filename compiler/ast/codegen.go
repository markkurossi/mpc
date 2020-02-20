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
	Func      *Func
	targets   []ssa.Variable
	BlockHead *ssa.Block
	BlockTail *ssa.Block
}

func NewCodegen(logger *utils.Logger) *Codegen {
	return &Codegen{
		logger: logger,
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
