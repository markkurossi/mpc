//
// Copyright (c) 2022-2024 Markku Rossi
//
// All rights reserved.
//

package bmr

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/text/superscript"
	"github.com/markkurossi/text/symbols"
)

const (
	// Security parameter k specifies the label sizes in bits.
	k = 32
)

// Player implements a multi-party player.
type Player struct {
	Verbose    bool
	id         int
	numPlayers int
	r          Label
	peers      []*Peer
	circ       *circuit.Circuit
	lambda     *big.Int

	// Everything below is synchronized with m.
	m           *sync.Mutex
	c           *sync.Cond
	completions int
	luv         *big.Int

	// The XOR shares luvw{0,1,2,3} matching λuvw, λuv̄w, λūvw, λūv̄w.
	luvw0 *big.Int
	luvw1 *big.Int
	luvw2 *big.Int
	luvw3 *big.Int

	// The XOR shares of Rj matching ρij,α,β
	rj [][]Label
}

// NewPlayer creates a new multi-party player.
func NewPlayer(id, numPlayers int) (*Player, error) {
	m := new(sync.Mutex)
	return &Player{
		id:         id,
		numPlayers: numPlayers,
		peers:      make([]*Peer, numPlayers),
		m:          m,
		c:          sync.NewCond(m),
		luv:        big.NewInt(0),
		luvw0:      big.NewInt(0),
		luvw1:      big.NewInt(0),
		luvw2:      big.NewInt(0),
		luvw3:      big.NewInt(0),
	}, nil
}

// Debugf prints debugging message if Verbose debugging is enabled for
// this Player.
func (p *Player) Debugf(format string, a ...interface{}) {
	if !p.Verbose {
		return
	}
	fmt.Printf(format, a...)
}

// IDString returns the player ID as string.
func (p *Player) IDString() string {
	return superscript.Itoa(p.id)
}

// SetCircuit sets the circuit that is evaluated.
func (p *Player) SetCircuit(c *circuit.Circuit) error {
	if len(c.Inputs) != p.numPlayers {
		return fmt.Errorf("invalid circuit: #inputs=%d != #players=%d",
			len(c.Inputs), p.numPlayers)
	}
	p.circ = c
	return nil
}

// Play runs the protocol with the peers.
func (p *Player) Play() error {
	var count int
	for _, peer := range p.peers {
		if peer != nil {
			count++
		}
	}
	if count != p.numPlayers-1 {
		return fmt.Errorf("invalid number of peers: expected %d, got %d",
			count, p.numPlayers-1)
	}

	// Init circuit-dependent fields.
	p.rj = make([][]Label, p.circ.NumGates)
	for i := 0; i < p.circ.NumGates; i++ {
		switch p.circ.Gates[i].Op {
		case circuit.AND:
			p.rj[i] = make([]Label, 4)
		default:
			return fmt.Errorf("gate %v not implemented yet", p.circ.Gates[i].Op)
		}
	}

	p.Debugf("BMR: #gates=%v\n", p.circ.NumGates)

	p.Debugf("Offline Phase...\n")
	err := p.offlinePhase()
	if err != nil {
		return err
	}

	// Start peers.
	err = p.initPeers()
	if err != nil {
		return err
	}

	p.Debugf("Online Phase...\n")
	err = p.fgc()
	if err != nil {
		return err
	}

	return nil
}

// offlinePhase implements the BMR Offline Phase (BMR Figure 2 - Page 6).
func (p *Player) offlinePhase() error {
	// Step 1: each peer chooses a random key offset R^i.
	r, err := NewLabel()
	if err != nil {
		return err
	}
	p.r = r
	p.Debugf("R%s:\t%v\n", p.IDString(), p.r)

	// Step 2.a: create random permutation bits lambda. We set the
	// bits initially for all wires but later reset the output bits of
	// XOR gates.
	p.lambda, err = rand.Int(rand.Reader,
		big.NewInt(int64((1<<p.circ.NumWires)-1)))
	if err != nil {
		return err
	}

	// Optimization for Step 6: set input wire lambdas to 0 for other
	// peers' inputs.
	var inputIndex int
	for id, input := range p.circ.Inputs {
		if id != p.id {
			for i := 0; i < int(input.Type.Bits); i++ {
				p.lambda.SetBit(p.lambda, inputIndex+i, 0)
			}
		}
		inputIndex += int(input.Type.Bits)
	}

	wires := make([]Wire, p.circ.NumWires)

	// Step 2: create label shares for all wires. We will reset the
	// output labels of XOR gates below.
	for i := 0; i < p.circ.NumWires; i++ {
		// 2.b: choose 0-garbled label at random.
		wires[i].L0, err = NewLabel()
		if err != nil {
			return err
		}
		// 2.c: set the 1-garbled label to be: k_{w,1} = k_{w,0} ⊕ R^i
		wires[i].L1 = wires[i].L0
		wires[i].L1.Xor(p.r)
	}

	if false {
		for i := 0; i < len(wires); i++ {
			p.Debugf("W%d:\t%v\n", i, wires[i])
		}
	}
	p.Debugf("%c%s:\t%v\n", symbols.Lambda, p.IDString(),
		lambda(p.lambda, len(wires)))

	// Step 3: patch output wires and permutation bits for XOR output
	// wires.
	for i := 0; i < p.circ.NumGates; i++ {
		gate := p.circ.Gates[i]
		if gate.Op != circuit.XOR {
			continue
		}
		u := int(gate.Input0)
		v := int(gate.Input1)
		w := int(gate.Output)

		// 3.a: set permutation bit: λ_w = λ_u ⊕ λ_v

		lu := p.lambda.Bit(u)
		lv := p.lambda.Bit(v)

		lo := lu ^ lv
		p.lambda.SetBit(p.lambda, w, lo)

		p.Debugf("%c[%d]: %v ^ %v = %v\n", symbols.Lambda, w, lu, lv, lo)

		// 3.b: set garbled label on wire 0: k_{w,0} = k_{u,0} ⊕ k_{v,0}
		wires[w].L0 = wires[u].L0
		wires[w].L0.Xor(wires[v].L0)

		// 3.b: set garbled label on wire 1: k_{w,1} = k_{w,0} ⊕ R^i
		wires[w].L1 = wires[w].L0
		wires[w].L1.Xor(p.r)
	}

	for i := 0; i < len(wires); i++ {
		p.Debugf("W%d:\t%v\n", i, wires[i])
	}

	p.Debugf("%c%s:\t%v\n", symbols.Lambda, p.IDString(),
		lambda(p.lambda, len(wires)))

	return nil
}

