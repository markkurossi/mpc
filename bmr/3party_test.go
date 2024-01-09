//
// Copyright (c) 2022-2024 Markku Rossi
//
// All rights reserved.
//

package bmr

import (
	"io"
	"testing"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/p2p"
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
			inR, inW := io.Pipe()
			outR, outW := io.Pipe()

			players[i].AddPeer(j, p2p.NewConn(&pipe{
				r: inR,
				w: outW,
			}))
			players[j].AddPeer(i, p2p.NewConn(&pipe{
				r: outR,
				w: inW,
			}))
		}
	}

	err = players[0].offlinePhase()
	if err != nil {
		t.Fatalf("offlinePhase: %v", err)
	}
}
