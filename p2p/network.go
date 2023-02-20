//
// Copyright (c) 2020-2023 Markku Rossi
//
// All rights reserved.
//

package p2p

import (
	"crypto/rsa"
	"fmt"
	"log"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/markkurossi/mpc/ot"
)

// Network implements peer-to-peer network.
type Network struct {
	ID       int
	m        sync.Mutex
	Peers    map[int]*Peer
	addr     string
	listener net.Listener
}

// NewNetwork creats a new peer-to-peer network.
func NewNetwork(addr string, id int) (*Network, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	nw := &Network{
		ID:       id,
		Peers:    make(map[int]*Peer),
		addr:     addr,
		listener: listener,
	}
	go nw.acceptLoop()
	return nw, nil
}

// Close closes the network.
func (nw *Network) Close() error {
	return nw.listener.Close()
}

// AddPeer adds a peer to the network.
func (nw *Network) AddPeer(addr string, id int) error {
	// Try to connect to peer.
	for {
		// Check if we have already accepted peer `id`.
		nw.m.Lock()
		_, ok := nw.Peers[id]
		nw.m.Unlock()
		if ok {
			return nil
		}

		log.Printf("NW %d: Connecting to peer %d...\n", nw.ID, id)
		nc, err := net.Dial("tcp", addr)
		if err != nil {
			delay := 5 * time.Second
			log.Printf("NW %d: Connect to %s failed, retrying in %s\n",
				nw.ID, addr, delay)
			<-time.After(delay)
			continue
		}
		log.Printf("NW %d: Connected to %s\n", nw.ID, addr)
		conn := NewConn(nc)

		if err := conn.SendUint32(nw.ID); err != nil {
			conn.Close()
			return err
		}
		if err := conn.Flush(); err != nil {
			conn.Close()
			return err
		}
		if err := nw.newPeer(true, conn, id); err != nil {
			fmt.Printf("Failed to add peer: %s\n", err)
		}
	}
}

// Ping sends a ping message to all peers.
func (nw *Network) Ping() {
	for _, peer := range nw.Peers {
		peer.Ping()
	}
}

// Stats returns the I/O stats from the network.
func (nw *Network) Stats() IOStats {
	var result IOStats
	for _, peer := range nw.Peers {
		result = result.Add(peer.conn.Stats)
	}
	return result
}

func (nw *Network) acceptLoop() {
	for {
		nc, err := nw.listener.Accept()
		if err != nil {
			log.Printf("NW %d: accept failed: %s\n", nw.ID, err)
			return
		}
		conn := NewConn(nc)

		// Read peer ID.
		id, err := conn.ReceiveUint32()
		if err != nil {
			log.Printf("NW %d: I/O error: %s\n", nw.ID, err)
			conn.Close()
			continue
		}

		err = nw.newPeer(false, conn, id)
		if err != nil {
			log.Printf("inbound connection error: %s\n", err)
		}
	}
}

func (nw *Network) newPeer(client bool, conn *Conn, id int) error {
	nw.m.Lock()
	peer, ok := nw.Peers[id]
	if ok {
		nw.m.Unlock()
		log.Printf("NW %d: peer %d already connected\n", nw.ID, id)
		return conn.Close()
	}
	peer = &Peer{
		id:     id,
		conn:   conn,
		client: client,
	}
	nw.Peers[id] = peer
	nw.m.Unlock()

	return peer.init()
}

// Peer implements a peer in the peer-to-peer network.
type Peer struct {
	id         int
	conn       *Conn
	client     bool
	otSender   *ot.Sender
	otReceiver *ot.Receiver
}

// Close closes the peer connection.
func (peer *Peer) Close() error {
	return peer.conn.Close()
}

// Ping sends a ping message to the peer.
func (peer *Peer) Ping() error {
	if err := peer.conn.SendUint32(0xffffffff); err != nil {
		return err
	}
	return peer.conn.Flush()
}

