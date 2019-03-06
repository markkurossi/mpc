//
// encryption_block_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package pkcs1

import (
	"bytes"
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

	block, err := NewEncryptionBlock(BT1, 2048/8, data)
	if err != nil {
		t.Fatalf("Failed to create BT1: %s", err)
	}
	parsed, err := ParseEncryptionBlock(block)
	if err != nil {
		t.Fatalf("Failed to parse BT1 block: %s", err)
	}
	if bytes.Compare(data, parsed) != 0 {
		t.Fatalf("Parsed invalid BT1 data")
	}

	block, err = NewEncryptionBlock(BT2, 2048/8, data)
	if err != nil {
		t.Fatalf("Failed to create BT2: %s", err)
	}
	parsed, err = ParseEncryptionBlock(block)
	if err != nil {
		t.Fatalf("Failed to parse BT2 block: %s", err)
	}
	if bytes.Compare(data, parsed) != 0 {
		t.Fatalf("Parsed invalid BT2 data")
	}

	block, err = NewEncryptionBlock(BT2, len(data)+MinPadLen+3-1, data)
	if err == nil {
		fmt.Printf("Encoded:\n%s", hex.Dump(block))
		t.Fatal("Too long data encoded")
	}
}
