//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"crypto/rand"
	"testing"
)

func TestBitCOT(t *testing.T) {

	const malicious = false
	const n = 1024

	choices := randomUint64Array(n)
	sent := make([]uint64, (n+63)/64)
	received := make([]uint64, (n+63)/64)

	sPipe, rPipe := NewPipe()

	done := make(chan error)
	var deltaBit uint

	go func(pipe *Pipe) {
		co := NewCO(rand.Reader)
		err := co.InitSender(pipe)
		if err != nil {
			done <- err
			return
		}
		iknp, err := NewIKNPSender(co, pipe, rand.Reader, nil)
		if err != nil {
			done <- err
			return
		}

		// Extract delta bit for column 0.
		deltaBit = iknp.Delta.Bit(0)

		done <- iknp.SendBits(n, sent)
	}(sPipe)

	go func(pipe *Pipe) {
		co := NewCO(rand.Reader)
		err := co.InitReceiver(pipe)
		if err != nil {
			done <- err
			return
		}
		iknp, err := NewIKNPReceiver(co, pipe, rand.Reader)
		if err != nil {
			done <- err
			return
		}
		err = iknp.ReceiveBits(choices, received, n)
		done <- err
	}(rPipe)

	// Wait sender and receiver.
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}

	// Verify correlated OT invariant.
	for i := 0; i < n; i++ {

		cbit := uint(choices[i/64] >> (i % 64) & 1)
		sbit := (sent[i/64] >> (i % 64)) & 1
		rbit := (received[i/64] >> (i % 64)) & 1

		expected := uint64(0)
		if deltaBit&cbit == 1 {
			expected = 1
		}

		if (sbit ^ rbit) != expected {
			t.Fatalf("bit %d mismatch: s=%d r=%d c=%v Δ=%v",
				i, sbit, rbit, choices[i], deltaBit)
		}
	}
}

func BenchmarkBitCOT(b *testing.B) {

	const n = 1024

	choices := randomUint64Array(n)
	sent := make([]uint64, (n+63)/64)
	received := make([]uint64, (n+63)/64)

	sPipe, rPipe := NewPipe()

	done := make(chan error, 2)

	go func(pipe *Pipe) {
		co := NewCO(rand.Reader)
		if err := co.InitSender(pipe); err != nil {
			done <- err
			pipe.Close()
			return
		}
		iknp, err := NewIKNPSender(co, pipe, rand.Reader, nil)
		if err != nil {
			done <- err
			pipe.Close()
			return
		}
		for i := 0; i < b.N; i++ {
			err = iknp.SendBits(n, sent)
			if err != nil {
				done <- err
				pipe.Close()
				return
			}
		}
		done <- nil
	}(sPipe)

	go func(pipe *Pipe) {
		co := NewCO(rand.Reader)
		if err := co.InitReceiver(pipe); err != nil {
			done <- err
			pipe.Close()
			return
		}
		iknp, err := NewIKNPReceiver(co, pipe, rand.Reader)
		if err != nil {
			done <- err
			pipe.Close()
			return
		}
		for i := 0; i < b.N; i++ {
			err = iknp.ReceiveBits(choices, received, n)
			if err != nil {
				done <- err
				pipe.Close()
				return
			}
		}
		done <- nil
	}(rPipe)

	// Wait both sides.
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			b.Fatal(err)
		}
	}
}

func randomUint64Array(n int) []uint64 {
	l := (n + 63) / 64
	result := make([]uint64, l)

	for i := 0; i < l; i++ {
		var buf [8]byte
		_, err := rand.Read(buf[:])
		if err != nil {
			panic(err)
		}

		result[i] = bo.Uint64(buf[:])
	}

	return result
}
