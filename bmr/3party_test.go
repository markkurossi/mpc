//
// Copyright (c) 2022-2024 Markku Rossi
//
// All rights reserved.
//

package bmr

import (
	"testing"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/ot"
)

func Test3Party(t *testing.T) {
	circuit, err := circuit.Parse("testdata/3party.mpclc")
	if err != nil {
		t.Fatalf("could not load circuit: %s", err)
	}

	const n = 3
	var players []*Player

	for i := 0; i < n; i++ {
		p, err := NewPlayer(i, n)
		if err != nil {
			t.Fatalf("failed to create player: %v", err)
		}
		err = p.SetCircuit(circuit)
		if err != nil {
			t.Fatalf("failed to set circuit: %v", err)
		}
		players = append(players, p)
	}

	// Add peers to players.
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			this, other := ot.NewPipe()

			players[i].AddPeer(j, this)
			players[j].AddPeer(i, other)
		}
	}

	// Start other peers.
	for i := 1; i < n; i++ {
		go players[i].Play()
	}

	// Play player 0.
	players[0].Verbose = true
	err = players[0].Play()
	if err != nil {
		t.Fatalf("Play: %v", err)
	}
}
