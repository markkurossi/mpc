//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.

package gmw

import (
	"crypto/rand"
	"encoding/binary"
	"sync"

	"github.com/markkurossi/mpc/p2p"
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
	conn  *p2p.Conn
	m     sync.Mutex
	c     *sync.Cond
	count int
}

func NewTriplePool() *TriplePool {
	pool := &TriplePool{}

	pool.c = sync.NewCond(&pool.m)

	return pool
}

func (nw *Network) sender() error {
	batchSize := 128

	nw.pool.m.Lock()
	for {
		for nw.pool.count > lowWaterMark {
			nw.pool.c.Wait()
		}
		nw.pool.m.Unlock()

		msg := int(msgTriple)<<24 | (batchSize & 0x00ffffff)

		err := nw.pool.conn.SendUint32(msg)
		if err != nil {
			return err
		}

		words := (batchSize + 63) / 64

		// XXX cache these
		a := make([]uint64, words)
		b := make([]uint64, words)
		c := make([]uint64, words)

		// Sample random local shares a_i, b_i.
		var buf [16]byte
		for i := 0; i < words; i++ {
			_, err = rand.Read(buf[:])
			if err != nil {
				return err
			}
			a[i] = bo.Uint64(buf[0:])
			b[i] = bo.Uint64(buf[8:])
		}

		// Compute local term a_i*b_i.
		for i := 0; i < words; i++ {
			c[i] = a[i] & b[i]
		}

		// Increase batch size after first batch.
		batchSize = 1024
	}
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

		peer.send(local)
	}

	for _, peer := range nw.peers {
		if peer.id == nw.id {
			continue
		}

		recv := peer.receive()
		result = xorBitVector(result, recv)
	}

	return result
}
