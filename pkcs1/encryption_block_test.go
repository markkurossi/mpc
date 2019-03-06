//
// encryption_block_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package pkcs1

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestEncryptionBlock(t *testing.T) {
	data := []byte{'h', 'e', 'l', 'l', 'o'}

	_, err := NewEncryptionBlock(BT0, 2048/8, data)
	if err == nil {
		t.Fatal("BT0 succeeded")
	}

	_, err = NewEncryptionBlock(BT1, 2048/8, data)
	if err != nil {
		t.Fatalf("Failed to create BT1: %s", err)
	}

	_, err = NewEncryptionBlock(BT2, 2048/8, data)
	if err != nil {
		t.Fatalf("Failed to create BT2: %s", err)
	}

	block, err := NewEncryptionBlock(BT2, len(data)+MinPadLen+3-1, data)
	if err == nil {
		fmt.Printf("Encoded:\n%s", hex.Dump(block))
		t.Fatal("Too long data encoded")
	}
}
