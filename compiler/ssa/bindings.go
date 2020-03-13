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

var (
	_ BindingValue = &Variable{}
	_ BindingValue = &Select{}
)

type Bindings []Binding

func (bindings *Bindings) Set(v Variable, val *Variable) {
	for idx, b := range *bindings {
		if b.Name == v.Name && b.Scope == v.Scope {
			b.Type = v.Type
			if val != nil {
				b.Bound = val
			} else {
				b.Bound = &v
			}
			(*bindings)[idx] = b
			return
		}
	}

	b := Binding{
		Name:  v.Name,
		Scope: v.Scope,
		Type:  v.Type,
	}
	if val != nil {
		b.Bound = val
	} else {
		b.Bound = &v
	}

	*bindings = append(*bindings, b)
}

func (bindings Bindings) Get(name string) (ret Binding, err error) {
	for _, b := range bindings {
		if b.Name == name {
			if len(ret.Name) == 0 || b.Scope > ret.Scope {
				ret = b
			}
		}
	}
	if len(ret.Name) == 0 {
		return ret, fmt.Errorf("undefined variable '%s'", name)
	}
	return
}

func (b Bindings) Clone() Bindings {
	result := make(Bindings, len(b))
	copy(result, b)
	return result
}

func (tBindings Bindings) Merge(cond Variable, fBindings Bindings) Bindings {
	names := make(map[string]bool)

	for _, b := range tBindings {
		names[b.Name] = true
	}
	for _, b := range fBindings {
		names[b.Name] = true
	}

	var result Bindings
	for name, _ := range names {
		bTrue, err1 := tBindings.Get(name)
		bFalse, err2 := fBindings.Get(name)
		if err1 != nil && err2 != nil {
			continue
		} else if err1 != nil {
			result = append(result, bFalse)
		} else if err2 != nil {
			result = append(result, bTrue)
		} else {
			if bTrue.Bound.Equal(bFalse.Bound) {
				result = append(result, bTrue)
			} else {
				result = append(result, Binding{
					Name:  name,
					Scope: bTrue.Scope,
					Type:  bTrue.Type,
					Bound: &Select{
						Cond:  cond,
						Type:  bTrue.Type,
						True:  bTrue.Bound,
						False: bFalse.Bound,
					},
				})
			}
		}
	}
	return result
}

type Binding struct {
	Name  string
	Scope int
	Type  types.Info
	Bound BindingValue
}

func (b Binding) String() string {
	return fmt.Sprintf("%s@%d/%s=%s", b.Name, b.Scope, b.Type, b.Bound)
}

func (b Binding) Value(block *Block, gen *Generator) Variable {
	return b.Bound.Value(block, gen)
}

type BindingValue interface {
	Equal(o BindingValue) bool
	Value(block *Block, gen *Generator) Variable
}

type Select struct {
	Cond     Variable
	Type     types.Info
	True     BindingValue
	False    BindingValue
	Resolved Variable
}

func (phi *Select) String() string {
	return fmt.Sprintf("\u03D5(%s|%s|%s)/%s",
		phi.Cond, phi.True, phi.False, phi.Type)
}

func (phi *Select) Equal(other BindingValue) bool {
	o, ok := other.(*Select)
	if !ok {
		return false
	}
	if !phi.Cond.Equal(&o.Cond) {
		return false
	}
	return phi.True.Equal(o.True) && phi.False.Equal(o.False)
}

func (phi *Select) Value(block *Block, gen *Generator) Variable {
	if phi.Resolved.Type.Type != types.Undefined {
		return phi.Resolved
	}

	t := phi.True.Value(block, gen)
	f := phi.False.Value(block, gen)
	v := gen.AnonVar(phi.Type)
	block.AddInstr(NewPhiInstr(phi.Cond, t, f, v))

	phi.Resolved = v

	return v
}