func (p *Player) syncBarrier(nth int) int {
	return p.circ.NumGates * (len(p.peers) - 1) * nth
}

// fgc computes the multiparty garbled circuit (3.1.2 The Protocol for
// Fgc - Page 7).
func (p *Player) fgc() (err error) {

	// Step 1.

	luv := big.NewInt(0)
	for i := 0; i < p.circ.NumGates; i++ {
		gate := p.circ.Gates[i]
		switch gate.Op {
		case circuit.AND:
			lu := p.lambda.Bit(int(gate.Input0))
			lv := p.lambda.Bit(int(gate.Input1))

			// Step 1: securely compute XOR shares of l_uv
			uv := lu * lv

			// For i!=j, run Fx(l_ui,l_vj)
			for _, peer := range p.peers {
				if peer == nil {
					continue
				}
				err = peer.to.SendByte(byte(OpFxLambda))
				if err != nil {
					return err
				}
				err = peer.to.SendUint32(i)
				if err != nil {
					return err
				}
				r, err := FxSend(peer.otSender, lu)
				if err != nil {
					return err
				}
				uv ^= r
			}
			luv.SetBit(luv, i, uv)

		default:
			return fmt.Errorf("gate %v not implemented yet", gate.Op)
		}
	}

	p.m.Lock()
	p.luv.Xor(p.luv, luv)
	for p.completions < p.syncBarrier(1) {
		p.c.Wait()
	}
	p.m.Unlock()

	fmt.Printf("Player%s: %cuv =%v\n", p.IDString(), symbols.Lambda,
		lambda(p.luv, p.circ.NumGates))

	// Step 2.
	for i := 0; i < p.circ.NumGates; i++ {
		gate := p.circ.Gates[i]
		switch gate.Op {
		case circuit.AND:
			lu := p.lambda.Bit(int(gate.Input0))
			lv := p.lambda.Bit(int(gate.Input1))
			lw := p.lambda.Bit(int(gate.Output))

			luv := p.luv.Bit(i)

			// λuvw = λuv ⊕ λw
			p.luvw0.SetBit(p.luvw0, i, luv^lw)

			// λuv̄w = λuv ⊕ λu ⊕ λw
			p.luvw1.SetBit(p.luvw1, i, luv^lu&lw)

			// λūvw = λuv ⊕ λv ⊕ λw
			p.luvw2.SetBit(p.luvw2, i, luv^lv&lw)

			// λūv̄w = λuv ⊕ λu ⊕ λv ⊕ λw
			p.luvw3.SetBit(p.luvw3, i, luv^lu^lv&lw)

		default:
			return fmt.Errorf("gate %v not implemented yet", gate.Op)
		}
	}
	fmt.Printf("Player%s: %cuvw=%v\n", p.IDString(), symbols.Lambda,
		lambda(p.luvw0, p.circ.NumGates))
	fmt.Printf("Player%s: %cuv̄w=%v\n", p.IDString(), symbols.Lambda,
		lambda(p.luvw1, p.circ.NumGates))
	fmt.Printf("Player%s: %cūvw=%v\n", p.IDString(), symbols.Lambda,
		lambda(p.luvw2, p.circ.NumGates))
	fmt.Printf("Player%s: %cūv̄w=%v\n", p.IDString(), symbols.Lambda,
		lambda(p.luvw3, p.circ.NumGates))

	// Step 3: for i!=j, run Fxk(R,luvw)
	for gid := 0; gid < p.circ.NumGates; gid++ {
		for _, peer := range p.peers {
			if peer == nil {
				continue
			}
			err = peer.to.SendByte(byte(OpFxR))
			if err != nil {
				return err
			}
			err = peer.to.SendUint32(gid)
			if err != nil {
				return err
			}
			for n := 0; n < len(p.rj[gid]); n++ {
				r, err := FxkSend(peer.otSender, p.r)
				if err != nil {
					return err
				}
				p.m.Lock()
				p.rj[gid][n].Xor(r)
				p.m.Unlock()
			}
		}
	}
	p.m.Lock()
	for p.completions < p.syncBarrier(2) {
		p.c.Wait()
	}
	p.m.Unlock()

	for gid := 0; gid < p.circ.NumGates; gid++ {
		p.Debugf("Player%s: rj[%v]:\t", p.IDString(), gid)
		for idx, l := range p.rj[gid] {
			if idx > 0 {
				p.Debugf(" ")
			}
			p.Debugf("%v", l)
		}
		p.Debugf("\n")
	}

	return nil
}

func lambda(v *big.Int, w int) string {
	str := v.Text(2)
	for len(str) < w {
		str = "0" + str
	}
	return str
}
