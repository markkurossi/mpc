//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

// Package gmw implements the GMW multi-party protocol.
package gmw

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"sort"
	"sync"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/p2p"
)

// Network implements P2P network.
type Network struct {
	m        sync.Mutex
	circ     *circuit.Circuit
	wires    *big.Int
	output   *big.Int
	listener net.Listener
	peers    []*Peer
	self     *Peer
}

// Peer implements a peer in the P2P network.
type Peer struct {
	id      int
	addr    string
	conn    *p2p.Conn
	input   *big.Int
	randBuf []byte
	shared  *big.Int
}

func (p *Peer) String() string {
	return fmt.Sprintf("%d[%v]", p.id, p.addr)
}

// Close closes the peer.
func (p *Peer) Close() {
	if p.conn != nil {
		p.conn.Close()
	}
}

// shareInput secret shares peer's input with the peer o.
func (p *Peer) shareInput(o *Peer) error {
	_, err := rand.Read(p.randBuf)
	if err != nil {
		return err
	}

	share := new(big.Int).SetBytes(p.randBuf)

	err = o.conn.SendData(p.randBuf)
	if err != nil {
		return err
	}
	err = o.conn.Flush()
	if err != nil {
		return err
	}
	if p.shared == nil {
		p.shared = share
	} else {
		p.shared.Xor(p.shared, share)
	}

	return nil
}

// CreateNetwork creates the network for the leader peer.
func CreateNetwork(addr string, circ *circuit.Circuit) (*Network, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	self := &Peer{
		addr: addr,
	}

	return &Network{
		circ:     circ,
		wires:    new(big.Int),
		listener: l,
		peers:    []*Peer{self},
		self:     self,
	}, nil
}

// JoinNetwork joins the leader's network.
func JoinNetwork(leader, this string, id int, circ *circuit.Circuit) (
	*Network, error) {

	if id == 0 || id >= circ.NumParties() {
		return nil, fmt.Errorf("invalid ID %v: expected [1...%v[",
			id, circ.NumParties())
	}

	l, err := net.Listen("tcp", this)
	if err != nil {
		return nil, err
	}

	c, err := net.Dial("tcp", leader)
	if err != nil {
		l.Close()
		return nil, err
	}

	self := &Peer{
		id:   id,
		addr: this,
	}

	nw := &Network{
		circ:     circ,
		wires:    new(big.Int),
		listener: l,
		peers:    []*Peer{self},
		self:     self,
	}

	err = nw.addPeer(&Peer{
		conn: p2p.NewConn(c),
		addr: leader,
	})
	if err != nil {
		nw.Close()
		return nil, err
	}

	return nw, nil
}

// Close closes the network and all its peer connections.
func (nw *Network) Close() {
	nw.m.Lock()
	defer nw.m.Unlock()

	for _, p := range nw.peers {
		p.Close()
	}

	if nw.listener != nil {
		nw.listener.Close()
	}
}

// Run runs the GMW protocol. The input argument specifies peer's
// input.
func (nw *Network) Run(input *big.Int) ([]*big.Int, error) {
	// Save peer's input.
	nw.self.input = input
	nw.self.randBuf = make([]byte, (nw.circ.Inputs[nw.self.id].Type.Bits+7)/8)

	var err error
	if nw.self.id == 0 {
		err = nw.runLeader()
	} else {
		err = nw.runPeer()
	}
	if err != nil {
		return nil, err
	}

	return nw.circ.Outputs.Split(nw.output), nil
}

func (nw *Network) runLeader() error {
	// Accept all peers.
	for len(nw.peers) < nw.circ.NumParties() {
		c, err := nw.listener.Accept()
		if err != nil {
			return err
		}
		conn := p2p.NewConn(c)
		id, err := conn.ReceiveUint32()
		if err != nil {
			conn.Close()
			return err
		}
		addr, err := conn.ReceiveString()
		if err != nil {
			conn.Close()
			return err
		}
		peer := &Peer{
			id:   id,
			addr: addr,
			conn: conn,
		}
		err = nw.addPeer(peer)
		if err != nil {
			return err
		}
	}
	nw.sortPeers()

	fmt.Printf("All peers connected\n")

	// Send network info to all peers.
	for _, peer := range nw.peers {
		if peer.id == nw.self.id {
			continue
		}
		err := peer.conn.SendUint32(len(nw.peers) - 2)
		if err != nil {
			return err
		}

		for _, i := range nw.peers {
			if i.id == nw.self.id || i.id == peer.id {
				continue
			}
			err = peer.conn.SendUint32(i.id)
			if err != nil {
				return err
			}
			err = peer.conn.SendString(i.addr)
			if err != nil {
				return err
			}
			err = peer.conn.Flush()
			if err != nil {
				return err
			}
		}
	}

	return nw.run()
}

