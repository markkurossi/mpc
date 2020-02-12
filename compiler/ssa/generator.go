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
	undef = "$undef"
	anon  = "$_"
)

type Generator struct {
	versions map[string]Variable
	blockID  int
}

func NewGenerator() *Generator {
	return &Generator{
		versions: make(map[string]Variable),
	}
}

func (gen *Generator) UndefVar() Variable {
	v, ok := gen.versions[undef]
	if !ok {
		v = Variable{
			Name: undef,
		}
	} else {
		v.Version = v.Version + 1
	}
	gen.versions[undef] = v
	return v
}

func (gen *Generator) AnonVar(t types.Info) Variable {
	v, ok := gen.versions[anon]
	if !ok {
		v = Variable{
			Name: anon,
			Type: t,
		}
	} else {
		v.Version = v.Version + 1
	}
	gen.versions[anon] = v

	// TODO: check that v.type == t

	return v
}

func (gen *Generator) Var(name string, t types.Info, scope int) Variable {
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

	return v
}

func (gen *Generator) Lookup(name string, scope int) (Variable, error) {
	key := fmtKey(name, scope)
	v, ok := gen.versions[key]
	if !ok {
		return Variable{}, fmt.Errorf("undefined variable %s", name)
	}
	return v, nil
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