func (peer *Peer) init() error {
	fmt.Printf("peer %d: init\n", peer.id)

	// Read peer public key.
	finished := make(chan error)
	go func() {
		pubN, err := peer.conn.ReceiveData()
		if err != nil {
			finished <- err
			return
		}
		pubE, err := peer.conn.ReceiveUint32()
		if err != nil {
			finished <- err
			return
		}
		pub := &rsa.PublicKey{
			N: new(big.Int).SetBytes(pubN),
			E: pubE,
		}
		receiver, err := ot.NewReceiver(pub)
		if err != nil {
			finished <- err
			return
		}
		peer.otReceiver = receiver
		finished <- nil
	}()

	// Init oblivious transfer.
	sender, err := ot.NewSender(2048)
	if err != nil {
		<-finished
		return err
	}
	peer.otSender = sender

	// Send our public key to peer.
	pub := sender.PublicKey()
	data := pub.N.Bytes()
	if err := peer.conn.SendData(data); err != nil {
		<-finished
		return err
	}
	if err := peer.conn.SendUint32(pub.E); err != nil {
		<-finished
		return err
	}
	if err := peer.conn.Flush(); err != nil {
		<-finished
		return err
	}

	return <-finished
}

// OTLambda runs the lambda oblivious transfers with peers.
func (peer *Peer) OTLambda(count int, choices, x1, x2 *big.Int) (
	result *big.Int, err error) {

	var mode string
	if peer.client {
		mode = "OT Lambda client"
	} else {
		mode = "OT Lambda server"
	}

	fmt.Printf("   - %s for peer %d: count=%d\n", mode, peer.id, count)

	if peer.client {
		// Client queries first.
		result, err = peer.otLambdaQuery(count, choices)
		if err != nil {
			return
		}

		// Serve peer queries.
		err = peer.otLambdaRespond(count, x1, x2)
		if err != nil {
			return
		}
	} else {
		// Serve peer queries.
		err = peer.otLambdaRespond(count, x1, x2)
		if err != nil {
			return
		}

		// Server queries second.
		result, err = peer.otLambdaQuery(count, choices)
		if err != nil {
			return
		}
	}
	return
}

func (peer *Peer) otLambdaQuery(count int, choices *big.Int) (
	*big.Int, error) {

	// Number of OTs following
	if err := peer.conn.SendUint32(count); err != nil {
		return nil, err
	}
	if err := peer.conn.Flush(); err != nil {
		return nil, err
	}

	// OTs for each query.
	result := new(big.Int)
	for i := 0; i < count; i++ {
		n, err := peer.conn.Receive(peer.otReceiver, uint(i), choices.Bit(i))
		if err != nil {
			return nil, err
		}
		if len(n) != 1 {
			return nil, fmt.Errorf("invalid OT result of length %d", len(n))
		}
		if n[0] != 0 {
			result.SetBit(result, i, 1)
		}
	}
	return result, nil
}

func (peer *Peer) otLambdaRespond(count int, x1, x2 *big.Int) error {
	pc, err := peer.conn.ReceiveUint32()
	if err != nil {
		return err
	}
	if pc != count {
		return fmt.Errorf("protocol error: peer count %d, our %d", pc, count)
	}
	for i := 0; i < count; i++ {
		bit, err := peer.conn.ReceiveUint32()
		if err != nil {
			return err
		}
		var m0, m1 [1]byte

		if x1.Bit(bit) != 0 {
			m0[0] = 1
		}
		if x2.Bit(bit) != 0 {
			m1[0] = 1
		}

		xfer, err := peer.otSender.NewTransfer(m0[:], m1[:])
		if err != nil {
			return err
		}
		x0, x1 := xfer.RandomMessages()
		if err := peer.conn.SendData(x0); err != nil {
			return err
		}
		if err := peer.conn.SendData(x1); err != nil {
			return err
		}
		if err := peer.conn.Flush(); err != nil {
			return err
		}

		v, err := peer.conn.ReceiveData()
		if err != nil {
			return err
		}
		xfer.ReceiveV(v)

		m0p, m1p, err := xfer.Messages()
		if err != nil {
			return err
		}
		if err := peer.conn.SendData(m0p); err != nil {
			return err
		}
		if err := peer.conn.SendData(m1p); err != nil {
			return err
		}
		if err := peer.conn.Flush(); err != nil {
			return err
		}
	}

	return nil
}

