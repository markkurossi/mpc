//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"

	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
)

const (
	anon = "%_"
)

// Generator implements code generator.
type Generator struct {
	Params    *utils.Params
	versions  map[string]Variable
	blockID   int
	constants map[string]ConstantInst
}

// ConstantInst defines a constant variable instance.
type ConstantInst struct {
	Count int
	Const Variable
}

// NewGenerator creates a new code generator.
func NewGenerator(params *utils.Params) *Generator {
	return &Generator{
		Params:    params,
		versions:  make(map[string]Variable),
		constants: make(map[string]ConstantInst),
	}
}

// Constants returns the constants.
func (gen *Generator) Constants() map[string]ConstantInst {
	return gen.constants
}

// UndefVar creates a new undefined variable.
func (gen *Generator) UndefVar() Variable {
	v, ok := gen.versions[anon]
	if !ok {
		v = Variable{
			Name: anon,
		}
	} else {
		v.Version = v.Version + 1
	}
	v.Type = types.Info{
		Type: types.Undefined,
		Bits: 0,
	}
	gen.versions[anon] = v
	return v
}

// AnonVar creates a new anonymous variable.
func (gen *Generator) AnonVar(t types.Info) Variable {
	v, ok := gen.versions[anon]
	if !ok {
		v = Variable{
			Name: anon,
		}
	} else {
		v.Version = v.Version + 1
	}
	v.Type = t
	gen.versions[anon] = v

	return v
}

// NewVar creates a new variable with the name, type, and scope.
func (gen *Generator) NewVar(name string, t types.Info, scope int) (
	Variable, error) {

	key := fmtKey(name, scope)
	v, ok := gen.versions[key]
	if !ok {
		v = Variable{
			Name:  name,
			Scope: scope,
			Type:  t,
		}
	} else {
		v.Version = v.Version + 1
		v.Type = t
	}
	gen.versions[key] = v

	return v, nil
}

// AddConstant adds a reference to the constant.
func (gen *Generator) AddConstant(c Variable) {
	inst, ok := gen.constants[c.Name]
	if !ok {
		inst = ConstantInst{
			Count: 1,
			Const: c,
		}
	} else {
		inst.Count++
	}
	gen.constants[c.Name] = inst
}

// RemoveConstant drops a reference from the constant.
func (gen *Generator) RemoveConstant(c Variable) {
	inst, ok := gen.constants[c.Name]
	if !ok {
		return
	}
	inst.Count--
	if inst.Count == 0 {
		delete(gen.constants, c.Name)
	} else {
		gen.constants[c.Name] = inst
	}
}

func fmtKey(name string, scope int) string {
	return fmt.Sprintf("%s@%d", name, scope)
}

// Block creates a new basic block.
func (gen *Generator) Block() *Block {
	block := &Block{
		ID: fmt.Sprintf("l%d", gen.blockID),
	}
	gen.blockID++

	return block
}

// NextBlock adds the next basic block to the argument block.
func (gen *Generator) NextBlock(b *Block) *Block {
	n := gen.Block()
	n.Bindings = b.Bindings.Clone()
	b.SetNext(n)
	return n
}

// BranchBlock creates a new branch basic block.
func (gen *Generator) BranchBlock(b *Block) *Block {
	n := gen.Block()
	n.Bindings = b.Bindings.Clone()
	b.SetBranch(n)
	return n
}
