//
// Copyright (c) 2025 Markku Rossi
//
// All rights reserved.
//

package vole

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/markkurossi/mpc/ot"
	"github.com/markkurossi/mpc/p2p"
)

// P-256 modulus
var p256P, _ = new(big.Int).SetString(
	"ffffffff00000001000000000000000000000000ffffffffffffffffffffffff",
	16,
)

func TestVOLEBasic(t *testing.T) {
	const m = 20

	// random x,y vectors
	xs := make([]*big.Int, m)
	ys := make([]*big.Int, m)
	for i := 0; i < m; i++ {
		xs[i], _ = randomFieldElementFromCrypto(rand.Reader, p256P)
		ys[i], _ = randomFieldElementFromCrypto(rand.Reader, p256P)
	}

	// Create pipe
	c0, c1 := p2p.Pipe()

	var rs []*big.Int
	var us []*big.Int
	var err0, err1 error

	done := make(chan bool)

	// Sender goroutine
	go func() {
		var ext0 *Sender
		ext0, err0 = NewSender(ot.NewCO(rand.Reader), c0, rand.Reader)
		if err0 != nil {
			done <- true
			return
		}
		rs, err0 = ext0.Mul(xs, p256P)
		done <- true
	}()

	// Receiver goroutine
	go func() {
		var ext1 *Receiver
		ext1, err1 = NewReceiver(ot.NewCO(rand.Reader), c1, rand.Reader)
		if err1 != nil {
			done <- true
			return
		}
		us, err1 = ext1.Mul(ys, p256P)
		done <- true
	}()

	<-done
	<-done

	if err0 != nil {
		t.Fatalf("sender error: %v", err0)
	}
	if err1 != nil {
		t.Fatalf("receiver error: %v", err1)
	}

	if len(rs) != m || len(us) != m {
		t.Fatalf("length mismatch: len(rs)=%d len(us)=%d", len(rs), len(us))
	}

	// Check VOLE relation: u_i - r_i == x_i * y_i mod p
	for i := 0; i < m; i++ {
		left := new(big.Int).Sub(us[i], rs[i])
		left.Mod(left, p256P)

		right := new(big.Int).Mul(xs[i], ys[i])
		right.Mod(right, p256P)

		if left.Cmp(right) != 0 {
			t.Fatalf("VOLE relation mismatch at %d:\n u-r = %x\n x*y = %x",
				i, left, right)
		}
	}
}