// OTR runs the R share oblivious transfers with peers.
func (peer *Peer) OTR(chA, chB, chC *big.Int,
	x1Ag, x2Ag, x1Bg, x2Bg, x1Cg, x2Cg []ot.Label) (
	ra, rb, rc []ot.Label, err error) {

	var mode string
	if peer.client {
		mode = "OT R client"
	} else {
		mode = "OT R server"
	}

	fmt.Printf("   - %s for peer %d: count=%d\n", mode, peer.id, len(x1Ag))

	if peer.client {
		ra, rb, rc, err = peer.otrQueries(len(x1Ag), chA, chB, chC)
		if err != nil {
			return
		}
		err = peer.otrResponses(x1Ag, x2Ag, x1Bg, x2Bg, x1Cg, x2Cg)
		if err != nil {
			return
		}
	} else {
		err = peer.otrResponses(x1Ag, x2Ag, x1Bg, x2Bg, x1Cg, x2Cg)
		if err != nil {
			return
		}
		ra, rb, rc, err = peer.otrQueries(len(x1Ag), chA, chB, chC)
		if err != nil {
			return
		}
	}

	return
}

func (peer *Peer) otrQueries(count int, chA, chB, chC *big.Int) (
	ra, rb, rc []ot.Label, err error) {

	ra, err = peer.otrQuery(count, chA)
	if err != nil {
		return
	}
	rb, err = peer.otrQuery(count, chB)
	if err != nil {
		return
	}
	rc, err = peer.otrQuery(count, chC)
	if err != nil {
		return
	}
	return
}

func (peer *Peer) otrQuery(count int, choices *big.Int) ([]ot.Label, error) {

	// Number of OTs following
	if err := peer.conn.SendUint32(count); err != nil {
		return nil, err
	}
	if err := peer.conn.Flush(); err != nil {
		return nil, err
	}

	result := make([]ot.Label, count)
	for i := 0; i < count; i++ {
		n, err := peer.conn.Receive(peer.otReceiver, uint(i), choices.Bit(i))
		if err != nil {
			return nil, err
		}
		result[i].SetBytes(n)
	}

	return result, nil
}

func (peer *Peer) otrResponses(x1Ag, x2Ag, x1Bg, x2Bg,
	x1Cg, x2Cg []ot.Label) error {
	if err := peer.otrRespond(x1Ag, x2Ag); err != nil {
		return err
	}
	if err := peer.otrRespond(x1Bg, x2Bg); err != nil {
		return err
	}
	if err := peer.otrRespond(x1Cg, x2Cg); err != nil {
		return err
	}
	return nil
}

func (peer *Peer) otrRespond(x1, x2 []ot.Label) error {

	pc, err := peer.conn.ReceiveUint32()
	if err != nil {
		return err
	}
	if pc != len(x1) {
		return fmt.Errorf("protocol error: peer count %d, our %d", pc, len(x1))
	}

	for i := 0; i < len(x1); i++ {
		bit, err := peer.conn.ReceiveUint32()
		if err != nil {
			return err
		}
		var m0Buf, m1Buf ot.LabelData
		m0 := x1[bit].Bytes(&m0Buf)
		m1 := x2[bit].Bytes(&m1Buf)

		xfer, err := peer.otSender.NewTransfer(m0, m1)
		if err != nil {
			return err
		}
		x0, x1 := xfer.RandomMessages()
		if err := peer.conn.SendData(x0); err != nil {
			return err
		}
		if err := peer.conn.SendData(x1); err != nil {
			return err
		}
		if err := peer.conn.Flush(); err != nil {
			return err
		}

		v, err := peer.conn.ReceiveData()
		if err != nil {
			return err
		}
		xfer.ReceiveV(v)

		m0p, m1p, err := xfer.Messages()
		if err != nil {
			return err
		}
		if err := peer.conn.SendData(m0p); err != nil {
			return err
		}
		if err := peer.conn.SendData(m1p); err != nil {
			return err
		}
		if err := peer.conn.Flush(); err != nil {
			return err
		}
	}

	return nil
}

