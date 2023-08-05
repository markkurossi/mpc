//
// garbler.go
//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

// Player runs the BMR protocol client on the P2P network.
func Player(nw *p2p.Network, circ *Circuit, inputs *big.Int, verbose bool) (
	[]*big.Int, error) {

	numPlayers := len(nw.Peers) + 1
	player := nw.ID

	timing := NewTiming()
	if verbose {
		fmt.Printf(" - Garbling...\n")
	}

	var key [32]byte
	_, err := rand.Read(key[:])
	if err != nil {
		return nil, err
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

	lu := new(big.Int)
	lv := new(big.Int)

	for g, gate := range circ.Gates {
		switch gate.Op {
		case XOR, XNOR:
		case INV:

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
				buf := make([]byte, len(circ.Gates)/8+1)
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
	luv := new(big.Int)
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

	ioStats := nw.Stats().Sum()
	timing.Sample("Fgc Step 1", []string{FileSize(ioStats).String()})

	// Step 2: generate XOR shares of Luvw
	if verbose {
		fmt.Printf(" - Step 2: generate XOR shares of Luvw\n")
	}

	// Init new gate values.
	Gs := NewGateValues(circ.NumGates, numPlayers, nw.ID)

	Ag := new(big.Int)
	Bg := new(big.Int)
	Cg := new(big.Int)

	for g, gate := range circ.Gates {
		switch gate.Op {
		case XOR, XNOR:
		case INV:

		default:
			tmp := luv.Bit(g) ^ garbled.Lambda(gate.Output)
			Ag.SetBit(Ag, g, tmp)
			if tmp != 0 {
				Gs.Ag[player][g].Xor(garbled.R)
				Gs.Dg[player][g].Xor(garbled.R)
			}

			Bg.SetBit(Bg, g, tmp^garbled.Lambda(gate.Input0))
			if tmp^garbled.Lambda(gate.Input0) != 0 {
				Gs.Bg[player][g].Xor(garbled.R)
				Gs.Dg[player][g].Xor(garbled.R)
			}

			Cg.SetBit(Cg, g, tmp^garbled.Lambda(gate.Input1))
			if tmp^garbled.Lambda(gate.Input1) != 0 {
				Gs.Cg[player][g].Xor(garbled.R)
				Gs.Dg[player][g].Xor(garbled.R)
			}

			Gs.Dg[player][g].Xor(garbled.R)
		}
	}

	X1LongAg := make([][]ot.Label, numPlayers)
	X1LongBg := make([][]ot.Label, numPlayers)
	X1LongCg := make([][]ot.Label, numPlayers)

	X2LongAg := make([][]ot.Label, numPlayers)
	X2LongBg := make([][]ot.Label, numPlayers)
	X2LongCg := make([][]ot.Label, numPlayers)

	for peerID := range nw.Peers {
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
				X1LongAg[peerID][g] = rand1
				Gs.Ag[player][g].Xor(rand1)

				rand2, err := ot.NewLabel(rand.Reader)
				if err != nil {
					return nil, err
				}
				X1LongBg[peerID][g] = rand2
				Gs.Bg[player][g].Xor(rand2)

				rand3, err := ot.NewLabel(rand.Reader)
				if err != nil {
					return nil, err
				}
				X1LongCg[peerID][g] = rand3
				Gs.Cg[player][g].Xor(rand3)

				Gs.Dg[player][g].Xor(rand1)
				Gs.Dg[player][g].Xor(rand2)
				Gs.Dg[player][g].Xor(rand3)

				X2LongAg[peerID][g] = garbled.R
				X2LongAg[peerID][g].Xor(rand1)

				X2LongBg[peerID][g] = garbled.R
				X2LongBg[peerID][g].Xor(rand2)

				X2LongCg[peerID][g] = garbled.R
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
			ra, rb, rc, err := peer.OTR(Ag, Bg, Cg,
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
				Gs.Ag[result.peerID][g].Xor(result.Ra[g])
				Gs.Bg[result.peerID][g].Xor(result.Rb[g])
				Gs.Cg[result.peerID][g].Xor(result.Rc[g])

				Gs.Dg[result.peerID][g].Xor(result.Ra[g])
				Gs.Dg[result.peerID][g].Xor(result.Rb[g])
				Gs.Dg[result.peerID][g].Xor(result.Rc[g])
			}
		}
	}

	xfer := nw.Stats().Sum() - ioStats
	ioStats = nw.Stats().Sum()
	timing.Sample("Fgc Step 3", []string{FileSize(xfer).String()})

	// Step 4: generate final secrets
	if verbose {
		fmt.Printf(" - Step 4: exchange gates\n")
	}

	// Output wire lambdas.
	Lo := new(big.Int)
	for w := 0; w < circ.Outputs.Size(); w++ {
		Lo.SetBit(Lo, w,
			garbled.Lambda(Wire(circ.NumWires-circ.Outputs.Size()+w)))
	}

	// Exchange gates with peers.

	gResultsC := make(chan GateResults)

	for peerID, peer := range nw.Peers {
		go func(peerID int, peer *p2p.Peer) {
			ra, rb, rc, rd, ro, err := peer.ExchangeGates(
				Gs.Ag, Gs.Bg, Gs.Cg, Gs.Dg, Lo)
			gResultsC <- GateResults{
				peerID: peerID,
				Ra:     ra,
				Rb:     rb,
				Rc:     rc,
				Rd:     rd,
				Ro:     ro,
				err:    err,
			}
		}(peerID, peer)
	}

	var gResults []GateResults
	for i := 0; i < len(nw.Peers); i++ {
		gResults = append(gResults, <-gResultsC)
	}
	for _, result := range gResults {
		if result.err != nil {
			return nil, fmt.Errorf("gate exchange with peer %d failed: %s",
				result.peerID, result.err)
		}
		for p := 0; p < numPlayers; p++ {
			for g, gate := range circ.Gates {
				switch gate.Op {
				case XOR, XNOR:
				case INV:

				default:
					Gs.Ag[p][g].Xor(result.Ra[p][g])
					Gs.Bg[p][g].Xor(result.Rb[p][g])
					Gs.Cg[p][g].Xor(result.Rc[p][g])
					Gs.Dg[p][g].Xor(result.Rd[p][g])
				}
			}
		}
		for w := 0; w < circ.Outputs.Size(); w++ {
			index := circ.NumWires - circ.Outputs.Size() + w
			// XOR peer lambda bit with our lambda bit i.e. only
			// result 1 changes our value.
			if result.Ro.Bit(index) == 1 {
				if garbled.Lambda(Wire(index)) == 0 {
					garbled.SetLambda(Wire(index), 1)
				} else {
					garbled.SetLambda(Wire(index), 0)
				}
			}
		}
	}

	for i := 0; i < numPlayers; i++ {
	gates:
		for g, gate := range circ.Gates {
			switch gate.Op {
			case XOR, XNOR:
			case INV:

			default:
				fmt.Printf("%d:%d: Ag: %s\n", i, g, Gs.Ag[i][g])
				fmt.Printf("%d:%d: Bg: %s\n", i, g, Gs.Bg[i][g])
				fmt.Printf("%d:%d: Cg: %s\n", i, g, Gs.Cg[i][g])
				fmt.Printf("%d:%d: Dg: %s\n", i, g, Gs.Dg[i][g])
				break gates
			}
		}
	}

	xfer = nw.Stats().Sum() - ioStats
	timing.Sample("Result", []string{FileSize(xfer).String()})
	if verbose {
		timing.Print(nw.Stats())
	}

	fmt.Printf("player not implemented yet\n")

	return []*big.Int{new(big.Int)}, nil
}

// OTLambdaResult contain oblivious transfer lambda results.
type OTLambdaResult struct {
	peerID int
	x1     *big.Int
	result *big.Int
	err    error
}

// OTRResult contain oblivious transfer results.
type OTRResult struct {
	peerID int
	Ra     []ot.Label
	Rb     []ot.Label
	Rc     []ot.Label
	err    error
}

// GateResults contain gate exchange results.
type GateResults struct {
	peerID int
	Ra     [][]ot.Label
	Rb     [][]ot.Label
	Rc     [][]ot.Label
	Rd     [][]ot.Label
	Ro     *big.Int
	err    error
}

// GateValues specify player's gate values.
type GateValues struct {
	Ag [][]ot.Label
	Bg [][]ot.Label
	Cg [][]ot.Label
	Dg [][]ot.Label
}

// NewGateValues creates GateValues for a player.
func NewGateValues(numGates, numPlayers, we int) *GateValues {
	v := &GateValues{
		Ag: make([][]ot.Label, numPlayers),
		Bg: make([][]ot.Label, numPlayers),
		Cg: make([][]ot.Label, numPlayers),
		Dg: make([][]ot.Label, numPlayers),
	}

	for p := 0; p < numPlayers; p++ {
		v.Ag[p] = arrayOfLabels(numGates)
		v.Bg[p] = arrayOfLabels(numGates)
		v.Cg[p] = arrayOfLabels(numGates)
		v.Dg[p] = arrayOfLabels(numGates)
	}
	return v
}

func arrayOfLabels(count int) []ot.Label {
	result := make([]ot.Label, count)
	for i := 0; i < count; i++ {
		l, err := ot.NewLabel(rand.Reader)
		if err != nil {
			panic(err)
		}
		result[i] = l
	}
	return result
}
