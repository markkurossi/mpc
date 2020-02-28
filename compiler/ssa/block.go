//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"io"
)

type Block struct {
	ID         string
	Name       string
	From       []*Block
	Next       *Block
	BranchCond Variable
	Branch     *Block
	Instr      []Instr
	Bindings   Bindings
	Dead       bool
	Processed  bool
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

	if len(b.Name) > 0 {
		fmt.Fprintf(out, "# %s:\n", b.Name)
	}

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
