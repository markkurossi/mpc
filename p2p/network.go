//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package p2p

import (
	"fmt"
	"net"
	"sort"
	"sync"
)

const (
	connMagic     = 0x474d5700
	connMagicMask = 0xffffff00
)

// Network implements P2P network.
type Network struct {
	m             sync.Mutex
	c             *sync.Cond
	NumParties    int
	listener      net.Listener
	listenerDone  bool
	listenerError error
	Peers         []*Peer
	need          []int
	peersByID     map[int]*Peer
	Self          *Peer
	Done          chan error
}

func newNetwork(numParties, numConns int, listener net.Listener,
	self *Peer) *Network {

	nw := &Network{
		NumParties: numParties,
		listener:   listener,
		peersByID:  make(map[int]*Peer),
		Self:       self,
		need:       make([]int, numConns),
		Done:       make(chan error),
	}
	nw.c = sync.NewCond(&nw.m)

	if err := nw.addPeer(self); err != nil {
		panic(err)
	}

	return nw
}

// Create initializes a P2P network for the given leader address.  The
// numParties argument specifies the number of parties in the network.
// The numConns argument specifies how many channels are opened
// between parties.
func Create(addr string, numParties, numConns int) (*Network, error) {
	if numParties < 2 {
		return nil, fmt.Errorf("invalid number of parties: %v", numParties)
	}
	if numConns < 1 {
		return nil, fmt.Errorf("invalid number of connections: %v", numConns)
	}

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return newNetwork(numParties, numConns, l, &Peer{
		Addr: addr,
	}), nil
}

// Join connects to a P2P network managed by the leader.  The self
// argument specifies this party's connection address.  The id
// argument specifies the party's ID in the network.  The numConns
// argument specifies how many channels are opened between parties.
func Join(leader, self string, id, numConns int) (*Network, error) {
	if id == 0 {
		return nil, fmt.Errorf("invalid ID %v", id)
	}
	if numConns < 1 {
		return nil, fmt.Errorf("invalid number of connections: %v", numConns)
	}
	l, err := net.Listen("tcp", self)
	if err != nil {
		return nil, err
	}
	nw := newNetwork(id+1, numConns, l, &Peer{
		ID:   id,
		Addr: self,
	})

	fmt.Printf("%v: connecting to leader %v\n", nw.Self, leader)

	c, err := net.Dial("tcp", leader)
	if err != nil {
		nw.Close()
		return nil, err
	}

	err = nw.addPeer(&Peer{
		ID:    0,
		Addr:  leader,
		Conns: []*Conn{NewConn(c)},
	})
	if err != nil {
		nw.Close()
		return nil, err
	}

	return nw, nil
}

// Close closes the P2P network.
func (nw *Network) Close() error {
	var err error

	nw.m.Lock()
	defer nw.m.Unlock()

	for _, p := range nw.Peers {
		p.Close()
	}
	if nw.listener != nil {
		nw.listener.Close()
	}

	return err
}

