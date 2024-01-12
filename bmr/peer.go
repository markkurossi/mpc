//
// Copyright (c) 2022-2024 Markku Rossi
//
// All rights reserved.
//

package bmr

import (
	"fmt"
	"io"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/text/superscript"
)

// Peer contains information about a protocol peer.
type Peer struct {
	this       *Player
	id         int
	to         ot.IO
	from       ot.IO
	otSender   ot.OT
	otReceiver ot.OT
}

// AddPeer adds a peer.
func (p *Player) AddPeer(idx int, to, from ot.IO) {
	p.peers[idx] = &Peer{
		this:       p,
		id:         idx,
		to:         to,
		from:       from,
		otSender:   ot.NewCO(),
		otReceiver: ot.NewCO(),
	}
}

func (p *Player) initPeers() error {
	for _, peer := range p.peers {
		if peer != nil {
			// Start consumer.
			go peer.consumer()

			// Init protocol.
			err := peer.to.SendByte(byte(OpInit))
			if err != nil {
				return err
			}
			err = peer.otSender.InitSender(peer.to)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (peer *Peer) consumer() {
	id := fmt.Sprintf("Player%s: consumer%s",
		superscript.Itoa(peer.this.id), superscript.Itoa(peer.id))
	err := peer.consumerMsgLoop(id)
	if err != nil {
		fmt.Printf("%s: %s\n", id, err)
	}
}

func (peer *Peer) consumerMsgLoop(id string) error {
	peer.this.Debugf("%s\n", id)
	for {
		v, err := peer.from.ReceiveByte()
		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}
		op := Operand(v)
		switch op {
		case OpInit:
			peer.this.Debugf("%s: %s\n", id, op)
			err = peer.otReceiver.InitReceiver(peer.from)
			if err != nil {
				return err
			}
		case OpFx:
			gid, err := peer.from.ReceiveUint32()
			if err != nil {
				return err
			}
			peer.this.Debugf("%s: %s: id=%v\n", id, op, gid)
			gate := peer.this.circ.Gates[gid]
			lv := peer.this.lambda.Bit(int(gate.Input1))

			xb, err := FxReceive(peer.otReceiver, lv)
			if err != nil {
				return err
			}
			peer.this.m.Lock()
			v := peer.this.luv.Bit(gid)
			v ^= xb
			peer.this.luv.SetBit(peer.this.luv, gid, v)
			peer.this.completions++
			if peer.this.completions == peer.this.circ.NumGates {
				peer.this.c.Signal()
			}
			peer.this.m.Unlock()

		default:
			return fmt.Errorf("%s: %s: not implemented\n", id, op)
		}

		peer.to.Flush()
	}
}
