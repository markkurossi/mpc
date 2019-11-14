//
// parser.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"fmt"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
)

func (unit *Unit) Compile() (*circuit.Circuit, error) {
	main, ok := unit.Functions["main"]
	if !ok {
		return nil, fmt.Errorf("No main function")
	}
	if len(main.Args) != 2 {
		return nil, fmt.Errorf("Only 2 argument main() supported")
	}
	var returnBits int
	for _, rt := range main.Return {
		returnBits += rt.Bits
	}

	c := circuits.NewCompiler(main.Args[0].Type.Bits, main.Args[1].Type.Bits,
		returnBits)

	fmt.Printf("n1=%d, n2=%d, n3=%d\n", c.N1, c.N2, c.N3)

	return nil, fmt.Errorf("Compile not implemented yet")
}
