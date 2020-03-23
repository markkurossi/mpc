//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package p2p

import (
	"log"
	"net"
	"sync"
	"time"
)

type Network struct {
	id       int
	m        sync.Mutex
	peers    map[int]*Peer
	addr     string
	listener net.Listener
}

func NewNetwork(addr string, id int) (*Network, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	nw := &Network{
		id:       id,
		peers:    make(map[int]*Peer),
		addr:     addr,
		listener: listener,
	}
	go nw.acceptLoop()
	return nw, nil
}

func (nw *Network) Close() error {
	return nw.listener.Close()
}

func (nw *Network) AddPeer(addr string, id int) error {
	// Try to connect to peer.
	for {
		log.Printf("NW %d: Connecting to %s...\n", nw.id, addr)
		nc, err := net.Dial("tcp", addr)
		if err != nil {
			delay := 5 * time.Second
			log.Printf("NW %d: Connect to %s failed, retrying in %s\n",
				nw.id, addr, delay)
			<-time.After(delay)
			continue
		}
		log.Printf("NW %d: Connected to %s\n", nw.id, addr)
		conn := NewConn(nc)
		defer conn.Close()

		if err := conn.SendUint32(nw.id); err != nil {
			return err
		}
		if err := conn.Flush(); err != nil {
			return err
		}
		return nw.newPeer(conn, id)
	}
}

func (nw *Network) Ping() {
	for _, peer := range nw.peers {
		peer.Ping()
	}
}

func (nw *Network) acceptLoop() {
	for {
		nc, err := nw.listener.Accept()
		if err != nil {
			log.Printf("NW %d: accept failed: %s\n", nw.id, err)
			break
		}
		conn := NewConn(nc)

		// Read peer ID.
		id, err := conn.ReceiveUint32()
		if err != nil {
			log.Printf("NW %d: I/O error: %s\n", nw.id, err)
			conn.Close()
			continue
		}

		err = nw.newPeer(conn, id)
		if err != nil {
			log.Printf("inbound connection error: %s\n", err)
		}
	}
}

func (nw *Network) newPeer(conn *Conn, id int) error {
	nw.m.Lock()
	peer, ok := nw.peers[id]
	if ok {
		nw.m.Unlock()
		log.Printf("NW %d: peer %d already connected\n", nw.id, id)
		return conn.Close()
	}
	peer = &Peer{
		id:   id,
		conn: conn,
	}
	nw.peers[id] = peer
	nw.m.Unlock()

	go peer.msgLoop()

	return nil
}

type Peer struct {
	id   int
	conn *Conn
}

func (peer *Peer) Close() error {
	return peer.conn.Close()
}

func (peer *Peer) Ping() error {
	if err := peer.conn.SendUint32(0xffffffff); err != nil {
		return err
	}
	return peer.conn.Flush()
}

func (peer *Peer) msgLoop() {
	var done bool

	for !done {
		op, err := peer.conn.ReceiveUint32()
		if err != nil {
			log.Printf("%s\n", err)
			done = true
			continue
		}
		switch op {
		case 0xffffffff:
			log.Printf("Peer %d: PING\n", peer.id)

		default:
			log.Printf("Peer %d: unknown message %d\n", peer.id, op)
		}
	}
	peer.Close()
}
