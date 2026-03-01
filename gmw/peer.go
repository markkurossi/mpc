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

// SendBitvec sends bit vector to the connection conn.
func (p *Peer) SendBitvec(conn *p2p.Conn, bits []uint64) error {
	var ld ot.LabelData
	var l ot.Label

	if err := conn.SendUint32(len(bits)); err != nil {
		return err
	}

	for i := 0; i < len(bits); i += 2 {
		l.D0 = bits[i]
		if i+1 < len(bits) {
			l.D1 = bits[i+1]
		}
		if err := conn.SendLabel(l, &ld); err != nil {
			return err
		}
	}

	return conn.Flush()
}

// ReceiveBitvec receives bits vector from the connection conn.
func (p *Peer) ReceiveBitvec(conn *p2p.Conn, bits []uint64) error {
	var ld ot.LabelData
	var l ot.Label

	count, err := conn.ReceiveUint32()
	if err != nil {
		return err
	}
	if count != len(bits) {
		return fmt.Errorf("bitvec length mismatch: expected %v, got %v",
			len(bits), count)
	}

	for i := 0; i < len(bits); i += 2 {
		if err := conn.ReceiveLabel(&l, &ld); err != nil {
			return err
		}
		bits[i] = l.D0
		if i+1 < len(bits) {
			bits[i+1] = l.D1
		}
	}
	return nil
}
