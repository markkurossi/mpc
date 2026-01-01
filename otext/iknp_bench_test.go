//
// Copyright (c) 2025 Markku Rossi
//
// All rights reserved.
//

package otext

import (
	"crypto/rand"
	"sync"
	"testing"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

// Measure IKNP setup.
func BenchmarkIKNPSetup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c0, c1 := p2p.Pipe()
		oti0 := ot.NewCO(rand.Reader)
		oti1 := ot.NewCO(rand.Reader)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			oti0.InitSender(c0)
			_, err := NewIKNPSender(oti0, c0, rand.Reader)
			if err != nil {
				panic(err)
			}
		}()

		go func() {
			defer wg.Done()
			oti1.InitReceiver(c1)
			_, err := NewIKNPReceiver(oti1, c1, rand.Reader)
			if err != nil {
				panic(err)
			}
		}()

		wg.Wait()
	}
}

// Benchmark expansion with N OTs
func BenchmarkIKNPExpand1K(b *testing.B)   { benchmarkIKNPExpand(b, 1000) }
func BenchmarkIKNPExpand10K(b *testing.B)  { benchmarkIKNPExpand(b, 10000) }
func BenchmarkIKNPExpand100K(b *testing.B) { benchmarkIKNPExpand(b, 100000) }

func benchmarkIKNPExpand(b *testing.B, N int) {
	c0, c1 := p2p.Pipe()
	oti0 := ot.NewCO(rand.Reader)
	oti1 := ot.NewCO(rand.Reader)

	var wg sync.WaitGroup
	wg.Add(1)

	// --- Receiver goroutine: always servicing incoming U blocks ---
	go func() {
		defer wg.Done()

		oti1.InitReceiver(c1)
		iknp, err := NewIKNPReceiver(oti1, c1, rand.Reader)
		if err != nil {
			panic(err)
		}

		flags := randomBools(N)
		for i := 0; i < b.N; i++ {
			_, err := iknp.Expand(flags)
			if err != nil {
				panic(err)
			}
		}
	}()

	// --- Sender measured path ---
	oti0.InitSender(c0)
	iknp, err := NewIKNPSender(oti0, c0, rand.Reader)
	if err != nil {
		panic(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := iknp.Expand(N)
		if err != nil {
			panic(err)
		}
	}

	wg.Wait()
}
