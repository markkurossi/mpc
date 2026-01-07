//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"
)

func TestCOTExpand(t *testing.T) {
	c0, c1 := NewPipe()
	oti0 := NewCO(rand.Reader)
	oti1 := NewCO(rand.Reader)

	errCh := make(chan error)

	const N = 8

	b := randomBools(N)
	if false {
		for i := range b {
			b[i] = true
		}
	}

	var sent []Label
	var rcvd []Label
	var sender *COTSender

	go func() {
		oti1.InitReceiver(c1)
		iknp, err := NewCOTReceiver(oti1, c1, rand.Reader)
		if err != nil {
			errCh <- err
			return
		}

		rcvd, err = iknp.Receive(b)
		errCh <- err
	}()

	go func() {
		oti0.InitSender(c0)
		var err error
		sender, err = NewCOTSender(oti0, c0, rand.Reader, nil)
		if err != nil {
			errCh <- err
		}

		sent, err = sender.Send(N)
		errCh <- err
	}()

	for i := 0; i < 2; i++ {
		err := <-errCh
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(sent) != len(rcvd) {
		t.Fatalf("len(sent)=%v != len(rcvd)=%v", len(sent), len(rcvd))
	}
	if len(sent) != N {
		t.Fatalf("sent %v but N was %v", len(sent), N)
	}

	for i, b0 := range sent {
		b1 := b0
		if b[i] {
			b1.Xor(sender.Delta)
		}

		if !rcvd[i].Equal(b1) {
			fmt.Printf("OT[%d]:\n", i)
			fmt.Printf(" - sent: %v\n", b0)
			fmt.Printf(" - rcvd: %v\n", rcvd[i])
			if b[i] {
				fmt.Printf(" - ⊕Δ  : %v\n", b1)
			}
			t.Errorf("OT[%d] mismatch: %v != %v", i, rcvd[i], b1)
		}
	}
}

// Measure IKNP setup.
func BenchmarkCOTSetup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c0, c1 := NewPipe()
		oti0 := NewCO(rand.Reader)
		oti1 := NewCO(rand.Reader)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			oti0.InitSender(c0)
			_, err := NewCOTSender(oti0, c0, rand.Reader, nil)
			if err != nil {
				panic(err)
			}
		}()

		go func() {
			defer wg.Done()
			oti1.InitReceiver(c1)
			_, err := NewCOTReceiver(oti1, c1, rand.Reader)
			if err != nil {
				panic(err)
			}
		}()

		wg.Wait()
	}
}

func BenchmarkCOTExpand1K(b *testing.B)   { benchmarkCOTExpand(b, 1000) }
func BenchmarkCOTExpand10K(b *testing.B)  { benchmarkCOTExpand(b, 10000) }
func BenchmarkCOTExpand100K(b *testing.B) { benchmarkCOTExpand(b, 100000) }

func benchmarkCOTExpand(b *testing.B, N int) {
	c0, c1 := NewPipe()
	oti0 := NewCO(rand.Reader)
	oti1 := NewCO(rand.Reader)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		oti1.InitReceiver(c1)
		iknp, err := NewCOTReceiver(oti1, c1, rand.Reader)
		if err != nil {
			panic(err)
		}

		flags := randomBools(N)
		for i := 0; i < b.N; i++ {
			_, err := iknp.Receive(flags)
			if err != nil {
				panic(err)
			}
		}
	}()

	oti0.InitSender(c0)
	iknp, err := NewCOTSender(oti0, c0, rand.Reader, nil)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := iknp.Send(N)
		if err != nil {
			panic(err)
		}
	}

	wg.Wait()
}

func randomBools(n int) []bool {
	buf := make([]byte, (n+7)/8)
	rand.Read(buf)
	out := make([]bool, n)
	for i := 0; i < n; i++ {
		out[i] = ((buf[i/8] >> uint(i%8)) & 1) == 1
	}
	return out
}

var xorTests = []struct {
	a []byte
	b []byte
	r []byte
}{
	{
		a: []byte{0b00000001, 0b00000010, 0b00000100, 0b00001000},
		b: []byte{0xff, 0xff, 0xff, 0xff},
		r: []byte{0b11111110, 0b11111101, 0b11111011, 0b11110111},
	},
}

func TestXORArray(t *testing.T) {
	for idx, test := range xorTests {
		tmp := make([]byte, len(test.a))
		copy(tmp, test.a)
		xor(tmp, test.b)
		if !bytes.Equal(tmp, test.r) {
			t.Errorf("test-%d: got %x, expected %x\n", idx, tmp, test.r)
		}
	}
}
