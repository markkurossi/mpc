//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

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

// TriplePool implements beaver triple pool.
type TriplePool struct {
	m       sync.Mutex
	c       *sync.Cond
	triples *Triples
}

// Triples contain a batch of beaver triples.
type Triples struct {
	Words int
	A     []uint64
	B     []uint64
	C     []uint64
}

// EnsureCapacity ensures that the triple batch has capacity for the
// specified number of words.
func (triples *Triples) EnsureCapacity(words int) {
	triples.A = expand(triples.A, words)
	triples.B = expand(triples.B, words)
	triples.C = expand(triples.C, words)
}

// Clear clears triples.
func (triples *Triples) Clear() {
	triples.Words = 0
	clear(triples.A)
	clear(triples.B)
	clear(triples.C)
}

// Append appends n (or maximum of n+63) triples from the src triple
// batch. The function returns the number of triples appended. The
// return value is smaller than n if the src batch did not have that
// many triples.  The function always appends full uint64 words.
func (triples *Triples) Append(src *Triples, n int) int {
	// Round n up to word boundary.
	words := (n + 63) / 64
	if words > src.Words {
		words = src.Words
	}

	triples.EnsureCapacity(triples.Words + words)

	copy(triples.A[triples.Words:], src.A[:words])
	copy(triples.B[triples.Words:], src.B[:words])
	copy(triples.C[triples.Words:], src.C[:words])

	triples.Words += words

	copy(src.A[0:], src.A[words:])
	copy(src.B[0:], src.B[words:])
	copy(src.C[0:], src.C[words:])

	src.Words -= words

	clear(src.A[src.Words:])
	clear(src.B[src.Words:])
	clear(src.C[src.Words:])

	return words * 64
}

// NewTriplePool creates a new beaver triple pool.
func NewTriplePool() *TriplePool {
	pool := &TriplePool{
		triples: new(Triples),
	}

	pool.c = sync.NewCond(&pool.m)

	return pool
}

// Get gets count triples from the triple pool and returns them in the
// triples. The function resizes triples if needed. If the triple pool
// does not have count triples, the function waits until the triple
// generation offline process produces the required amount of new
// triples.
func (pool *TriplePool) Get(count int, triples *Triples) {
	words := (count + 63) / 64
	if words > len(triples.A) {
		// Resize result arrays.
		triples.A = make([]uint64, count)
		triples.B = make([]uint64, count)
		triples.C = make([]uint64, count)
	}

	for ofs := 0; ofs < count; {
		pool.m.Lock()
		for pool.triples.Words == 0 {
			pool.c.Wait()
		}
		ofs += triples.Append(pool.triples, count-ofs)
		pool.c.Signal()
		pool.m.Unlock()
	}
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
	batchSize := 1024

	for {
		nw.pool.m.Lock()
		for nw.pool.triples.Words > lowWaterMark {
			nw.pool.c.Wait()
		}
		nw.pool.m.Unlock()

		msg := int(msgTriple)<<24 | (batchSize & 0x00ffffff)

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
		batchSize = 4096
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
	if false {
		fmt.Printf("%v: new triple batch: size=%v\n", self, size)
	}

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
			if err := peer.SendBitvec(peer.offline, u); err != nil {
				return err
			}

			// Receive v = b_peer
			if err := peer.ReceiveBitvec(peer.offline, v); err != nil {
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
			if err := peer.SendBitvec(peer.offline, b); err != nil {
				return err
			}

			// receive u = a_peer ⊕ Δ
			if err := peer.ReceiveBitvec(peer.offline, u); err != nil {
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
			if err := peer.SendBitvec(peer.offline, b); err != nil {
				return err
			}

			// receive u = a_peer ⊕ Δ
			if err := peer.ReceiveBitvec(peer.offline, u); err != nil {
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

			if err := peer.SendBitvec(peer.offline, u); err != nil {
				return err
			}

			if err := peer.ReceiveBitvec(peer.offline, v); err != nil {
				return err
			}

			for w := 0; w < words; w++ {
				c[w] ^= sBits[w] ^ (u[w] & v[w])
			}
		}
	}

	if false {
		fmt.Printf("-batch%v=%x,%x,%x\n", self.id, a[0], b[0], c[0])
	}

	nw.pool.m.Lock()
	nw.pool.triples.Append(&Triples{
		Words: words,
		A:     a,
		B:     b,
		C:     c,
	}, size)
	nw.pool.c.Signal()
	nw.pool.m.Unlock()

	return nil
}
