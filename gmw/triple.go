//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.

package gmw

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/markkurossi/mpc/ot"
)

var (
	bo = binary.BigEndian
)

const (
	lowWaterMark = 1024
)

type msgType byte

const (
	msgTriple msgType = iota
)

type TriplePool struct {
	m     sync.Mutex
	c     *sync.Cond
	count int
}

func NewTriplePool() *TriplePool {
	pool := &TriplePool{}

	pool.c = sync.NewCond(&pool.m)

	return pool
}

func (nw *Network) startOffline() error {
	self := nw.self

	// Create offline OT instances.
	for _, peer := range nw.peers {
		if peer.id == self.id {
			continue
		}
		if self.id < peer.id {
			if err := nw.iknpSender(peer); err != nil {
				return err
			}
			if err := nw.iknpReceiver(peer); err != nil {
				return err
			}
		} else {
			if err := nw.iknpReceiver(peer); err != nil {
				return err
			}
			if err := nw.iknpSender(peer); err != nil {
				return err
			}
		}
	}

	if self.id == 0 {
		go nw.tripleSender()
	} else {
		go nw.tripleReceiver()
	}
	return nil
}

func (nw *Network) iknpSender(peer *Peer) error {
	co := ot.NewCO(rand.Reader)
	err := co.InitSender(peer.offline)
	if err != nil {
		return err
	}

	peer.iknpS, err = ot.NewIKNPSender(co, peer.offline, rand.Reader, nil)
	return err
}

func (nw *Network) iknpReceiver(peer *Peer) error {
	co := ot.NewCO(rand.Reader)
	err := co.InitReceiver(peer.offline)
	if err != nil {
		return err
	}

	peer.iknpR, err = ot.NewIKNPReceiver(co, peer.offline, rand.Reader)
	return err
}

func (nw *Network) tripleSender() error {
	self := nw.self
	batchSize := 128

	for {
		nw.pool.m.Lock()
		for nw.pool.count > lowWaterMark {
			nw.pool.c.Wait()
		}
		nw.pool.m.Unlock()

		msg := int(msgTriple)<<24 | (batchSize & 0x00ffffff)

		fmt.Printf("%v: new triple batch: size=%v\n", self, batchSize)

		// Notify all peers about the new patch.
		for _, peer := range nw.peers {
			if peer.id == self.id {
				continue
			}
			if err := peer.offline.SendUint32(msg); err != nil {
				return err
			}
			if err := peer.offline.Flush(); err != nil {
				return err
			}
		}
		err := nw.tripleBatch(batchSize)
		if err != nil {
			return err
		}

		// Increase batch size after first batch.
		batchSize = 1024
	}
}

func (nw *Network) tripleReceiver() error {
	leader, err := nw.getPeer(0)
	if err != nil {
		return err
	}
	self := nw.self
	if self.id == leader.id {
		return fmt.Errorf("leader %v acting as triple receiver", leader)
	}

	for {
		// Wait for the next batch.
		batchSize, err := leader.offline.ReceiveUint32()
		if err != nil {
			return err
		}
		err = nw.tripleBatch(batchSize)
		if err != nil {
			fmt.Printf("tripleBatch: %v\n", err)
			return err
		}
	}
}

func (nw *Network) tripleBatch(size int) error {
	self := nw.self
	fmt.Printf("%v: new triple batch: size=%v\n", self, size)

	words := (size + 63) / 64

	a := make([]uint64, words)
	b := make([]uint64, words)
	c := make([]uint64, words)

	// Sample random local shares
	var buf [16]byte
	for i := 0; i < words; i++ {
		_, err := rand.Read(buf[:])
		if err != nil {
			return err
		}
		a[i] = bo.Uint64(buf[0:])
		b[i] = bo.Uint64(buf[8:])
	}

	// Local term
	for i := 0; i < words; i++ {
		c[i] = a[i] & b[i]
	}

	// Cross terms
	for _, peer := range nw.peers {
		if peer.id == self.id {
			continue
		}

		sBits := make([]uint64, words)
		rBits := make([]uint64, words)
		u := make([]uint64, words)
		v := make([]uint64, words)

		if self.id < peer.id {

			// =================================================
			// Term 1: a_self & b_peer
			// self = sender
			// =================================================

			err := peer.iknpS.SendBits(size, sBits)
			if err != nil {
				return err
			}

			delta := peer.iknpS.Delta.Bit(0)

			// u = a ⊕ Δ
			if delta == 1 {
				for w := 0; w < words; w++ {
					u[w] = a[w] ^ ^uint64(0)
				}
			} else {
				copy(u, a)
			}

			// Send u
			if err := peer.SendBitsVec(u); err != nil {
				return err
			}

			// Receive v = b_peer
			if err := peer.ReceiveBitsVec(v); err != nil {
				return err
			}

			// sender share: s ⊕ (u & v)
			for w := 0; w < words; w++ {
				c[w] ^= sBits[w] ^ (u[w] & v[w])
			}

			// =================================================
			// Term 2: a_peer & b_self
			// self = receiver
			// =================================================

			err = peer.iknpR.ReceiveBits(b, rBits, size)
			if err != nil {
				return err
			}

			// send v = b_self
			if err := peer.SendBitsVec(b); err != nil {
				return err
			}

			// receive u = a_peer ⊕ Δ
			if err := peer.ReceiveBitsVec(u); err != nil {
				return err
			}

			// receiver share: r
			for w := 0; w < words; w++ {
				c[w] ^= rBits[w]
			}

		} else {

			// =================================================
			// Term 1: a_peer & b_self
			// self = receiver
			// =================================================

			err := peer.iknpR.ReceiveBits(b, rBits, size)
			if err != nil {
				return err
			}

			// send v = b_self
			if err := peer.SendBitsVec(b); err != nil {
				return err
			}

			// receive u = a_peer ⊕ Δ
			if err := peer.ReceiveBitsVec(u); err != nil {
				return err
			}

			for w := 0; w < words; w++ {
				c[w] ^= rBits[w]
			}

			// =================================================
			// Term 2: a_self & b_peer
			// self = sender
			// =================================================

			err = peer.iknpS.SendBits(size, sBits)
			if err != nil {
				return err
			}

			delta := peer.iknpS.Delta.Bit(0)

			if delta == 1 {
				for w := 0; w < words; w++ {
					u[w] = a[w] ^ ^uint64(0)
				}
			} else {
				copy(u, a)
			}

			if err := peer.SendBitsVec(u); err != nil {
				return err
			}

			if err := peer.ReceiveBitsVec(v); err != nil {
				return err
			}

			for w := 0; w < words; w++ {
				c[w] ^= sBits[w] ^ (u[w] & v[w])
			}
		}
	}

	fmt.Printf("-batch%v=%x,%x,%x\n", self.id, a[0], b[0], c[0])

	nw.pool.m.Lock()
	nw.pool.count += size
	nw.pool.m.Unlock()

	return nil
}