func (nw *Network) runPeer() error {
	self := nw.self

	leader, err := nw.getPeer(0)
	if err != nil {
		return err
	}
	err = leader.conn.SendUint32(self.id)
	if err != nil {
		return err
	}
	err = leader.conn.SendString(self.addr)
	if err != nil {
		return err
	}
	err = leader.conn.Flush()
	if err != nil {
		return err
	}

	// Get other peers' connection endpoints.
	n, err := leader.conn.ReceiveUint32()
	if err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		id, err := leader.conn.ReceiveUint32()
		if err != nil {
			return err
		}
		addr, err := leader.conn.ReceiveString()
		if err != nil {
			return err
		}
		peer := &Peer{
			id:   id,
			addr: addr,
		}
		err = nw.addPeer(peer)
		if err != nil {
			return err
		}
	}
	nw.sortPeers()

	// Connect network.
	for _, peer := range nw.peers {
		if peer.id == 0 || peer.id == self.id {
			continue
		}
		var conn *p2p.Conn
		if self.id < peer.id {
			c, err := net.Dial("tcp", peer.addr)
			if err != nil {
				return err
			}
			conn = p2p.NewConn(c)
		} else {
			c, err := nw.listener.Accept()
			if err != nil {
				return err
			}
			conn = p2p.NewConn(c)
		}
		peer.conn = conn
	}

	return nw.run()
}

func (nw *Network) run() error {
	self := nw.self

	// Secret share inputs.
	for _, peer := range nw.peers {
		if peer.id == self.id {
			continue
		}
		if self.id < peer.id {
			err := self.shareInput(peer)
			if err != nil {
				return err
			}
			err = nw.receiveInput(peer)
			if err != nil {
				return err
			}
		} else {
			err := nw.receiveInput(peer)
			if err != nil {
				return err
			}
			err = self.shareInput(peer)
			if err != nil {
				return err
			}
		}
	}
	// Set our input.
	self.shared.Xor(self.shared, self.input)
	nw.setWires(self, self.shared)

	// Evaluate circuit.
	for i := 0; i < len(nw.circ.Gates); i++ {
		gate := &nw.circ.Gates[i]

		a := int(gate.Input0)
		c := int(gate.Output)

		var b int

		if gate.Op != circuit.INV {
			b = int(gate.Input1)
		}

		var bit uint

		switch gate.Op {
		case circuit.XOR:
			bit = nw.wires.Bit(a) ^ nw.wires.Bit(b)

		default:
			return fmt.Errorf("gate %v not supported", gate.Op)
		}
		nw.wires.SetBit(nw.wires, c, bit)
	}

	// Share outputs.

	nout := nw.circ.Outputs.Size()
	nw.output = new(big.Int).Rsh(nw.wires, uint(nw.circ.NumWires-nout))
	outputBuf := nw.output.Bytes()

	for _, peer := range nw.peers {
		if peer.id == self.id {
			continue
		}
		if self.id < peer.id {
			err := nw.sendOutput(peer, outputBuf)
			if err != nil {
				return err
			}
			err = nw.receiveOutput(peer)
			if err != nil {
				return err
			}
		} else {
			err := nw.receiveOutput(peer)
			if err != nil {
				return err
			}
			err = nw.sendOutput(peer, outputBuf)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// receiveInput receives the input share from the peer o.
func (nw *Network) receiveInput(o *Peer) error {
	data, err := o.conn.ReceiveData()
	if err != nil {
		return err
	}
	input := new(big.Int).SetBytes(data)
	nw.setWires(o, input)

	return nil
}

func (nw *Network) sendOutput(peer *Peer, output []byte) error {
	err := peer.conn.SendData(output)
	if err != nil {
		return err
	}
	return peer.conn.Flush()
}

func (nw *Network) receiveOutput(peer *Peer) error {
	data, err := peer.conn.ReceiveData()
	if err != nil {
		return err
	}
	output := new(big.Int).SetBytes(data)

	if nw.output == nil {
		nw.output = output
	} else {
		nw.output.Xor(nw.output, output)
	}

	return nil
}

func (nw *Network) setWires(o *Peer, input *big.Int) {
	var ofs int
	for i := 0; i < o.id; i++ {
		ofs += int(nw.circ.Inputs[i].Type.Bits)
	}
	for i := 0; i < int(nw.circ.Inputs[o.id].Type.Bits); i++ {
		nw.wires.SetBit(nw.wires, ofs+i, input.Bit(i))
	}

}

func (nw *Network) addPeer(peer *Peer) error {
	if peer.id >= nw.circ.NumParties() {
		return fmt.Errorf("invalid peer ID %v: expected [0...%v[",
			peer.id, nw.circ.NumParties())
	}
	nw.m.Lock()
	defer nw.m.Unlock()

	for _, p := range nw.peers {
		if p.id == peer.id {
			return fmt.Errorf("peer %v already defined", peer.id)
		}
	}
	nw.peers = append(nw.peers, peer)

	fmt.Printf("New peer %v\n", peer)

	return nil
}

func (nw *Network) sortPeers() {
	nw.m.Lock()
	defer nw.m.Unlock()

	sort.Slice(nw.peers, func(i, j int) bool {
		return nw.peers[i].id < nw.peers[j].id
	})
}

func (nw *Network) getPeer(id int) (*Peer, error) {
	nw.m.Lock()
	defer nw.m.Unlock()

	for _, p := range nw.peers {
		if p.id == id {
			return p, nil
		}
	}
	return nil, fmt.Errorf("unknown peer %v", id)
}
