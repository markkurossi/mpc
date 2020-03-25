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

	"github.com/markkurossi/mpc/p2p"
)

// XXX check this
var key [32]byte

func Player(nw *p2p.Network, circ *Circuit, inputs []*big.Int, verbose bool) (
	[]*big.Int, error) {

	timing := NewTiming()
	if verbose {
		fmt.Printf(" - Garbling...\n")
	}

	garbled, err := circ.Garble(key[:])
	if err != nil {
		return nil, err
	}

	timing.Sample("Garble", nil)

	if verbose {
		fmt.Printf(" - Sending garbled circuit...\n")
	}

	// Compute LambdaUV for each non-XOR gate.

	luv := new(big.Int)
	lu := new(big.Int)
	lv := new(big.Int)

	for i, gate := range circ.Gates {
		switch gate.Op {
		case XOR, XNOR:

		case INV:
			if garbled.Wires[int(gate.Input0)].L0.S() {
				lu.SetBit(lu, i, 1)
			}

		default:
			if garbled.Wires[int(gate.Input0)].L0.S() {
				lu.SetBit(lu, i, 1)
			}
			if garbled.Wires[int(gate.Input1)].L0.S() {
				lv.SetBit(lv, i, 1)
			}
		}
	}

	// Compute lu AND lv.
	luv.And(lu, lv)

	// Oblivious transfer with peers.

	results := make(chan OTResult)

	for id, peer := range nw.Peers {
		go func(id int, peer *p2p.Peer) {
			x1, result, err := func(id int, peer *p2p.Peer) (
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

				result, err := peer.OT(len(circ.Gates), lu, x1, x2)
				if err != nil {
					return nil, nil, err
				}
				return x1, result, nil
			}(id, peer)
			results <- OTResult{
				id:     id,
				x1:     x1,
				result: result,
				err:    err,
			}
		}(id, peer)
	}

	for i := 0; i < len(nw.Peers); i++ {
		result := <-results
		if result.err != nil {
			return nil, fmt.Errorf("OT with peer %d failed: %s",
				result.id, result.err)
		}
		luv.Xor(luv, result.x1)
		luv.Xor(luv, result.result)
	}

	fmt.Printf("luv: %s\n", luv.Text(2))

	for false {
		<-time.After(5 * time.Second)
	}
	return nil, fmt.Errorf("player not implemented yet")
}

type OTResult struct {
	id     int
	x1     *big.Int
	result *big.Int
	err    error
}
