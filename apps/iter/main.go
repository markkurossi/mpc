//
// main.go
//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/markkurossi/mpc/compiler"
	"github.com/markkurossi/mpc/compiler/utils"
)

var template = `
package main

func main(a, b int%d) int {
    return a * b
}
`

type result struct {
	bits       int
	bestLimit  int
	bestCost   int
	worstLimit int
	worstCost  int
	costs      []int
}

const (
	numWorkers = 8
	startBits  = 8
)

func main() {
	results := make(map[int]*result)
	ch := make(chan *result)

	for i := 0; i < numWorkers; i++ {
		go func(bits int) {
			for ; ; bits += numWorkers {
				code := fmt.Sprintf(template, bits)
				var bestLimit int
				var bestCost int
				var worstLimit int
				var worstCost int
				var costs []int

				params := utils.NewParams()

				for limit := 8; limit <= 22; limit++ {
					params.CircMultArrayTreshold = limit
					circ, _, err := compiler.New(params).Compile(code)
					if err != nil {
						log.Fatalf("Compilation %d:%d failed: %s\n%s",
							bits, limit, err, code)
					}
					cost := circ.Cost()
					costs = append(costs, cost)

					if bestCost == 0 || cost < bestCost {
						bestCost = cost
						bestLimit = limit
					}
					if cost > worstCost {
						worstCost = cost
						worstLimit = limit
					}
				}
				ch <- &result{
					bits:       bits,
					bestLimit:  bestLimit,
					bestCost:   bestCost,
					worstLimit: worstLimit,
					worstCost:  worstCost,
					costs:      costs,
				}
			}
		}(startBits + i)
	}

	next := startBits

	for result := range ch {
		results[result.bits] = result
		for {
			r, ok := results[next]
			if !ok {
				break
			}
			if r.bestLimit == 19 {
				fmt.Printf("\t// %d: %d,\t%d\t%.4f\t%s\n",
					r.bits, r.bestLimit,
					r.bestCost, float64(r.bestCost)/float64(r.worstCost),
					Sparkline(r.costs))
			} else {
				fmt.Printf("\t%d: %d, //\t%d\t%.4f\t%s\n",
					r.bits, r.bestLimit,
					r.bestCost, float64(r.bestCost)/float64(r.worstCost),
					Sparkline(r.costs))
			}
			next++
		}
	}
}

// Sparkline creates a histogram chart of values. The chart is scaled
// to [min...max] containing differences between values.
func Sparkline(values []int) string {
	if len(values) == 0 {
		return ""
	}

	min := math.MaxInt32
	max := math.MinInt32

	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	delta := max - min

	var sb strings.Builder
	for _, v := range values {
		var tick int
		if delta == 0 {
			tick = 4
		} else {
			tick = (v - min) * 7 / delta
		}
		if v == min && false {
			sb.WriteString("\x1b[92m")
			sb.WriteRune(rune(0x2581 + tick))
			sb.WriteString("\x1b[0m")
		} else {
			sb.WriteRune(rune(0x2581 + tick))
		}
	}
	return sb.String()
}
