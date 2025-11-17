//
// Copyright (c) 2020-2025 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"log"

	"github.com/markkurossi/mpc"
	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/env"
	"github.com/markkurossi/mpc/p2p"
)

func bmrMode(file string, params *utils.Params, player int) error {
	cfg := &env.Config{}
	rand := cfg.GetRandom()

	fmt.Printf("semi-honest secure BMR protocol\n")
	fmt.Printf("player: %d\n", player)

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

	// Create network.
	addr := makeAddr(player)
	nw, err := p2p.NewNetwork(rand, addr, player)
	if err != nil {
		return err
	}
	defer nw.Close()

	numPlayers := len(circ.Inputs)

	for i := 0; i < numPlayers; i++ {
		if i == player {
			continue
		}
		err := nw.AddPeer(makeAddr(i), i)
		if err != nil {
			return err
		}
	}

	log.Printf("Network created\n")

	result, err := circuit.Player(cfg, nw, circ, input, verbose)
	if err != nil {
		return err
	}

	mpc.PrintResults(result, circ.Outputs)
	return nil
}

func makeAddr(player int) string {
	return fmt.Sprintf("127.0.0.1:%d", 8080+player)
}
