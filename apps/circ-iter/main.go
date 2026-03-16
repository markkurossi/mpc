//
// main.go
//
// Copyright (c) 2019-2026 Markku Rossi
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
	step := flag.String("step", "1", "iterator step: integer or xNum")
	size := flag.Int("size", 1, "iterator size in bits")
	inputs := flag.String("i", "",
		"comma-separated list of circuit input sizes")
	peerInputs := flag.String("pi", "",
		"comma-separated list of peer's circuit input sizes")
	gmw := flag.Int("gmw", -1, "semi-honest secure GMW protocol party number")
	flag.Parse()

	if len(flag.Args()) != 1 {
		log.Fatalf("no input file specified")
	}
	file := flag.Args()[0]

	iSizes, iIndices, err := parseInputSizes(*inputs)
	if err != nil {
		log.Fatal(err)
	}
	piSizes, piIndices, err := parseInputSizes(*peerInputs)
	if err != nil {
		log.Fatal(err)
	}
	if len(iIndices) == 0 && len(piIndices) == 0 {
		log.Fatal("no iterators specified")
	}

	var stepExp int
	var stepInc int

	if (*step)[0] == 'x' {
		stepExp, err = strconv.Atoi((*step)[1:])
		if err != nil {
			log.Fatalf("invalid exponential step: %v", *step)
		}
	} else {
		stepInc, err = strconv.Atoi(*step)
		if err != nil {
			log.Fatalf("invalid incremental step: %v", *step)
		}
	}

	increment := func(value, n int) int {
		if stepExp != 0 {
			for range n {
				value *= stepExp
			}
		} else {
			value += n * stepInc
		}
		return value
	}

	results := make(map[int]*result)
	ch := make(chan *result)

	params := utils.NewParams()
	defer params.Close()

	if *gmw >= 0 {
		params.Target = utils.TargetGMW
	}

	params.OptPruneGates = true

	for i := 0; i < *numWorkers; i++ {
		go func(iter int) {
			inputSizes := make([][]int, 2)

			inputSizes[0] = make([]int, len(iSizes))
			copy(inputSizes[0], iSizes)

			inputSizes[1] = make([]int, len(piSizes))
			copy(inputSizes[1], piSizes)

			for ; iter <= *iterEnd; iter = increment(iter, *numWorkers) {
				iterBits := iter * *size
				for _, idx := range iIndices {
					inputSizes[0][idx] = iterBits
				}
				for _, idx := range piIndices {
					inputSizes[1][idx] = iterBits
				}

				// fmt.Printf("iSizes: %v, iter=%v\n", inputSizes, iter)

				circ, _, err := compiler.New(params).CompileFile(file,
					inputSizes)
				if err != nil {
					log.Fatal(err)
				}
				circ.AssignLevels(params.Target)

				ch <- &result{
					iter:  iter,
					stats: circ.Stats,
				}
			}
		}(increment(*iterStart, i))
	}

	next := *iterStart

	fmt.Printf("Iter,XOR,NonXOR,Cost,Depth\n")

outer:
	for result := range ch {
		results[result.iter] = result
		for {
			r, ok := results[next]
			if !ok {
				break
			}
			fmt.Printf("%v,%v,%v,%v,%v\n", r.iter, r.stats.NumXOR(),
				r.stats.NumNonXOR(), r.stats.Cost(), r.stats[circuit.NumLevels])
			if next >= *iterEnd {
				break outer
			}
			next = increment(next, 1)
		}
	}
}

func parseInputSizes(tmpl string) ([]int, []int, error) {
	var result []int
	var indices []int

	parts := strings.Split(tmpl, ",")
	for idx, part := range parts {
		size, err := strconv.Atoi(part)
		if err != nil {
			indices = append(indices, idx)
			size = 0
		}
		result = append(result, size*8)
	}

	return result, indices, nil
}
