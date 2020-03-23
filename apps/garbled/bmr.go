//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/p2p"
)

func bmrMode(circ *circuit.Circuit, input []*big.Int, player int) error {
	// Create network.
	addr := makeAddr(player)
	nw, err := p2p.NewNetwork(addr, player)
	if err != nil {
		return err
	}
	defer nw.Close()

	numPlayers := len(circ.N1) + len(circ.N2)

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
	for {
		<-time.After(5 * time.Second)
		nw.Ping()
	}
}

func makeAddr(player int) string {
	return fmt.Sprintf("127.0.0.1:%d", 8080+player)
}
