//
// Copyright (c) 2026 Markku Rossi
//
// All rights reserved.
//

package gmw

import (
	"testing"
)

func makeTriple() *Triples {
	return &Triples{
		Count: 128,
		A:     []uint64{0x0807060504030201, 0x1817161514131211},
		B:     []uint64{0x0807060504030201, 0x1817161514131211},
		C:     []uint64{0x0807060504030201, 0x1817161514131211},
	}
}

func TestTriples(t *testing.T) {
	from := makeTriple()
	to := &Triples{
		Count: 0,
		A:     make([]uint64, 8),
		B:     make([]uint64, 8),
		C:     make([]uint64, 8),
	}

	// Append full words.
	n := to.Append(from, 64)
	if n != 64 {
		t.Errorf("invalid return value %v for Append", n)
	}
	if to.A[0] != 0x0807060504030201 {
		t.Errorf("invalid to.A[0]: %v", to.A[0])
	}
	if from.A[0] != 0x1817161514131211 {
		t.Errorf("invalid from.A[0]: %v", to.A[0])
	}

	// Append one byte.

	from = makeTriple()
	to.Clear()

	n = to.Append(from, 8)
	if n != 8 {
		t.Errorf("invalid return value %v from Append", n)
	}
	if to.A[0] != 1 {
		t.Errorf("invalid to.A[0]: %v", to.A[0])
	}

	if from.Count != 120 {
		t.Errorf("invalid from.Count %v", from.Count)
	}
	if from.A[0] != 0x1108070605040302 {
		t.Errorf("invalid from.A[0]: %x", from.A[0])
	}
	if from.A[1] != 0x0018171615141312 {
		t.Errorf("invalid from.A[1]: %x", from.A[1])
	}

	// Append 64 bits 1 bit at a time.

	from = makeTriple()
	if from.A[0] != 0x0807060504030201 {
		t.Errorf("invalid from triple")
	}
	to.Clear()
	if to.A[0] != 0 {
		t.Errorf("invalid to triple")
	}

	for i := 0; i < 64; i++ {
		n := to.Append(from, 1)
		if n != 1 {
			t.Fatalf("to.Append=%v", n)
		}
		if from.Count != 128-i-1 {
			t.Fatalf("from.Count %v", from.Count)
		}
		if to.Count != i+1 {
			t.Fatalf("to.Count %v", to.Count)
		}
	}
	if from.Count != 64 {
		t.Errorf("invalid from.Count %v", from.Count)
	}
	if from.A[0] != 0x1817161514131211 {
		t.Errorf("from.A[0]=%016x", from.A[0])
	}
	if to.Count != 64 {
		t.Errorf("invalid to.Count %v", to.Count)
	}
	if to.A[0] != 0x0807060504030201 {
		t.Errorf("to.A[0]=%016x", to.A[0])
	}

}
