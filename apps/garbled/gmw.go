//
// Copyright (c) 2020-2026 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"

	"github.com/markkurossi/mpc"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/gmw"
)

func gmwMode(file string, params *utils.Params,
	player int, leader, addr string, loop bool) error {

	fmt.Printf("semi-honest secure GWM protocol\n")
	fmt.Printf(" - player: %d\n", player)
	fmt.Printf(" - leader: %v\n", leader)
	fmt.Printf(" - addr  : %v\n", addr)

	circ, err := loadCircuit(file, params, nil)
	if err != nil {
		return err
	}

	if player >= len(circ.Inputs) {
		return fmt.Errorf("invalid party number %d for %d-party computation",
			player, len(circ.Inputs))
	}

	input, err := circ.Inputs[player].Parse(inputFlag)
	if err != nil {
		return err
	}

	for idx, arg := range circ.Inputs {
		if idx == player {
			fmt.Printf(" + In%d: %s\n", idx, arg)
		} else {
			fmt.Printf(" - In%d: %s\n", idx, arg)
		}
	}

	fmt.Printf(" - Out: %s\n", circ.Outputs)
	fmt.Printf(" - In:  %s\n", inputFlag)

	for {
		// Create network.
		var nw *gmw.Network
		if player == 0 {
			nw, err = gmw.CreateNetwork(addr, circ)
		} else {
			nw, err = gmw.JoinNetwork(leader, addr, player, circ)

		}
		if err != nil {
			return err
		}

		result, err := nw.Run(input)
		if err != nil {
			return err
		}
		nw.Close()

		mpc.PrintResults(result, circ.Outputs, base)
		if !loop {
			break
		}
	}

	return nil
}
