package vole

import (
	"crypto/rand"
	"math/big"
	"testing"

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

	ext0 := NewExt(nil, c0, SenderRole)
	ext1 := NewExt(nil, c1, ReceiverRole)

	// Setup() now works with nil OT
	_ = ext0.Setup(rand.Reader)
	_ = ext1.Setup(rand.Reader)

	var rs []*big.Int
	var us []*big.Int
	var err0, err1 error

	done := make(chan bool)

	// Sender goroutine
	go func() {
		rs, err0 = ext0.MulSender(xs, p256P)
		done <- true
	}()

	// Receiver goroutine
	go func() {
		us, err1 = ext1.MulReceiver(ys, p256P)
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