// ExchangeGates exchanges gate values with peers.
func (peer *Peer) ExchangeGates(ag, bg, cg, dg [][]ot.Label, lo *big.Int) (
	ra, rb, rc, rd [][]ot.Label, ro *big.Int, err error) {

	var mode string
	if peer.client {
		mode = "Exch client"
	} else {
		mode = "Exch server"
	}

	fmt.Printf("   - %s for peer %d\n", mode, peer.id)

	if peer.client {
		err = peer.exchangeSend(ag, bg, cg, dg, lo)
		if err != nil {
			return
		}
		ra, rb, rc, rd, ro, err = peer.exchangeReceive()
		if err != nil {
			return
		}
	} else {
		ra, rb, rc, rd, ro, err = peer.exchangeReceive()
		if err != nil {
			return
		}
		err = peer.exchangeSend(ag, bg, cg, dg, lo)
		if err != nil {
			return
		}
	}

	return
}

func (peer *Peer) exchangeSend(ag, bg, cg, dg [][]ot.Label, lo *big.Int) (
	err error) {
	// Number of peers
	if err := peer.conn.SendUint32(len(ag)); err != nil {
		return err
	}
	// Gates for all peers.
	for p := 0; p < len(ag); p++ {
		if err := peer.exchangeSendArr(ag[p]); err != nil {
			return err
		}
		if err := peer.exchangeSendArr(bg[p]); err != nil {
			return err
		}
		if err := peer.exchangeSendArr(cg[p]); err != nil {
			return err
		}
		if err := peer.exchangeSendArr(dg[p]); err != nil {
			return err
		}
	}
	if err := peer.conn.SendData(lo.Bytes()); err != nil {
		return err
	}
	return peer.conn.Flush()
}

func (peer *Peer) exchangeSendArr(arr []ot.Label) (err error) {
	if err := peer.conn.SendUint32(len(arr)); err != nil {
		return err
	}
	var labelData ot.LabelData
	for _, label := range arr {
		if err := peer.conn.SendLabel(label, &labelData); err != nil {
			return err
		}
	}
	return nil
}

func (peer *Peer) exchangeReceive() (
	ras, rbs, rcs, rds [][]ot.Label, ro *big.Int, err error) {

	// Number of peers.
	var count int
	count, err = peer.conn.ReceiveUint32()
	if err != nil {
		return
	}
	for p := 0; p < count; p++ {
		var arr []ot.Label
		arr, err = peer.exchangeReceiveArr()
		if err != nil {
			return
		}
		ras = append(ras, arr)
		arr, err = peer.exchangeReceiveArr()
		if err != nil {
			return
		}
		rbs = append(rbs, arr)
		arr, err = peer.exchangeReceiveArr()
		if err != nil {
			return
		}
		rcs = append(rcs, arr)
		arr, err = peer.exchangeReceiveArr()
		if err != nil {
			return
		}
		rds = append(rds, arr)
	}

	var buf []byte
	buf, err = peer.conn.ReceiveData()
	if err != nil {
		return
	}

	ro = new(big.Int).SetBytes(buf)
	return
}

func (peer *Peer) exchangeReceiveArr() ([]ot.Label, error) {
	count, err := peer.conn.ReceiveUint32()
	if err != nil {
		return nil, err
	}
	var result []ot.Label
	var label ot.Label
	var data ot.LabelData
	for i := 0; i < count; i++ {
		err := peer.conn.ReceiveLabel(&label, &data)
		if err != nil {
			return nil, err
		}
		result = append(result, label)
	}
	return result, nil
}
