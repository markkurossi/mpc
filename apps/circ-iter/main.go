//
// main.go
//
// Copyright (c) 2019-2025 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler"
	"github.com/markkurossi/mpc/compiler/utils"
)

type result struct {
	iter  int
	stats circuit.Stats
}

func main() {
	numWorkers := flag.Int("workers", 8, "number of workers")
	iterStart := flag.Int("start", 8, "iterator start value")
	iterEnd := flag.Int("end", 1024, "iterator end value (inclusive)")
	step := flag.Int("step", 1, "iterator step")
	inputs := flag.String("i", "",
		"comma-separated list of circuit input sizes")
	peerInputs := flag.String("pi", "",
		"comma-separated list of peer's circuit input sizes")
	flag.Parse()

	if len(flag.Args()) != 1 {
		log.Fatalf("no input file specified")
	}
	file := flag.Args()[0]

	iSizes, iIdx, err := parseInputSizes(*inputs)
	if err != nil {
		log.Fatal(err)
	}
	piSizes, piIdx, err := parseInputSizes(*peerInputs)
	if err != nil {
		log.Fatal(err)
	}
	if iIdx >= 0 && piIdx >= 0 {
		log.Fatalf("multiple iterators specified")
	}
	var ix, iy int
	if iIdx >= 0 {
		iy = 0
		ix = iIdx
	} else if piIdx >= 0 {
		iy = 1
		ix = piIdx
	} else {
		log.Fatal("no iterators specified")
	}

	results := make(map[int]*result)
	ch := make(chan *result)

	params := utils.NewParams()
	defer params.Close()

	params.OptPruneGates = true

	for i := 0; i < *numWorkers; i++ {
		go func(iter int) {
			inputSizes := make([][]int, 2)

			inputSizes[0] = make([]int, len(iSizes))
			copy(inputSizes[0], iSizes)

			inputSizes[1] = make([]int, len(piSizes))
			copy(inputSizes[1], piSizes)

			for ; iter <= *iterEnd; iter += *numWorkers * *step {
				inputSizes[iy][ix] = iter * 8
				// fmt.Printf("iSizes: %v, iter=%v\n", inputSizes, iter)

				circ, _, err := compiler.New(params).CompileFile(file,
					inputSizes)
				if err != nil {
					log.Fatal(err)
				}
				circ.AssignLevels()

				ch <- &result{
					iter:  iter,
					stats: circ.Stats,
				}
			}
		}(*iterStart + i**step)
	}

	next := *iterStart

	fmt.Printf("Iter,XOR,NonXOR,Cost\n")

outer:
	for result := range ch {
		results[result.iter] = result
		for {
			r, ok := results[next]
			if !ok {
				break
			}
			fmt.Printf("%v,%v,%v,%v\n", r.iter, r.stats.NumXOR(),
				r.stats.NumNonXOR(), r.stats.Cost())
			if next >= *iterEnd {
				break outer
			}
			next += *step
		}
	}
}

func parseInputSizes(tmpl string) ([]int, int, error) {
	var result []int

	index := -1

	parts := strings.Split(tmpl, ",")
	for idx, part := range parts {
		size, err := strconv.Atoi(part)
		if err != nil {
			if index >= 0 {
				return nil, -1, fmt.Errorf("multiple iterators")
			}
			index = idx
			size = 0
		}
		result = append(result, size*8)
	}

	return result, index, nil
}
