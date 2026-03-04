//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package gmw

import (
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

	Pool *TriplePool

	triples       *Triples
	andD          []uint64
	andE          []uint64
	andZ          []uint64
	andBatchCount int
	andBatchMax   int
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
		Pool:       NewTriplePool(),
		triples:    new(Triples),
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
func (nw *Network) Close() error {
	var err error

	if nw.Pool.started {
		// Wait for triple pool to terminate.
		nw.Pool.Close()
		err = <-nw.Pool.done
	}

	nw.m.Lock()
	defer nw.m.Unlock()

	for _, p := range nw.peers {
		p.Close()
	}

	if nw.listener != nil {
		nw.listener.Close()
	}

	return err
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

	// Collect gates by levels.
	numLevels := int(nw.circ.Stats[circuit.NumLevels]) + 1
	ands := make([][]*circuit.Gate, numLevels)
	rest := make([][]*circuit.Gate, numLevels)

	for i := range nw.circ.Gates {
		gate := &nw.circ.Gates[i]
		level := gate.Level
		if gate.Op == circuit.AND {
			ands[level] = append(ands[level], gate)
		} else {
			rest[level] = append(rest[level], gate)
		}
	}
	debugf("Level\tAND\tRest\n")

	for i := 0; i < numLevels; i++ {
		debugf("%v\t%v\t%v\n", i, len(ands[i]), len(rest[i]))

		for _, gate := range rest[i] {
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

		if err := nw.andBatchFlush(ands[i]); err != nil {
			return err
		}
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

func (nw *Network) andBatchFlush(batch []*circuit.Gate) error {
	self := nw.self

	if len(batch) == 0 {
		return nil
	}

	debugf("AND batch %v: count=%v\n", nw.andBatchCount+1, len(batch))
	nw.andBatchCount++
	if len(batch) > nw.andBatchMax {
		nw.andBatchMax = len(batch)
	}

	words := (len(batch) + 63) / 64

	// Ensure temp arrays has enough space, and clear them for the new
	// batch.
	nw.andD = expandClear(nw.andD, words)
	nw.andE = expandClear(nw.andE, words)
	nw.andZ = expandClear(nw.andZ, words)

	nw.Pool.Get(len(batch), nw.triples)

	if len(nw.triples.A) < words {
		panic("triple size mismatch")
	}
	if nw.triples.Words < words {
		panic("triple word count mismatch")
	}

	// --------------------------------------------
	// Step 1: Compute masked differences
	// d = x XOR a
	// e = y XOR b
	// --------------------------------------------

	var andA, andB uint64
	for i := 0; i < words*64; i++ {
		if i < len(batch) {
			gate := batch[i]
			ofs := i % 64

			a := nw.wires.Bit(int(gate.Input0))
			if a == 1 {
				andA |= (1 << ofs)
			}

			b := nw.wires.Bit(int(gate.Input1))
			if b == 1 {
				andB |= (1 << ofs)
			}
		}

		if (i+1)%64 == 0 {
			// Full word accumulated.
			w := i / 64
			nw.andD[w] = andA ^ nw.triples.A[w]
			nw.andE[w] = andB ^ nw.triples.B[w]

			andA = 0
			andB = 0
		}

	}

	// --------------------------------------------
	// Step 2: Open d and e
	// --------------------------------------------

	dOpen, eOpen, err := nw.broadcastXORs(nw.andD, nw.andE)
	if err != nil {
		return err
	}

	// --------------------------------------------
	// Step 3: Final local computation
	// z = c XOR (d & b) XOR (e & a) XOR (d & e)
	// --------------------------------------------

	for w := 0; w < words; w++ {
		nw.andZ[w] = nw.triples.C[w] ^
			(dOpen[w] & nw.triples.B[w]) ^
			(eOpen[w] & nw.triples.A[w])

		if self.id == 0 {
			nw.andZ[w] ^= (dOpen[w] & eOpen[w])
		}
	}

	// Set result wires.
	for i, gate := range batch {
		nw.wires.SetBit(nw.wires, int(gate.Output), bit(nw.andZ, i))
	}

	nw.triples.Clear()

	return nil
}

func (nw *Network) broadcastXOR(local []uint64) ([]uint64, error) {
	self := nw.self
	result := copyOf(local)

	recv := make([]uint64, len(local))

	for _, peer := range nw.peers {
		if peer.id == self.id {
			continue
		}
		if self.id < peer.id {
			if err := peer.SendBitvec(peer.online, local); err != nil {
				return nil, err
			}
			if err := peer.ReceiveBitvec(peer.online, recv); err != nil {
				return nil, err
			}
		} else {
			if err := peer.ReceiveBitvec(peer.online, recv); err != nil {
				return nil, err
			}
			if err := peer.SendBitvec(peer.online, local); err != nil {
				return nil, err
			}
		}
		xorBitvec(result, recv)
	}

	return result, nil
}

func (nw *Network) broadcastXORs(d, e []uint64) ([]uint64, []uint64, error) {
	self := nw.self
	dOpen := copyOf(d)
	eOpen := copyOf(e)

	dR := make([]uint64, len(d))
	eR := make([]uint64, len(e))

	for _, peer := range nw.peers {
		if peer.id == self.id {
			continue
		}
		if self.id < peer.id {
			if err := peer.SendBitvec2(peer.online, d, e); err != nil {
				return nil, nil, err
			}
			if err := peer.ReceiveBitvec2(peer.online, dR, eR); err != nil {
				return nil, nil, err
			}
		} else {
			if err := peer.ReceiveBitvec2(peer.online, dR, eR); err != nil {
				return nil, nil, err
			}
			if err := peer.SendBitvec2(peer.online, d, e); err != nil {
				return nil, nil, err
			}
		}
		xorBitvec(dOpen, dR)
		xorBitvec(eOpen, eR)
	}

	return dOpen, eOpen, nil
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
