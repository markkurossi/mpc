//
// enc_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package main

import (
	"bytes"
	"crypto/aes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/markkurossi/mpc/ot"
)

func TestEnc(t *testing.T) {
	a, _ := ot.NewLabel(rand.Reader)
	b, _ := ot.NewLabel(rand.Reader)
	c, _ := ot.NewLabel(rand.Reader)
	tweak := uint32(42)

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

	if bytes.Compare(c.Bytes(), plain) != 0 {
		t.Fatalf("Encrypt-decrypt failed")
	} else {
		fmt.Printf("c:\n%s", hex.Dump(c.Bytes()))
		fmt.Printf("plain:\n%s", hex.Dump(plain))
	}
}
