package otext

import (
	"crypto/rand"
	"sync"
	"testing"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

//
// --- Benchmark Setup -----------------------------------------------------
//

func makeFlags(n int) []bool {
	return randomBools(n)
}

//
// --- Benchmarks ----------------------------------------------------------
//

// Only measure setup
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
			ext := NewIKNPExt(oti0, c0, SenderRole)
			if err := ext.Setup(rand.Reader); err != nil {
				panic(err)
			}
		}()

		go func() {
			defer wg.Done()
			oti1.InitReceiver(c1)
			ext := NewIKNPExt(oti1, c1, ReceiverRole)
			if err := ext.Setup(rand.Reader); err != nil {
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
		ext := NewIKNPExt(oti1, c1, ReceiverRole)
		if err := ext.Setup(rand.Reader); err != nil {
			panic(err)
		}

		flags := makeFlags(N)
		for i := 0; i < b.N; i++ {
			_, err := ext.ExpandReceive(flags)
			if err != nil {
				panic(err)
			}
		}
	}()

	// --- Sender measured path ---
	oti0.InitSender(c0)
	ext := NewIKNPExt(oti0, c0, SenderRole)
	if err := ext.Setup(rand.Reader); err != nil {
		panic(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ext.ExpandSend(N)
		if err != nil {
			panic(err)
		}
	}

	wg.Wait()
}
