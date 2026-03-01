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
		Words: 2,
		A:     []uint64{0x0807060504030201, 0x1817161514131211},
		B:     []uint64{0x0807060504030201, 0x1817161514131211},
		C:     []uint64{0x0807060504030201, 0x1817161514131211},
	}
}

func TestTriples(t *testing.T) {
	from := makeTriple()
	to := &Triples{
		Words: 0,
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
}
