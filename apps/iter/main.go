//
// main.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"log"

	"github.com/markkurossi/mpc/compiler"
	"github.com/markkurossi/mpc/compiler/utils"
)

var template = `
package main

func main(a, b int%d) int {
    return a * b
}
`

func main() {
	for bits := 8; bits < 1024; bits += 8 {
		code := fmt.Sprintf(template, bits)
		var bestLimit int
		var bestCost int

		params := utils.NewParams()

		for limit := 4; limit < 64; limit += 2 {
			params.CircMultArrayTreshold = limit
			circ, _, err := compiler.New(params).Compile(code)
			if err != nil {
				log.Fatalf("Compilation %d:%d failed: %s", bits, limit, err)
			}
			cost := circ.Cost()

			if bestCost == 0 || cost < bestCost {
				bestCost = cost
				bestLimit = limit
			}
		}
		fmt.Printf("%d\t%d\t%d\n", bits, bestLimit, bestCost)
	}
}
