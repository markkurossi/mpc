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

func (b *Block) PP(out io.Writer, seen map[string]bool) {
	if seen[b.ID] {
		return
	}
	seen[b.ID] = true

	fmt.Fprintf(out, "%s:\n", b.ID)
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
