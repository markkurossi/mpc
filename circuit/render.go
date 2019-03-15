//
// render.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"fmt"
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
			to[w] = g
		}
		for _, w := range g.Outputs {
			from[w] = g
		}
	}

	// Assign columns
	var maxCol int
	columns := make(map[int][]*Placement)
	for _, g := range c.Gates {
		place := placements[g.ID]
		for _, w := range g.Inputs {
			src, ok := from[w]
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