/**************************** Triple generation *****************************/

type TripleBatch struct {
	A []uint64 // bit-packed shares
	B []uint64
	C []uint64
}

type INetwork struct {
	id         int
	numParties int
	peers      []*Peer // index by party id
}

func randomUint64() uint64 {
	return 0
}

func randomBitVector(words int) []uint64 {
	return nil
}

func xorBitVector(a, b []uint64) []uint64 {
	return nil
}

func OTSend(peer *Peer, m0, m1 []uint64) {
}

func OTRecv(peer *Peer, flags []uint64) []uint64 {
	return nil
}

func copyOf(src []uint64) []uint64 {
	result := make([]uint64, len(src))
	copy(result, src)
	return result
}

func (nw *INetwork) GenerateTripleBatch(N int) *TripleBatch {

	words := (N + 63) / 64

	A := make([]uint64, words)
	B := make([]uint64, words)
	C := make([]uint64, words)

	// ------------------------------------------------
	// Step 1: Sample random local shares a_i, b_i
	// ------------------------------------------------

	for w := 0; w < words; w++ {
		A[w] = randomUint64()
		B[w] = randomUint64()
	}

	// ------------------------------------------------
	// Step 2: Local term a_i * b_i
	// ------------------------------------------------

	for w := 0; w < words; w++ {
		C[w] = A[w] & B[w] // bitwise AND
	}

	// ------------------------------------------------
	// Step 3: Cross terms with all other parties
	// ------------------------------------------------

	for j := 0; j < nw.numParties; j++ {

		if j == nw.id {
			continue
		}

		peer := nw.peers[j]

		if nw.id < j {
			// ----------------------------------------
			// Compute share of a_i * b_j
			// ----------------------------------------

			r := randomBitVector(words)

			m0 := r
			m1 := xorBitVector(r, A)

			// Sender role
			OTSend(peer, m0, m1)

			// Receiver role for opposite cross term
			recv := OTRecv(peer, B)

			// recv = s XOR (a_j * b_i)
			// accumulate cross share
			C = xorBitVector(C, recv)

			// add our share of first cross term
			C = xorBitVector(C, r)

		} else {
			// ----------------------------------------
			// Reverse roles
			// ----------------------------------------

			// Receiver role
			recv := OTRecv(peer, B)

			// Sender role
			r := randomBitVector(words)
			m0 := r
			m1 := xorBitVector(r, A)
			OTSend(peer, m0, m1)

			// accumulate cross shares
			C = xorBitVector(C, recv)
			C = xorBitVector(C, r)
		}
	}

	return &TripleBatch{
		A: A,
		B: B,
		C: C,
	}
}

// Now: XOR_i C_i = (XOR_i A_i) AND (XOR_i B_i)

/************************ Batched AND Using Triples *************************/

func (nw *INetwork) AndWithTriples(
	X, Y []uint64,
	T *TripleBatch,
) []uint64 {

	words := len(X)

	D := make([]uint64, words)
	E := make([]uint64, words)

	// --------------------------------------------
	// Step 1: Compute masked differences
	// d = x XOR a
	// e = y XOR b
	// --------------------------------------------

	for w := 0; w < words; w++ {
		D[w] = X[w] ^ T.A[w]
		E[w] = Y[w] ^ T.B[w]
	}

	// --------------------------------------------
	// Step 2: Open d and e
	// --------------------------------------------

	Dopen := nw.BroadcastXOR(D)
	Eopen := nw.BroadcastXOR(E)

	// --------------------------------------------
	// Step 3: Final local computation
	// z = c XOR (d & b) XOR (e & a) XOR (d & e)
	// --------------------------------------------

	Z := make([]uint64, words)

	for w := 0; w < words; w++ {
		Z[w] =
			T.C[w] ^
				(Dopen[w] & T.B[w]) ^
				(Eopen[w] & T.A[w]) ^
				(Dopen[w] & Eopen[w])
	}

	return Z
}

func (nw *INetwork) BroadcastXOR(local []uint64) []uint64 {

	result := copyOf(local)

	for _, peer := range nw.peers {
		if peer.id == nw.id {
			continue
		}

		peer.SendBitsVec(local)
	}

	for _, peer := range nw.peers {
		if peer.id == nw.id {
			continue
		}

		recv := make([]uint64, len(local))

		peer.ReceiveBitsVec(recv)
		result = xorBitVector(result, recv)
	}

	return result
}
