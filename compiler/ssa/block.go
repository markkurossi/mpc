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
	ID    string
	From  []*Block
	To    []*Block
	Instr []Instr
}

func (b *Block) String() string {
	return b.ID
}

func (b *Block) Equals(o *Block) bool {
	return b.ID == o.ID
}

func (b *Block) AddFrom(o *Block) {
	b.addFrom(o)
	o.addTo(b)
}

func (b *Block) addFrom(o *Block) {
	for _, f := range b.From {
		if f.Equals(o) {
			return
		}
	}
	b.From = append(b.From, o)
}

func (b *Block) AddTo(o *Block) {
	b.addTo(o)
	o.addFrom(b)
}

func (b *Block) addTo(o *Block) {
	for _, f := range b.To {
		if f.Equals(o) {
			return
		}
	}
	b.To = append(b.To, o)
}

func (b *Block) AddInstr(instr Instr) {
	b.Instr = append(b.Instr, instr)
}

func (b *Block) PP(out io.Writer, seen map[string]bool) {
	if seen[b.ID] {
		return
	}
	seen[b.ID] = true

	fmt.Fprintf(out, "%s:\t\t\t\t\tfrom: %v, to: %v\n", b.ID, b.From, b.To)
	for _, i := range b.Instr {
		i.PP(out)
	}
	for _, to := range b.To {
		to.PP(out, seen)
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

	for _, to := range b.To {
		to.DotNodes(out, seen)
	}
}

func (b *Block) DotLinks(out io.Writer, seen map[string]bool) {
	if seen[b.ID] {
		return
	}
	seen[b.ID] = true
	for _, to := range b.To {
		fmt.Fprintf(out, "  %s -> %s [label=\"%s\"];\n", b.ID, to.ID, to.ID)
	}

	for _, to := range b.To {
		to.DotLinks(out, seen)
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
