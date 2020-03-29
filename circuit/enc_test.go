//
// enc_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package circuit

import (
	"crypto/aes"
	"crypto/rand"
	"testing"

	"github.com/markkurossi/mpc/ot"
)

func TestEnc(t *testing.T) {
	a, _ := ot.NewLabel(rand.Reader)
	b, _ := ot.NewLabel(rand.Reader)
	c, _ := ot.NewLabel(rand.Reader)
	tweak := uint32(42)
	var key [32]byte

	cipher, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatalf("Failed to create cipher: %s", err)
	}

	encrypted := encrypt(cipher, a, b, c, tweak)
	if err != nil {
		t.Fatalf("Encrypt failed: %s", err)
	}

	plain, err := decrypt(cipher, a, b, tweak, encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %s", err)
	}

	if !c.Equal(plain) {
		t.Fatalf("Encrypt-decrypt failed")
	}
}
