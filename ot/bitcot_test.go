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

	choices := randomBoolArray(n)
	sent := make([]uint64, (n+63)/64)
	received := make([]uint64, (n+63)/64)

	sPipe, rPipe := NewPipe()

	done := make(chan error)
	var deltaBit bool

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

		// Extract delta bit for column 0
		deltaBit = iknp.Delta.Bit(0) == 1

		err = iknp.SendBits(n, sent)

		done <- err

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
		err = iknp.ReceiveBits(choices, received)
		done <- err
	}(rPipe)

	// Wait sender and receiver.
	for i := 0; i < 2; i++ {
		err := <-done
		if err != nil {
			t.Fatal(err)
		}
	}

	// Verify correlated OT invariant
	for i := 0; i < n; i++ {

		sbit := (sent[i/64] >> (i % 64)) & 1
		rbit := (received[i/64] >> (i % 64)) & 1

		expected := uint64(0)
		if deltaBit && choices[i] {
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

	// Pre-generate choices outside timing
	choices := randomBoolArray(n)
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
			err = iknp.ReceiveBits(choices, received)
			if err != nil {
				done <- err
				pipe.Close()
				return
			}
		}
		done <- nil
	}(rPipe)

	// Wait both sides
	for j := 0; j < 2; j++ {
		if err := <-done; err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBitCOT_ExtensionOnly(b *testing.B) {

	const n = 1 << 16 // 65536 bits

	choices := randomBoolArray(n)
	sent := make([]uint64, (n+63)/64)
	received := make([]uint64, (n+63)/64)

	sPipe, rPipe := NewPipe()

	done := make(chan error, 2)

	// Initialize once
	var sender *IKNPSender
	var receiver *IKNPReceiver

	go func() {
		co := NewCO(rand.Reader)
		co.InitSender(sPipe)
		sender, _ = NewIKNPSender(co, sPipe, rand.Reader, nil)
		done <- nil
	}()

	go func() {
		co := NewCO(rand.Reader)
		co.InitReceiver(rPipe)
		receiver, _ = NewIKNPReceiver(co, rPipe, rand.Reader)
		done <- nil
	}()

	<-done
	<-done

	b.ResetTimer()

	done2 := make(chan error, 2)

	go func() {
		for i := 0; i < b.N; i++ {
			err := sender.SendBits(n, sent)
			if err != nil {
				done2 <- err
				return
			}
		}
		done2 <- nil
	}()

	go func() {
		for i := 0; i < b.N; i++ {
			err := receiver.ReceiveBits(choices, received)
			if err != nil {
				done2 <- err
				return
			}
		}
		done2 <- nil
	}()

	if err := <-done2; err != nil {
		b.Fatal(err)
	}
	if err := <-done2; err != nil {
		b.Fatal(err)
	}
}

func randomBoolArray(n int) []bool {
	result := make([]bool, n)

	buf := make([]byte, (n+7)/8)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	for i := 0; i < n; i++ {
		result[i] = ((buf[i/8] >> (i % 8)) & 1) == 1
	}

	return result
}
