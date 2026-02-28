//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package gmw

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

const (
	// MagicOnline identifies the online connections: GMWn
	MagicOnline = 0x474d576e

	// MagicOffline identifies the offline connections: GMWf
	MagicOffline = 0x474d5766
)

func debugf(format string, a ...interface{}) {
	if false {
		fmt.Printf(format, a...)
	}
}

// Peer implements a peer in the P2P network.
type Peer struct {
	id      int
	addr    string
	online  *p2p.Conn
	offline *p2p.Conn
	input   *big.Int
	randBuf []byte
	shared  *big.Int
	otS     ot.OT
	otR     ot.OT

	iknpS *ot.IKNPSender
	iknpR *ot.IKNPReceiver
}

func (p *Peer) String() string {
	return fmt.Sprintf("%d[%v]", p.id, p.addr)
}

// Close closes the peer.
func (p *Peer) Close() {
	if p.online != nil {
		p.online.Close()
	}
	if p.offline != nil {
		p.offline.Close()
	}
}

// shareInput secret shares peer's input with the peer o.
func (p *Peer) shareInput(o *Peer) error {
	_, err := rand.Read(p.randBuf)
	if err != nil {
		return err
	}

	share := new(big.Int).SetBytes(p.randBuf)

	err = o.online.SendData(p.randBuf)
	if err != nil {
		return err
	}
	err = o.online.Flush()
	if err != nil {
		return err
	}
	p.shared.Xor(p.shared, share)

	return nil
}

func (p *Peer) otSend(self *Peer, a []uint) ([]uint, error) {
	n := len(a)
	wires := make([]ot.Wire, n)

	if err := p.otS.Send(wires); err != nil {
		return nil, err
	}

	corr := new(big.Int)
	share := make([]uint, n)

	for i := 0; i < n; i++ {
		r0 := wires[i].L0.Bit(0)
		r1 := wires[i].L1.Bit(0)
		corr.SetBit(corr, i, r0^r1^a[i])
		share[i] = r0
	}
	if err := p.online.SendData(corr.Bytes()); err != nil {
		return nil, err
	}
	if err := p.online.Flush(); err != nil {
		return nil, err
	}

	return share, nil
}

func (p *Peer) otReceive(self *Peer, b []uint) ([]uint, error) {
	n := len(b)
	flags := make([]bool, n)
	for idx, bit := range b {
		flags[idx] = bit == 1
	}

	labels := make([]ot.Label, n)
	if err := p.otR.Receive(flags, labels); err != nil {
		return nil, err
	}
	data, err := p.online.ReceiveData()
	if err != nil {
		return nil, err
	}

	corr := new(big.Int).SetBytes(data)
	share := make([]uint, n)

	for i := 0; i < n; i++ {
		t := labels[i].Bit(0)
		if flags[i] {
			share[i] = t ^ corr.Bit(i)
		} else {
			share[i] = t
		}
	}

	return share, nil
}

// SendBitsVec sends bits vector to the peer's offline channel.
func (p *Peer) SendBitsVec(bits []uint64) error {
	var ld ot.LabelData
	var l ot.Label

	for i := 0; i < len(bits); i += 2 {
		l.D0 = bits[i]
		if i+1 < len(bits) {
			l.D1 = bits[i+1]
		}
		if err := p.offline.SendLabel(l, &ld); err != nil {
			return err
		}
	}

	return p.offline.Flush()
}

// ReceiveBitsVec receives bits vector from the peer's offline channel.
func (p *Peer) ReceiveBitsVec(bits []uint64) error {
	var ld ot.LabelData
	var l ot.Label

	for i := 0; i < len(bits); i += 2 {
		if err := p.offline.ReceiveLabel(&l, &ld); err != nil {
			return err
		}
		bits[i] = l.D0
		if i+1 < len(bits) {
			bits[i+1] = l.D1
		}
	}
	return nil
}
