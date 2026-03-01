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
	Count int
	A     []uint64
	B     []uint64
	C     []uint64
}

// Clear clears triples.
func (triples *Triples) Clear() {
	triples.Count = 0
	clear(triples.A)
	clear(triples.B)
	clear(triples.C)
}

// Append appends maximum n triples from the src triple batch. The
// function returns the number of triples appended. The return value
// is smaller than n if the src batch did not have that many triples.
func (triples *Triples) Append(src *Triples, n int) int {
	if n > src.Count {
		n = src.Count
	}
	ofs := triples.Count / 64
	shift := triples.Count % 64

	// Words and trailing bits.
	words := n / 64
	tail := n % 64

	tailMask := uint64(0xffffffffffffffff)
	tailMask >>= 64 - tail

	trailMask := uint64(0xffffffffffffffff)
	trailMask <<= tail

	if shift == 0 {
		for i := 0; i < words; i++ {
			triples.A[ofs] = src.A[i]
			triples.B[ofs] = src.B[i]
			triples.C[ofs] = src.C[i]
			ofs++
		}
		if tail != 0 {
			triples.A[ofs] = src.A[words] & tailMask
			triples.B[ofs] = src.B[words] & tailMask
			triples.C[ofs] = src.C[words] & tailMask
		}
	} else {
		for i := 0; i < words; i++ {
			triples.A[ofs] |= src.A[i] << shift
			triples.B[ofs] |= src.B[i] << shift
			triples.C[ofs] |= src.C[i] << shift

			triples.A[ofs+1] = src.A[i] >> (64 - shift)
			triples.B[ofs+1] = src.B[i] >> (64 - shift)
			triples.C[ofs+1] = src.C[i] >> (64 - shift)

			ofs++
		}
		if tail != 0 {
			a := src.A[words] & tailMask
			b := src.B[words] & tailMask
			c := src.C[words] & tailMask

			triples.A[ofs] |= a << shift
			triples.B[ofs] |= b << shift
			triples.C[ofs] |= c << shift

			if 64-shift < tail {
				triples.A[ofs+1] = a >> (64 - shift)
				triples.B[ofs+1] = b >> (64 - shift)
				triples.C[ofs+1] = c >> (64 - shift)
			}
		}
	}

	// Compact src arrays.
	if src.Count == n {
		src.Clear()
	} else {
		src.Count -= n
		remainingWords := src.Count / 64

		if tail == 0 {
			copy(src.A[0:], src.A[words:])
			copy(src.B[0:], src.B[words:])
			copy(src.C[0:], src.C[words:])

			clear(src.A[remainingWords:])
			clear(src.B[remainingWords:])
			clear(src.C[remainingWords:])
		} else {
			for i := 0; i*64 < src.Count; i++ {
				src.A[i] = src.A[words+i] >> tail
				src.B[i] = src.B[words+i] >> tail
				src.C[i] = src.C[words+i] >> tail

				if i*64+(64-tail) < src.Count {
					src.A[i] |= (src.A[words+i+1] & tailMask) << (64 - tail)
					src.B[i] |= (src.B[words+i+1] & tailMask) << (64 - tail)
					src.C[i] |= (src.C[words+i+1] & tailMask) << (64 - tail)
				}
			}
		}
	}

	triples.Count += n

	return n
}

// NewTriplePool creates a new beaver triple pool.
func NewTriplePool() *TriplePool {
	pool := &TriplePool{}

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
		for pool.triples.Count == 0 {
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
	batchSize := 128

	for {
		nw.pool.m.Lock()
		for nw.pool.triples.Count > lowWaterMark {
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
	nw.pool.triples.Append(&Triples{
		Count: size,
		A:     a,
		B:     b,
		C:     c,
	}, size)
	nw.pool.c.Signal()
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
