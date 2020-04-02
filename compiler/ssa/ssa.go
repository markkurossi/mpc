//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/circuits"
	"github.com/markkurossi/mpc/compiler/utils"
)

type SSA struct {
	Inputs    circuit.IO
	Outputs   circuit.IO
	Start     *Block
	Generator *Generator
}

func (code *SSA) CompileCircuit(params *utils.Params) (
	*circuit.Circuit, error) {

	cc, err := circuits.NewCompiler(code.Inputs, code.Outputs)
	if err != nil {
		return nil, err
	}

	err = code.Generator.DefineConstants(cc)
	if err != nil {
		return nil, err
	}

	if params.Verbose {
		fmt.Printf("Creating circuit...\n")
	}
	err = code.Start.Circuit(code.Generator, cc)
	if err != nil {
		return nil, err
	}

	if params.Verbose {
		fmt.Printf("Compiling circuit...\n")
	}
	if params.OptPruneGates {
		pruned := cc.Prune()
		if params.Verbose {
			fmt.Printf(" - Pruned %d gates\n", pruned)
		}
	}
	circ := cc.Compile()
	if params.CircOut != nil {
		if params.Verbose {
			fmt.Printf("Serializing circuit...\n")
		}
		switch params.CircFormat {
		case "mpclc":
			if err := circ.Marshal(params.CircOut); err != nil {
				return nil, err
			}
		case "bristol":
			circ.MarshalBristol(params.CircOut)
		default:
			return nil, fmt.Errorf("unsupported circuit format: %s",
				params.CircFormat)
		}
	}
	if params.CircDotOut != nil {
		circ.Dot(params.CircDotOut)
	}

	return circ, nil
}
