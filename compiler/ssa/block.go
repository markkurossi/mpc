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