// Connect connects the p2p network.
func (nw *Network) Connect() error {
	if nw.Self.ID == 0 {
		// Init leader's accept loop.
		nw.m.Lock()
		for i := 0; i < len(nw.need); i++ {
			nw.need[i] = nw.NumParties - 1
		}
		go nw.accept()
		nw.m.Unlock()
	}

	for i := 0; i < len(nw.need); i++ {
		err := nw.connect(i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (nw *Network) connect(connID int) error {
	if nw.Self.ID == 0 {
		return nw.connectLeader(connID)
	}
	return nw.connectPeer(connID)
}

func (nw *Network) connectLeader(connID int) error {
	// Accept all connections for the connID.

	var success bool

	nw.m.Lock()
	for nw.need[connID] > 0 && !nw.listenerDone {
		nw.c.Wait()
	}
	success = (nw.need[connID] == 0 && !nw.listenerDone)
	nw.m.Unlock()

	if !success {
		return nw.listenerError
	}

	if connID > 0 {
		return nil
	}

	// First connection, send network info to all peers.
	for _, peer := range nw.Peers {
		if peer.ID == nw.Self.ID {
			continue
		}
		// Number of connections.
		if err := peer.Conns[0].SendUint32(len(nw.need)); err != nil {
			return err
		}
		// Number of peers - 2 (this+peer).
		err := peer.Conns[0].SendUint32(len(nw.Peers) - 2)
		if err != nil {
			return err
		}
		for _, i := range nw.Peers {
			if i.ID == nw.Self.ID || i.ID == peer.ID {
				continue
			}
			err = peer.Conns[0].SendUint32(i.ID)
			if err != nil {
				return err
			}
			err = peer.Conns[0].SendString(i.Addr)
			if err != nil {
				return err
			}
		}
		err = peer.Conns[0].Flush()
		if err != nil {
			return err
		}
	}

	return nil
}

func (nw *Network) connectPeer(connID int) error {
	self := nw.Self

	leader, err := nw.getPeer(0)
	if err != nil {
		return err
	}

	if connID == 0 {
		// The first connection, sync network information with the
		// leader.
		if err := nw.connectPeerToLeader(leader); err != nil {
			return err
		}
		go nw.accept()
	}

	// Connect network for connection connID.
	for _, peer := range nw.Peers {
		if peer.ID == 0 {
			if connID == 0 {
				continue
			}
		} else if peer.ID <= self.ID {
			continue
		}
		err = nw.dial(peer, connID)
		if err != nil {
			return err
		}
	}

	// Wait until all peers have been connected
	var success bool
	nw.m.Lock()
	for nw.need[connID] > 0 && !nw.listenerDone {
		nw.c.Wait()
	}
	success = (nw.need[connID] == 0 && !nw.listenerDone)
	nw.m.Unlock()

	if !success {
		return nw.listenerError
	}

	return nil
}

func (nw *Network) connectPeerToLeader(leader *Peer) error {
	self := nw.Self

	if err := leader.Conns[0].SendUint32(connMagic); err != nil {
		return err
	}
	if err := leader.Conns[0].SendUint32(self.ID); err != nil {
		return err
	}
	if err := leader.Conns[0].SendString(self.Addr); err != nil {
		return err
	}
	if err := leader.Conns[0].Flush(); err != nil {
		return err
	}

	// Get network information.
	numConns, err := leader.Conns[0].ReceiveUint32()
	if err != nil {
		return err
	}
	if numConns != len(nw.need) {
		return fmt.Errorf("invalid number of connections %v: expected %v",
			numConns, len(nw.need))
	}
	n, err := leader.Conns[0].ReceiveUint32()
	if err != nil {
		return err
	}
	nw.NumParties = 2 + n

	var numAccept int
	for i := 0; i < n; i++ {
		id, err := leader.Conns[0].ReceiveUint32()
		if err != nil {
			return err
		}
		addr, err := leader.Conns[0].ReceiveString()
		if err != nil {
			return err
		}
		peer := &Peer{
			ID:   id,
			Addr: addr,
		}
		err = nw.addPeer(peer)
		if err != nil {
			return err
		}
		if self.ID > id {
			numAccept++
		}
	}

	// How many peers we accept per connection.
	nw.m.Lock()
	defer nw.m.Unlock()
	for i := range nw.need {
		nw.need[i] = numAccept
	}

	return nil
}

func (nw *Network) dial(peer *Peer, connID int) error {
	self := nw.Self

	fmt.Printf("%v: dial(%v, %v)\n", self, peer, connID)

	c, err := net.Dial("tcp", peer.Addr)
	if err != nil {
		return err
	}
	conn := NewConn(c)
	magic := connMagic | connID
	if err := conn.SendUint32(magic); err != nil {
		return err
	}
	if err := conn.SendUint32(self.ID); err != nil {
		return err
	}
	if err := conn.SendString(self.Addr); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	peer.SetConn(connID, conn)

	return nil
}

func (nw *Network) accept() {
	err := nw.acceptLoop()

	nw.m.Lock()
	defer nw.m.Unlock()

	nw.listenerDone = true
	nw.listenerError = err
	nw.c.Broadcast()
}

func (nw *Network) acceptLoop() error {
	for {
		c, err := nw.listener.Accept()
		if err != nil {
			return err
		}
		conn := NewConn(c)
		err = nw.acceptConn(conn)
		if err != nil {
			conn.Close()
			return err
		}
	}
}

func (nw *Network) acceptConn(conn *Conn) error {
	magic, err := conn.ReceiveUint32()
	if err != nil {
		return err
	}
	id, err := conn.ReceiveUint32()
	if err != nil {
		return err
	}
	addr, err := conn.ReceiveString()
	if err != nil {
		return err
	}

	if magic&connMagicMask != connMagic {
		return fmt.Errorf("invalid connection magic %x from peer %v", magic, id)
	}
	connID := int(byte(magic))
	if connID >= len(nw.need) {
		return fmt.Errorf("invalid connection ID %v from peer %v", connID, id)
	}

	nw.m.Lock()

	if nw.need[connID] == 0 {
		nw.m.Unlock()
		return fmt.Errorf("%v: too many connections for ID %v from peer %v: magic=%x",
			nw.Self, connID, id, magic)
	}
	nw.need[connID]--
	nw.c.Broadcast()
	nw.m.Unlock()

	peer := &Peer{
		ID:   id,
		Addr: addr,
	}
	peer.SetConn(connID, conn)

	return nw.addPeer(peer)
}

func (nw *Network) addPeer(peer *Peer) error {
	if peer.ID >= nw.NumParties {
		return fmt.Errorf("invalid peer ID %v: expected [0...%v[",
			peer.ID, nw.NumParties)
	}
	nw.m.Lock()
	defer nw.m.Unlock()

	old, ok := nw.peersByID[peer.ID]
	if ok {
		if old.Addr != peer.Addr {
			return fmt.Errorf("address change from %v to %v",
				old.Addr, peer.Addr)
		}
		for i, c := range peer.Conns {
			if c == nil {
				continue
			}
			err := old.SetConn(i, c)
			if err != nil {
				return err
			}
		}
	} else {
		nw.Peers = append(nw.Peers, peer)
		nw.peersByID[peer.ID] = peer
		fmt.Printf("%v: new peer %v\n", nw.Self, peer)

		sort.Slice(nw.Peers, func(i, j int) bool {
			return nw.Peers[i].ID < nw.Peers[j].ID
		})
	}

	return nil
}

func (nw *Network) getPeer(id int) (*Peer, error) {
	nw.m.Lock()
	defer nw.m.Unlock()

	peer, ok := nw.peersByID[id]
	if !ok {
		return nil, fmt.Errorf("unknown peer %v", id)
	}

	return peer, nil
}
