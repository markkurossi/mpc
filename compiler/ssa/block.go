//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"io"

	"github.com/markkurossi/mpc/compiler/types"
)

type Block struct {
	ID         string
	From       []*Block
	Next       *Block
	BranchCond Variable
	Branch     *Block
	Instr      []Instr
	Bindings   Bindings
	Dead       bool
}

func (b *Block) String() string {
	return b.ID
}

func (b *Block) Equals(o *Block) bool {
	return b.ID == o.ID
}

func (b *Block) SetNext(o *Block) {
	if b.Next != nil && b.Next.ID != o.ID {
		panic(fmt.Sprintf("%s.Next already set to %s, now setting to %s",
			b.ID, b.Next.ID, o.ID))
	}
	b.Next = o
	o.addFrom(b)
}

func (b *Block) SetBranch(o *Block) {
	if b.Branch != nil && b.Branch.ID != o.ID {
		panic(fmt.Sprintf("%s.Branch already set to %s, now setting to %s",
			b.ID, b.Next.ID, o.ID))
	}
	b.Branch = o
	o.addFrom(b)
}

func (b *Block) addFrom(o *Block) {
	for _, f := range b.From {
		if f.Equals(o) {
			return
		}
	}
	b.From = append(b.From, o)
}

func (b *Block) AddInstr(instr Instr) {
	b.Instr = append(b.Instr, instr)
}

func (b *Block) ReturnBinding(name string, retBlock *Block, gen *Generator) (
	v Variable, err error) {

	if b.Branch == nil {
		// Sequential block, return latest value
		if b.Next != nil {
			v, err = b.Next.ReturnBinding(name, retBlock, gen)
			if err == nil {
				return v, nil
			}
			// Next didn't have value, take ours below.
		}
		bind, err := b.Bindings.Get(name)
		if err != nil {
			return v, err
		}
		return bind.Value(retBlock, gen), nil
	}
	vTrue, err := b.Branch.ReturnBinding(name, retBlock, gen)
	if err != nil {
		return v, err
	}
	vFalse, err := b.Next.ReturnBinding(name, retBlock, gen)
	if err != nil {
		return v, err
	}
	if vTrue.Equal(&vFalse) {
		return vTrue, nil
	}

	v = gen.AnonVar(vTrue.Type)
	retBlock.AddInstr(NewPhiInstr(b.BranchCond, vTrue, vFalse, v))

	return v, nil
}

func (b *Block) PP(out io.Writer, seen map[string]bool) {
	if seen[b.ID] {
		return
	}
	seen[b.ID] = true

	fmt.Fprintf(out, "%s:\n", b.ID)
	for _, i := range b.Instr {
		i.PP(out)
	}
	if b.Next != nil {
		b.Next.PP(out, seen)
	}
	if b.Branch != nil {
		b.Branch.PP(out, seen)
	}
}

func PP(out io.Writer, block *Block) {
	block.PP(out, make(map[string]bool))
}

func (b *Block) DotNodes(out io.Writer, seen map[string]bool) {
	if seen[b.ID] {
		return
	}
	seen[b.ID] = true

	var label string
	if len(b.Instr) == 1 {
		label = b.Instr[0].String()
	} else {
		for _, i := range b.Instr {
			label += i.String()
			label += "\\l"
		}
	}

	fmt.Fprintf(out, "  %s [label=\"%s\"]\n", b.ID, label)

	if b.Next != nil {
		b.Next.DotNodes(out, seen)
	}
	if b.Branch != nil {
		b.Branch.DotNodes(out, seen)
	}
}

func (b *Block) DotLinks(out io.Writer, seen map[string]bool) {
	if seen[b.ID] {
		return
	}
	seen[b.ID] = true
	if b.Next != nil {
		fmt.Fprintf(out, "  %s -> %s [label=\"%s\"];\n",
			b.ID, b.Next.ID, b.Next.ID)
	}
	if b.Branch != nil {
		fmt.Fprintf(out, "  %s -> %s [label=\"%s\"];\n",
			b.ID, b.Branch.ID, b.Branch.ID)
	}

	if b.Next != nil {
		b.Next.DotLinks(out, seen)
	}
	if b.Branch != nil {
		b.Branch.DotLinks(out, seen)
	}
}

func Dot(out io.Writer, block *Block) {
	fontname := "Courier"
	fontsize := 10

	fmt.Fprintln(out, "digraph program {")
	fmt.Fprintf(out, "  node [shape=box fontname=\"%s\" fontsize=\"%d\"]\n",
		fontname, fontsize)
	fmt.Fprintf(out, "  edge [fontname=\"%s\" fontsize=\"%d\"]\n",
		fontname, fontsize)
	block.DotNodes(out, make(map[string]bool))
	block.DotLinks(out, make(map[string]bool))
	fmt.Fprintln(out, "}")
}

type Bindings []Binding

func (bindings *Bindings) Set(v Variable) {
	for idx, b := range *bindings {
		if b.Name == v.Name && b.Scope == v.Scope {
			b.Type = v.Type
			b.value = &v
			(*bindings)[idx] = b
			return
		}
	}
	*bindings = append(*bindings, Binding{
		Name:  v.Name,
		Scope: v.Scope,
		Type:  v.Type,
		value: &v,
	})
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
			if bTrue.value.Equal(bFalse.value) {
				result = append(result, bTrue)
			} else {
				result = append(result, Binding{
					Name:  name,
					Scope: bTrue.Scope,
					Type:  bTrue.Type,
					value: &Select{
						Cond:  cond,
						Type:  bTrue.Type,
						True:  bTrue.value,
						False: bFalse.value,
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
	value BindingValue
}

func (b Binding) String() string {
	return fmt.Sprintf("%s@%d/%s=%s", b.Name, b.Scope, b.Type, b.value)
}

func (b Binding) Value(block *Block, gen *Generator) Variable {
	return b.value.Value(block, gen)
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

var (
	_ BindingValue = &Variable{}
	_ BindingValue = &Select{}
)
