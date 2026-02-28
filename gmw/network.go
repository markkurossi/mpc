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
	"net"
	"sort"
	"sync"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

// Network implements P2P network.
type Network struct {
	m           sync.Mutex
	numParties  int
	needOnline  int
	needOffline int
	circ        *circuit.Circuit
	inputSizes  [][]int
	wires       *big.Int
	output      *big.Int
	listener    net.Listener
	peers       []*Peer
	peersByID   map[int]*Peer
	self        *Peer
	done        chan error

	pool *TriplePool

	andBatch      []*circuit.Gate
	andA          []uint
	andB          []uint
	andBatchCount int
	andBatchMax   int
	andBatchLevel int
}

// NewNetwork creates a new network instance.
func NewNetwork(numParties int, listener net.Listener, self *Peer) *Network {
	nw := &Network{
		numParties: numParties,
		inputSizes: make([][]int, numParties),
		wires:      new(big.Int),
		listener:   listener,
		peersByID:  make(map[int]*Peer),
		self:       self,
		done:       make(chan error),
		pool:       NewTriplePool(),
	}
	err := nw.addPeer(self)
	if err != nil {
		panic(err)
	}

	return nw
}

func (nw *Network) accept(online, offline bool) {
	nw.done <- nw.acceptLoop(online, offline)
}

func (nw *Network) acceptLoop(online, offline bool) error {

	for (online && nw.needOnline > 0) || (offline && nw.needOffline > 0) {
		c, err := nw.listener.Accept()
		if err != nil {
			return err
		}
		conn := p2p.NewConn(c)
		magic, err := conn.ReceiveUint32()
		if err != nil {
			conn.Close()
			return err
		}
		switch magic {
		case MagicOnline:
			if nw.needOnline == 0 {
				conn.Close()
				return fmt.Errorf("unexpected online connection")
			}
			nw.needOnline--
			err = nw.acceptOnline(conn)
		case MagicOffline:
			if nw.needOffline == 0 {
				conn.Close()
				return fmt.Errorf("unexpected offline connection")
			}
			nw.needOffline--
			err = nw.acceptOffline(conn)
		default:
			err = fmt.Errorf("invalid connection magic: %x", magic)
		}
		if err != nil {
			conn.Close()
			return err
		}
	}

	return nil
}

func (nw *Network) acceptOnline(conn *p2p.Conn) error {
	id, err := conn.ReceiveUint32()
	if err != nil {
		return err
	}
	addr, err := conn.ReceiveString()
	if err != nil {
		return err
	}
	peer := &Peer{
		id:     id,
		addr:   addr,
		online: conn,
	}
	err = nw.addPeer(peer)
	if err != nil {
		return err
	}
	inputs, err := conn.ReceiveInputSizes()
	if err != nil {
		return err
	}
	nw.inputSizes[id] = inputs

	// Send out input sizes.
	err = conn.SendInputSizes(nw.inputSizes[nw.self.id])
	if err != nil {
		return err
	}
	return conn.Flush()
}

func (nw *Network) acceptOffline(conn *p2p.Conn) error {
	id, err := conn.ReceiveUint32()
	if err != nil {
		return err
	}
	nw.m.Lock()
	defer nw.m.Unlock()
	peer, ok := nw.peersByID[id]
	if !ok {
		return fmt.Errorf("invalid offline peer ID %v", id)
	}
	peer.offline = conn

	return nil
}

func (nw *Network) addPeer(peer *Peer) error {
	if peer.id >= nw.numParties {
		return fmt.Errorf("invalid peer ID %v: expected [0...%v[",
			peer.id, nw.numParties)
	}
	nw.m.Lock()
	defer nw.m.Unlock()

	old, ok := nw.peersByID[peer.id]
	if ok {
		if old.online != nil {
			return fmt.Errorf("peer %v already connected", peer.id)
		}
		old.online = peer.online
	} else {
		nw.peers = append(nw.peers, peer)
		nw.peersByID[peer.id] = peer
		fmt.Printf("New peer %v\n", peer)
	}

	return nil
}

