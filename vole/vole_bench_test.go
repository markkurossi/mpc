// vole_bench_test.go
package vole

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

// helper: create a VOLE pair ext (sender & receiver). If concrete base OT constructor
// is available (ot.NewCO), use it; otherwise return exts with oti==nil -> shim mode.
func newVOLEPair(connA, connB *p2p.Conn) (*Ext, *Ext, error) {
	// Try to create concrete base-OT instances. If ot.NewCO is unavailable at compile
	// time this will not compile; your repo already has ot.NewCO used in tests earlier.
	otiA := ot.NewCO(rand.Reader)
	otiB := ot.NewCO(rand.Reader)

	extA := NewExt(otiA, connA, SenderRole)
	extB := NewExt(otiB, connB, ReceiverRole)

	if err := extA.Setup(rand.Reader); err != nil {
		return nil, nil, fmt.Errorf("extA setup: %w", err)
	}
	if err := extB.Setup(rand.Reader); err != nil {
		return nil, nil, fmt.Errorf("extB setup: %w", err)
	}
	return extA, extB, nil
}

func BenchmarkVOLEEndToEnd(b *testing.B) {
	sizes := []int{1, 8, 64, 256, 1024}

	for _, m := range sizes {
		b.Run(fmt.Sprintf("m=%d", m), func(b *testing.B) {

			// Pre-generate random inputs outside the loop
			p := hexP256()
			xs := make([]*big.Int, m)
			ys := make([]*big.Int, m)
			for i := 0; i < m; i++ {
				xs[i], _ = randomFieldElementFromCrypto(rand.Reader, p)
				ys[i], _ = randomFieldElementFromCrypto(rand.Reader, p)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {

				// Fresh pipe every iteration
				c0, c1 := p2p.Pipe()

				// IMPORTANT: close both ends when done
				// prevents goroutine leaks & deadlocks
				defer c0.Close()
				defer c1.Close()

				// Fresh VOLE setup
				extS := NewExt(nil, c0, SenderRole)
				extR := NewExt(nil, c1, ReceiverRole)
				if err := extS.Setup(rand.Reader); err != nil {
					b.Fatalf("setup S: %v", err)
				}
				if err := extR.Setup(rand.Reader); err != nil {
					b.Fatalf("setup R: %v", err)
				}

				done := make(chan error, 2)

				go func() {
					_, err := extS.MulSender(xs, p)
					done <- err
				}()
				go func() {
					_, err := extR.MulReceiver(ys, p)
					done <- err
				}()

				if err := <-done; err != nil {
					b.Fatalf("sender err: %v", err)
				}
				if err := <-done; err != nil {
					b.Fatalf("receiver err: %v", err)
				}
			}
		})
	}
}

// If you have GenerateBeaverTriplesOTBatch exposed, benchmark it similarly.
// Adjust the signature usage to match your repo.
func BenchmarkTriplegenEndToEnd(b *testing.B) {
	// If GenerateBeaverTriplesOTBatch exists in your repo and takes
	// (conn *p2p.Conn, oti ot.OT, id int, triples []*Triple) ([]*Share, error)
	// then you can adapt this benchmark. For now, this is a template.
	b.Skip("Uncomment and adapt BenchmarkTriplegenEndToEnd to your GenerateBeaverTriplesOTBatch signature")
}
