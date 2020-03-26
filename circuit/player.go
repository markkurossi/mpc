//
// garbler.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

// XXX check this
var key [32]byte

func Player(nw *p2p.Network, circ *Circuit, inputs []*big.Int, verbose bool) (
	[]*big.Int, error) {

	numPlayers := len(nw.Peers) + 1
	player := nw.ID

	timing := NewTiming()
	if verbose {
		fmt.Printf(" - Garbling...\n")
	}

	garbled, err := circ.Garble(key[:])
	if err != nil {
		return nil, err
	}

	timing.Sample("Garble", nil)

	// The protocol for Fgc (Protocol 3.1)

	// Step 1: Compute Lambda{u,v} for each non-XOR gate.
	if verbose {
		fmt.Printf(" - Step 1: compute Luv\n")
	}

	luv := new(big.Int)
	lu := new(big.Int)
	lv := new(big.Int)

	for g, gate := range circ.Gates {
		switch gate.Op {
		case XOR, XNOR:

		case INV:
			lu.SetBit(lu, g, garbled.Lambda(gate.Input0))

		default:
			lu.SetBit(lu, g, garbled.Lambda(gate.Input0))
			lv.SetBit(lv, g, garbled.Lambda(gate.Input1))
		}
	}

	// OTs with peers.

	lambdaResults := make(chan OTLambdaResult)

	for peerID, peer := range nw.Peers {
		go func(peerID int, peer *p2p.Peer) {
			x1, result, err := func(peer *p2p.Peer) (
				*big.Int, *big.Int, error) {

				// Random X1
				buf := make([]byte, luv.BitLen()/8+1)
				_, err := rand.Read(buf[:])
				if err != nil {
					return nil, nil, err
				}
				x1 := new(big.Int).SetBytes(buf)
				shift := len(buf)*8 - len(circ.Gates)
				x1.Rsh(x1, uint(shift))

				// X2.
				x2 := new(big.Int).Xor(lu, x1)

				result, err := peer.OTLambda(len(circ.Gates), lv, x1, x2)
				if err != nil {
					return nil, nil, err
				}
				return x1, result, nil
			}(peer)
			lambdaResults <- OTLambdaResult{
				peerID: peerID,
				x1:     x1,
				result: result,
				err:    err,
			}
		}(peerID, peer)
	}

	// Compute lu AND lv.
	luv.And(lu, lv)

	for i := 0; i < len(nw.Peers); i++ {
		result := <-lambdaResults
		if result.err != nil {
			return nil, fmt.Errorf("OT-Lambda with peer %d failed: %s",
				result.peerID, result.err)
		}
		luv.Xor(luv, result.x1)
		luv.Xor(luv, result.result)
	}

	fmt.Printf("Luv: %s\n", luv.Text(2))
	timing.Sample("Fgc Step 1", nil)

	// Step 2: generate XOR shares of Luvw
	if verbose {
		fmt.Printf(" - Step 2: generate XOR shares of Luvw\n")
	}

	// Init new gate values.
	Gs := make([]*GateValues, circ.NumGates)
	for i, gate := range circ.Gates {
		switch gate.Op {
		case XOR, XNOR:
		case INV:

		default:
			Gs[i] = NewGateValues(numPlayers)
		}
	}

	Ag := new(big.Int)
	Bg := new(big.Int)
	Cg := new(big.Int)

	for g, gate := range circ.Gates {
		switch gate.Op {
		case XOR, XNOR:

		case INV:

		default:
			tmp := garbled.Lambda(gate.Output)
			Ag.SetBit(Ag, g, 1)
			if tmp != 0 {
				Gs[g].Ag[player].Xor(garbled.R)
				Gs[g].Dg[player].Xor(garbled.R)
			}

			Bg.SetBit(Bg, g, tmp^garbled.Lambda(gate.Input0))
			if tmp^garbled.Lambda(gate.Input0) != 0 {
				Gs[g].Bg[player].Xor(garbled.R)
				Gs[g].Dg[player].Xor(garbled.R)
			}

			Cg.SetBit(Cg, g, tmp^garbled.Lambda(gate.Input1))
			if tmp^garbled.Lambda(gate.Input1) != 0 {
				Gs[g].Cg[player].Xor(garbled.R)
				Gs[g].Dg[player].Xor(garbled.R)
			}
		}
	}

	X1LongAg := make([][]ot.Label, numPlayers)
	X1LongBg := make([][]ot.Label, numPlayers)
	X1LongCg := make([][]ot.Label, numPlayers)

	X2LongAg := make([][]ot.Label, numPlayers)
	X2LongBg := make([][]ot.Label, numPlayers)
	X2LongCg := make([][]ot.Label, numPlayers)

	for peerID, _ := range nw.Peers {
		X1LongAg[peerID] = make([]ot.Label, len(circ.Gates))
		X1LongBg[peerID] = make([]ot.Label, len(circ.Gates))
		X1LongCg[peerID] = make([]ot.Label, len(circ.Gates))

		X2LongAg[peerID] = make([]ot.Label, len(circ.Gates))
		X2LongBg[peerID] = make([]ot.Label, len(circ.Gates))
		X2LongCg[peerID] = make([]ot.Label, len(circ.Gates))

		for g, gate := range circ.Gates {
			switch gate.Op {
			case XOR, XNOR:
			case INV:

			default:
				rand1, err := ot.NewLabel(rand.Reader)
				if err != nil {
					return nil, err
				}
				X1LongAg[peerID][g] = *rand1
				Gs[g].Ag[peerID].Xor(rand1)

				rand2, err := ot.NewLabel(rand.Reader)
				if err != nil {
					return nil, err
				}
				X1LongBg[peerID][g] = *rand2
				Gs[g].Bg[peerID].Xor(rand2)

				rand3, err := ot.NewLabel(rand.Reader)
				if err != nil {
					return nil, err
				}
				X1LongCg[peerID][g] = *rand3
				Gs[g].Cg[peerID].Xor(rand3)

				Gs[g].Cg[peerID].Xor(rand1)
				Gs[g].Cg[peerID].Xor(rand2)
				Gs[g].Cg[peerID].Xor(rand3)

				X2LongAg[peerID][g].Xor(rand1)
				X2LongBg[peerID][g].Xor(rand2)
				X2LongCg[peerID][g].Xor(rand3)
			}
		}
	}

	timing.Sample("Fgc Step 2", nil)

	// Step 3: generate XOR shares of Rj
	if verbose {
		fmt.Printf(" - Step 3: generate XOR shares or Rj\n")
	}

	// OTs with peers.

	rResults := make(chan OTRResult)

	for peerID, peer := range nw.Peers {
		go func(peerID int, peer *p2p.Peer) {
			ra, rb, rc, err := peer.OTR(Ag,
				X1LongAg[peerID], X2LongAg[peerID],
				X1LongBg[peerID], X2LongBg[peerID],
				X1LongCg[peerID], X2LongCg[peerID])

			rResults <- OTRResult{
				peerID: peerID,
				Ra:     ra,
				Rb:     rb,
				Rc:     rc,
				err:    err,
			}
		}(peerID, peer)
	}

	for i := 0; i < len(nw.Peers); i++ {
		result := <-rResults
		if result.err != nil {
			return nil, fmt.Errorf("OT-R with peer %d failed: %s",
				result.peerID, result.err)
		}
		for g, gate := range circ.Gates {
			switch gate.Op {
			case XOR, XNOR:
			case INV:

			default:
				Gs[g].Ag[result.peerID].Xor(&result.Ra[g])
				Gs[g].Bg[result.peerID].Xor(&result.Rb[g])
				Gs[g].Cg[result.peerID].Xor(&result.Rc[g])

				Gs[g].Dg[result.peerID].Xor(&result.Ra[g])
				Gs[g].Dg[result.peerID].Xor(&result.Rb[g])
				Gs[g].Dg[result.peerID].Xor(&result.Rc[g])
			}
		}
	}

	for false {
		<-time.After(5 * time.Second)
	}
	return nil, fmt.Errorf("player not implemented yet")
}

type OTLambdaResult struct {
	peerID int
	x1     *big.Int
	result *big.Int
	err    error
}

type OTRResult struct {
	peerID int
	Ra     []ot.Label
	Rb     []ot.Label
	Rc     []ot.Label
	err    error
}

type GateValues struct {
	Ag []ot.Label
	Bg []ot.Label
	Cg []ot.Label
	Dg []ot.Label
}

func NewGateValues(numPlayers int) *GateValues {
	return &GateValues{
		Ag: arrayOfLabels(numPlayers),
		Bg: arrayOfLabels(numPlayers),
		Cg: arrayOfLabels(numPlayers),
		Dg: arrayOfLabels(numPlayers),
	}
}

func arrayOfLabels(count int) []ot.Label {
	result := make([]ot.Label, count)
	for i := 0; i < count; i++ {
		l, err := ot.NewLabel(rand.Reader)
		if err != nil {
			panic(err)
		}
		result[i] = *l
	}
	return result
}
