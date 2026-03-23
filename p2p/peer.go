//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package p2p

import (
	"fmt"
)

// Peer implements a peer in the P2P network.
type Peer struct {
	ID    int
	Addr  string
	Conns []*Conn
}

func (p *Peer) String() string {
	return fmt.Sprintf("%d[%v]", p.ID, p.Addr)
}

func (p *Peer) Close() {
	for _, conn := range p.Conns {
		conn.Close()
	}
}

func (p *Peer) SetConn(connID int, conn *Conn) error {
	if len(p.Conns) <= connID {
		n := make([]*Conn, connID+1)
		copy(n, p.Conns)
		p.Conns = n
	}
	if p.Conns[connID] != nil {
		return fmt.Errorf("%v: connection %v already set", p, connID)
	}
	p.Conns[connID] = conn

	return nil
}
