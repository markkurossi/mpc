//
// label_test.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package ot

import (
	"testing"
)

func TestLabel(t *testing.T) {
	label := &Label{
		d0: 0xffffffffffffffff,
		d1: 0xffffffffffffffff,
	}

	label.SetS(true)
	if label.d0 != 0xffffffffffffffff {
		t.Fatal("Failed to set S-bit")
	}

	label.SetS(false)
	if label.d0 != 0x7fffffffffffffff {
		t.Fatalf("Failed to clear S-bit: %x", label.d0)
	}
}
