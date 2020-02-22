//
// Copyright (c) 2019-2020 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
	"io"
)

type Placement struct {
	Col int
	Row float64
	G   *Gate
}

func (c *Circuit) Render() {
	from := make(map[int]*Gate)
	to := make(map[int]*Gate)

	placements := make(map[uint32]*Placement)

	// Create wire mappings.
	for _, g := range c.Gates {
		placements[g.ID] = &Placement{
			G: g,
		}
		for _, w := range g.Inputs {
			to[w.ID()] = g
		}
		for _, w := range g.Outputs {
			from[w.ID()] = g
		}
	}

	// Assign columns
	var maxCol int
	columns := make(map[int][]*Placement)
	for _, g := range c.Gates {
		place := placements[g.ID]
		for _, w := range g.Inputs {
			src, ok := from[w.ID()]
			if ok {
				srcPlace := placements[src.ID]
				if srcPlace.Col >= place.Col {
					place.Col = srcPlace.Col + 1
				}
			}
		}
		columns[place.Col] = append(columns[place.Col], place)
		if place.Col > maxCol {
			maxCol = place.Col
		}
	}

	fmt.Printf("#cols=%d\n", maxCol)
	for i := 0; i <= maxCol; i++ {
		fmt.Printf("Col%d:\t%d\n", i, len(columns[i]))
	}
}

func (c *Circuit) Dot(out io.Writer) {
	fmt.Fprintf(out, "digraph circuit\n{\n")
	fmt.Fprintf(out, "  overlap=scale;\n")
	fmt.Fprintf(out, "  node\t[fontname=\"Helvetica\"];\n")
	fmt.Fprintf(out, "  {\n    node [shape=plaintext];\n")
	for w := 0; w < c.NumWires; w++ {
		fmt.Fprintf(out, "    w%d\t[label=\"%d\"];\n", w, w)
	}
	fmt.Fprintf(out, "  }\n")

	fmt.Fprintf(out, "  {\n    node [shape=box];\n")
	for idx, gate := range c.Gates {
		fmt.Fprintf(out, "    g%d\t[label=\"%s\"];\n", idx, gate.Op)
	}
	fmt.Fprintf(out, "  }\n")

	if true {
		fmt.Fprintf(out, "  {  rank=same")
		for w := 0; w < c.N1.Size()+c.N2.Size(); w++ {
			fmt.Fprintf(out, "; w%d", w)
		}
		fmt.Fprintf(out, ";}\n")

		fmt.Fprintf(out, "  {  rank=same")
		for w := 0; w < c.N3.Size(); w++ {
			fmt.Fprintf(out, "; w%d", c.NumWires-w-1)
		}
		fmt.Fprintf(out, ";}\n")
	}

	for idx, gate := range c.Gates {
		for _, i := range gate.Inputs {
			fmt.Fprintf(out, "  w%d -> g%d;\n", i, idx)
		}
		for _, o := range gate.Outputs {
			fmt.Fprintf(out, "  g%d -> w%d;\n", idx, o)
		}
	}
	fmt.Fprintf(out, "}\n")
}
