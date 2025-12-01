package vole

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/markkurossi/mpc/p2p"
)

// TestPackedPathFallback runs the VOLE flow with oti==nil fallback (channel shim).
func TestPackedPathFallback(t *testing.T) {
	p := hexP256()
	m := 32
	xs, ys := randVectors(t, m, p)

	c0, c1 := p2p.Pipe()
	ext0 := NewExt(nil, c0, SenderRole)
	ext1 := NewExt(nil, c1, ReceiverRole)
	_ = ext0.Setup(rand.Reader)
	_ = ext1.Setup(rand.Reader)

	chDone := make(chan error, 2)
	var rs []*big.Int
	var us []*big.Int
	go func() {
		var err error
		rs, err = ext0.MulSender(xs, p)
		chDone <- err
	}()
	go func() {
		var err error
		us, err = ext1.MulReceiver(ys, p)
		chDone <- err
	}()
	if err := <-chDone; err != nil {
		t.Fatalf("sender error: %v", err)
	}
	if err := <-chDone; err != nil {
		t.Fatalf("receiver error: %v", err)
	}
	// Verify u-r == x*y
	for i := 0; i < m; i++ {
		left := new(big.Int).Sub(us[i], rs[i])
		left.Mod(left, p)
		right := new(big.Int).Mul(xs[i], ys[i])
		right.Mod(right, p)
		if left.Cmp(right) != 0 {
			t.Fatalf("mismatch at %d", i)
		}
	}
}

// Helper to generate random vectors
func randVectors(t *testing.T, m int, p *big.Int) ([]*big.Int, []*big.Int) {
	xs := make([]*big.Int, m)
	ys := make([]*big.Int, m)
	for i := 0; i < m; i++ {
		xs[i], _ = randomFieldElementFromCrypto(rand.Reader, p)
		ys[i], _ = randomFieldElementFromCrypto(rand.Reader, p)
	}
	return xs, ys
}

func hexP256() *big.Int {
	p, _ := new(big.Int).SetString("ffffffff00000001000000000000000000000000ffffffffffffffffffffffff", 16)
	return p
}