// CreateNetwork creates the network for the leader peer.
func CreateNetwork(addr string, numParties int) (*Network, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewNetwork(numParties, l, &Peer{
		addr: addr,
	}), nil
}

// JoinNetwork joins the leader's network.
func JoinNetwork(leader, this string, id int) (*Network, error) {
	if id == 0 {
		return nil, fmt.Errorf("invalid ID %v", id)
	}
	l, err := net.Listen("tcp", this)
	if err != nil {
		return nil, err
	}
	nw := NewNetwork(id+1, l, &Peer{
		id:   id,
		addr: this,
	})

	c, err := net.Dial("tcp", leader)
	if err != nil {
		nw.Close()
		return nil, err
	}

	err = nw.addPeer(&Peer{
		online: p2p.NewConn(c),
		addr:   leader,
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

// Stats returns the network IO statistics.
func (nw *Network) Stats() (p2p.IOStats, p2p.IOStats) {
	online := p2p.NewIOStats()
	offline := p2p.NewIOStats()

	for _, p := range nw.peers {
		if p.online != nil {
			online = online.Add(p.online.Stats)
		}
		if p.offline != nil {
			offline = offline.Add(p.offline.Stats)
		}
	}

	return online, offline
}

// Connect connects the p2p network. After this call, all peers have
// connected with each other and exchanged their input sizes.
func (nw *Network) Connect(inputSizes []int) error {
	nw.inputSizes[nw.self.id] = inputSizes

	if nw.self.id == 0 {
		return nw.connectLeader()
	}
	return nw.connectPeer()
}

func (nw *Network) connectLeader() error {
	// Accept all online connections.

	accept := nw.numParties - 1
	nw.needOnline = accept
	nw.needOffline = accept

	go nw.accept(true, false)
	err := <-nw.done
	if err != nil {
		return err
	}
	nw.sortPeers()

	fmt.Printf("All peers connected\n")

	// Accept all offline connections.
	go nw.accept(true, true)

	// Send network info to all peers.
	for _, peer := range nw.peers {
		if peer.id == nw.self.id {
			continue
		}
		err := peer.online.SendUint32(len(nw.peers) - 2)
		if err != nil {
			return err
		}
		for _, i := range nw.peers {
			if i.id == nw.self.id || i.id == peer.id {
				continue
			}
			err = peer.online.SendUint32(i.id)
			if err != nil {
				return err
			}
			err = peer.online.SendString(i.addr)
			if err != nil {
				return err
			}
		}
		err = peer.online.Flush()
		if err != nil {
			return err
		}
	}

	// Wait all offline connections.
	err = <-nw.done
	if err != nil {
		return err
	}

	return nw.startOffline()
}

func (nw *Network) connectPeer() error {
	self := nw.self

	leader, err := nw.getPeer(0)
	if err != nil {
		return err
	}
	if err := leader.online.SendUint32(MagicOnline); err != nil {
		return err
	}
	if err := leader.online.SendUint32(self.id); err != nil {
		return err
	}
	if err := leader.online.SendString(self.addr); err != nil {
		return err
	}
	if err := leader.online.SendInputSizes(nw.inputSizes[self.id]); err != nil {
		return err
	}
	if err := leader.online.Flush(); err != nil {
		return err
	}
	// Get leader's input sizes.
	inputs, err := leader.online.ReceiveInputSizes()
	if err != nil {
		return err
	}
	nw.inputSizes[leader.id] = inputs

	// Get other peers' connection endpoints.

	n, err := leader.online.ReceiveUint32()
	if err != nil {
		return err
	}
	nw.numParties = 2 + n
	inputSizes := make([][]int, nw.numParties)
	copy(inputSizes, nw.inputSizes)
	nw.inputSizes = inputSizes

	var numAccept int

	for i := 0; i < n; i++ {
		id, err := leader.online.ReceiveUint32()
		if err != nil {
			return err
		}
		addr, err := leader.online.ReceiveString()
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
		if self.id > id {
			numAccept++
		}
	}
	nw.needOnline = numAccept
	nw.needOffline = numAccept
	nw.sortPeers()

	// Offline connection with leader.
	err = nw.dialOffline(leader)
	if err != nil {
		return err
	}

	go nw.accept(true, true)

	// Connect network.
	for _, peer := range nw.peers {
		if peer.id == 0 || peer.id <= self.id {
			continue
		}
		err = nw.dialOnline(peer)
		if err != nil {
			return err
		}
		err = nw.dialOffline(peer)
		if err != nil {
			return err
		}
	}

	// Wait until all peers have been connected.
	err = <-nw.done
	if err != nil {
		return err
	}

	return nw.startOffline()
}

func (nw *Network) dialOnline(peer *Peer) error {
	self := nw.self

	fmt.Printf("%v: dialOnline(%v)\n", self, peer)

	c, err := net.Dial("tcp", peer.addr)
	if err != nil {
		return err
	}
	conn := p2p.NewConn(c)
	if err := conn.SendUint32(MagicOnline); err != nil {
		return err
	}
	if err := conn.SendUint32(self.id); err != nil {
		return err
	}
	if err := conn.SendString(self.addr); err != nil {
		return err
	}
	if err := conn.SendInputSizes(nw.inputSizes[self.id]); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	// Get peer's input sizes.
	inputs, err := conn.ReceiveInputSizes()
	if err != nil {
		return err
	}
	nw.m.Lock()
	nw.inputSizes[peer.id] = inputs
	nw.m.Unlock()
	peer.online = conn

	return nil
}

func (nw *Network) dialOffline(peer *Peer) error {
	self := nw.self

	fmt.Printf("%v: dialOffline(%v)\n", self, peer)

	c, err := net.Dial("tcp", peer.addr)
	if err != nil {
		return err
	}
	conn := p2p.NewConn(c)
	if err := conn.SendUint32(MagicOffline); err != nil {
		return err
	}
	if err := conn.SendUint32(self.id); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	peer.offline = conn

	return nil
}

// NumParties returns the number of parties in the network.
func (nw *Network) NumParties() int {
	return nw.numParties
}

// InputSizes returns the input sizes of the network parties.
func (nw *Network) InputSizes() [][]int {
	return nw.inputSizes
}

// Run runs the GMW protocol. The input argument specifies peer's
// input.
func (nw *Network) Run(input *big.Int, circ *circuit.Circuit, verbose bool) (
	[]*big.Int, error) {

	if circ.NumParties() != nw.numParties {
		return nil, fmt.Errorf("invalid %v-party circuit for %d-party MPC",
			circ.NumParties(), nw.numParties)
	}
	nw.circ = circ

	// Save peer's input.
	nw.self.input = input
	nw.self.randBuf = make([]byte, (nw.circ.Inputs[nw.self.id].Type.Bits+7)/8)
	nw.self.shared = big.NewInt(0)

	err := nw.run(verbose)
	if err != nil {
		return nil, err
	}

	return nw.circ.Outputs.Split(nw.output), nil
}

func (nw *Network) run(verbose bool) error {
	self := nw.self

	if verbose {
		fmt.Printf("%v: run: %v\n", self, nw.circ)
	}

	// Create OT instances.
	for _, peer := range nw.peers {
		if peer.id == self.id {
			continue
		}
		peer.otS = nw.newOT()
		peer.otR = nw.newOT()

		if self.id < peer.id {
			if err := peer.otS.InitSender(peer.online); err != nil {
				return err
			}
			if err := peer.otR.InitReceiver(peer.online); err != nil {
				return err
			}
		} else {
			if err := peer.otR.InitReceiver(peer.online); err != nil {
				return err
			}
			if err := peer.otS.InitSender(peer.online); err != nil {
				return err
			}
		}
	}

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

		if int(gate.Level) != nw.andBatchLevel {
			if err := nw.andBatchFlush(int(gate.Level)); err != nil {
				return err
			}
		}

		a := nw.wires.Bit(int(gate.Input0))

		var b uint
		if gate.Op != circuit.INV {
			b = nw.wires.Bit(int(gate.Input1))
		}

		var bit uint

		switch gate.Op {
		case circuit.XOR:
			bit = a ^ b

		case circuit.XNOR:
			bit = a ^ b
			if self.id == 0 {
				bit ^= 1
			}

		case circuit.AND:
			if err := nw.andBatchAdd(gate, a, b); err != nil {
				return err
			}
			continue

		case circuit.INV:
			if self.id == 0 {
				bit = a ^ 1
			} else {
				bit = a
			}

		default:
			return fmt.Errorf("gate %v not supported", gate.Op)
		}
		nw.wires.SetBit(nw.wires, int(gate.Output), bit)
	}

	if err := nw.andBatchFlush(0); err != nil {
		return err
	}

	if verbose {
		fmt.Printf("AND: #batches=%v, max=%v, AND/batch=%.2f\n",
			nw.andBatchCount, nw.andBatchMax,
			float64(nw.circ.Stats[circuit.AND])/float64(nw.andBatchCount))
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

func (nw *Network) andBatchAdd(gate *circuit.Gate, a, b uint) error {
	nw.andBatch = append(nw.andBatch, gate)
	nw.andA = append(nw.andA, a)
	nw.andB = append(nw.andB, b)
	return nil
}

func (nw *Network) andBatchFlush(level int) error {
	if len(nw.andBatch) == 0 {
		nw.andBatchLevel = level
		return nil
	}

	debugf("AND batch %v: count=%v\n", nw.andBatchLevel+1, len(nw.andBatch))
	nw.andBatchCount++
	if len(nw.andBatch) > nw.andBatchMax {
		nw.andBatchMax = len(nw.andBatch)
	}

	self := nw.self
	n := len(nw.andBatch)

	// Compute local terms.
	z := make([]uint, n)
	for i := 0; i < len(z); i++ {
		z[i] = nw.andA[i] & nw.andB[i]
	}
	var m sync.Mutex
	var wg sync.WaitGroup
	var otErr error

	setOTErr := func(err error) {
		m.Lock()
		otErr = err
		m.Unlock()
	}

	// Batched cross terms via ROT.
	for _, peer := range nw.peers {
		if peer.id == self.id {
			continue
		}
		wg.Go(func() {
			var err error
			var senderShare, receiverShare []uint

			if self.id < peer.id {
				senderShare, err = peer.otSend(self, nw.andA)
				if err != nil {
					setOTErr(err)
					return
				}
				receiverShare, err = peer.otReceive(self, nw.andB)
				if err != nil {
					setOTErr(err)
					return
				}
			} else {
				receiverShare, err = peer.otReceive(self, nw.andB)
				if err != nil {
					setOTErr(err)
					return
				}
				senderShare, err = peer.otSend(self, nw.andA)
				if err != nil {
					setOTErr(err)
					return
				}
			}

			// Combine result into z.
			m.Lock()
			for i := 0; i < n; i++ {
				z[i] ^= senderShare[i]
				z[i] ^= receiverShare[i]
			}
			m.Unlock()
		})
	}
	wg.Wait()
	if otErr != nil {
		return otErr
	}

	// Set result wires.
	for i, gate := range nw.andBatch {
		nw.wires.SetBit(nw.wires, int(gate.Output), z[i])
	}

	nw.andBatch = nw.andBatch[:0]
	nw.andA = nw.andA[:0]
	nw.andB = nw.andB[:0]
	nw.andBatchLevel = level

	return nil
}

func (nw *Network) newOT() ot.OT {
	return ot.NewROT(ot.NewCO(rand.Reader), rand.Reader, false, true)
}

// receiveInput receives the input share from the peer o.
func (nw *Network) receiveInput(o *Peer) error {
	data, err := o.online.ReceiveData()
	if err != nil {
		return err
	}
	input := new(big.Int).SetBytes(data)
	nw.setWires(o, input)

	return nil
}

func (nw *Network) sendOutput(peer *Peer, output []byte) error {
	err := peer.online.SendData(output)
	if err != nil {
		return err
	}
	return peer.online.Flush()
}

func (nw *Network) receiveOutput(peer *Peer) error {
	data, err := peer.online.ReceiveData()
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
