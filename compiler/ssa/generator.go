//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"

	"github.com/markkurossi/mpc/compiler/types"
)

const (
	anon = "%_"
)

type Generator struct {
	versions  map[string]Variable
	blockID   int
	constants map[string]Variable
}

func NewGenerator() *Generator {
	return &Generator{
		versions:  make(map[string]Variable),
		constants: make(map[string]Variable),
	}
}

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
	}
	gen.versions[key] = v

	// TODO: check that v.type == t

	return v, nil
}

func (gen *Generator) Lookup(name string, scope int, assign bool) (
	Variable, error) {

	key := fmtKey(name, scope)
	v, ok := gen.versions[key]
	if !ok {
		return Variable{}, fmt.Errorf("undefined variable '%s'", name)
	}
	if assign {
		v.Version = v.Version + 1
		gen.versions[key] = v
	}
	return v, nil
}

func (gen *Generator) AddConstant(c Variable) {
	_, ok := gen.constants[c.Name]
	if !ok {
		gen.constants[c.Name] = c
	}
}

func fmtKey(name string, scope int) string {
	return fmt.Sprintf("%s@%d", name, scope)
}

func (gen *Generator) Block() *Block {
	block := &Block{
		ID: fmt.Sprintf("l%d", gen.blockID),
	}
	gen.blockID++

	return block
}

func (gen *Generator) NextBlock(b *Block) *Block {
	n := gen.Block()
	n.Bindings = b.Bindings.Clone()
	b.SetNext(n)
	return n
}

func (gen *Generator) BranchBlock(b *Block) *Block {
	n := gen.Block()
	n.Bindings = b.Bindings.Clone()
	b.SetBranch(n)
	return n
}
