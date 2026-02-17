//
// Copyright (c) 2020-2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"

	"github.com/markkurossi/mpc"
	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/gmw"
)

func gmwMode(file string, params *utils.Params,
	party, numParties int, leader, addr string, loop bool) error {

	fmt.Printf("semi-honest secure GWM protocol\n")
	if party == 0 {
		fmt.Printf(" - party : %d/%d\n", party, numParties)
	} else {
		fmt.Printf(" - party : %d\n", party)
	}
	fmt.Printf(" - leader: %v\n", leader)
	fmt.Printf(" - addr  : %v\n", addr)

	inputSizes, err := circuit.InputSizes(inputFlag)
	if err != nil {
		return err
	}

	for {
		timing := circuit.NewTiming()

		// Create network.
		var nw *gmw.Network
		if party == 0 {
			nw, err = gmw.CreateNetwork(addr, numParties)
		} else {
			nw, err = gmw.JoinNetwork(leader, addr, party)

		}
		if err != nil {
			return err
		}
		err = nw.Connect(inputSizes)
		if err != nil {
			return err
		}
		ioStats := nw.Stats().Sum()
		timing.Sample("Network", []string{circuit.FileSize(ioStats).String()})

		circ, err := loadCircuit(file, params, nw.InputSizes())
		if err != nil {
			return err
		}
		if nw.NumParties() != circ.NumParties() {
			return fmt.Errorf("invalid %v-party circuit for %d-party MPC",
				circ.NumParties(), numParties)
		}
		timing.Sample("Compile", nil)

		input, err := circ.Inputs[party].Parse(inputFlag)
		if err != nil {
			return err
		}

		for idx, arg := range circ.Inputs {
			if idx == party {
				fmt.Printf(" + In%d: %s\n", idx, arg)
			} else {
				fmt.Printf(" - In%d: %s\n", idx, arg)
			}
		}

		fmt.Printf(" - Out: %s\n", circ.Outputs)
		fmt.Printf(" - In:  %s\n", inputFlag)

		result, err := nw.Run(input, circ, verbose)
		if err != nil {
			return err
		}
		xfer := nw.Stats().Sum() - ioStats
		timing.Sample("Eval", []string{circuit.FileSize(xfer).String()})

		if verbose {
			timing.Print(nw.Stats())
		}

		nw.Close()

		mpc.PrintResults(result, circ.Outputs, base)
		if !loop {
			break
		}
	}

	return nil
}
