//
// main.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/markkurossi/mpc/circuit"
)

func main() {
	render := flag.Bool("r", false, "Render circuit")
	flag.Parse()

	for _, file := range flag.Args() {
		f, err := os.Open(file)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		c, err := circuit.Parse(f)
		if err != nil {
			log.Fatal(err)
		}

		if *render {
			c.Render()
			continue
		}

		fmt.Printf("digraph circuit\n{\n")
		fmt.Printf("  overlap=scale;\n")
		fmt.Printf("  node\t[fontname=\"Helvetica\"];\n")
		fmt.Printf("  {\n    node [shape=plaintext];\n")
		for w := 0; w < c.NumWires; w++ {
			fmt.Printf("    w%d\t[label=\"%d\"];\n", w, w)
		}
		fmt.Printf("  }\n")

		fmt.Printf("  {\n    node [shape=box];\n")
		for idx, gate := range c.Gates {
			fmt.Printf("    g%d\t[label=\"%s\"];\n", idx, gate.Op)
		}
		fmt.Printf("  }\n")

		if false {
			fmt.Printf("  {  rank=same")
			for w := 0; w < c.N1+c.N2; w++ {
				fmt.Printf("; w%d", w)
			}
			fmt.Printf(";}\n")

			fmt.Printf("  {  rank=same")
			for w := 0; w < c.N3; w++ {
				fmt.Printf("; w%d", c.NumWires-w-1)
			}
			fmt.Printf(";}\n")
		}

		for idx, gate := range c.Gates {
			for _, i := range gate.Inputs {
				fmt.Printf("  w%d -> g%d;\n", i, idx)
			}
			for _, o := range gate.Outputs {
				fmt.Printf("  g%d -> w%d;\n", idx, o)
			}
		}
		fmt.Printf("}\n")
	}
}
