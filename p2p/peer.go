//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package p2p

import (
	"fmt"
	"sync"
)

// Peer implements a peer in the P2P network.
type Peer struct {
	m     sync.Mutex
	ID    int
	Addr  string
	Conns []*Conn
}

func (p *Peer) String() string {
	return fmt.Sprintf("%d[%v]", p.ID, p.Addr)
}

// Close closes all peer connections.
func (p *Peer) Close() error {
	p.m.Lock()
	defer p.m.Unlock()

	var err error

	for _, conn := range p.Conns {
		if e := conn.Close(); e != nil && err == nil {
			err = e
		}
	}
	return err
}

// SetConn assigns conn to connID.
func (p *Peer) SetConn(connID int, conn *Conn) error {
	p.m.Lock()
	defer p.m.Unlock()

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
