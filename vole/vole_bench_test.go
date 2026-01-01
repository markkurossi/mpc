//
// Copyright (c) 2025 Markku Rossi
//
// All rights reserved.
//

package vole

import (
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

var (
	p = elliptic.P256().Params().P
)

func BenchmarkVOLEEndToEnd(b *testing.B) {
	sizes := []int{1, 8, 64, 256, 1024}

	for _, m := range sizes {
		b.Run(fmt.Sprintf("m=%d", m), func(b *testing.B) {

			// Pre-generate random inputs outside the loop
			xs := make([]*big.Int, m)
			ys := make([]*big.Int, m)
			for i := 0; i < m; i++ {
				xs[i], _ = randomFieldElementFromCrypto(rand.Reader, p)
				ys[i], _ = randomFieldElementFromCrypto(rand.Reader, p)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				// Fresh pipe every iteration
				c0, c1 := p2p.Pipe()

				done := make(chan error, 2)

				go func() {
					send, err := NewSender(ot.NewCO(rand.Reader), c0,
						rand.Reader)
					if err != nil {
						done <- err
						return
					}
					_, err = send.Mul(xs, p)
					done <- err
				}()
				go func() {
					recv, err := NewReceiver(ot.NewCO(rand.Reader), c1,
						rand.Reader)
					if err != nil {
						done <- err
						return
					}
					_, err = recv.Mul(ys, p)
					done <- err
				}()

				if err := <-done; err != nil {
					b.Fatalf("sender err: %v", err)
				}
				if err := <-done; err != nil {
					b.Fatalf("receiver err: %v", err)
				}
				c0.Close()
				c1.Close()
			}
		})
	}
}
