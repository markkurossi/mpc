package circuit

import (
	"testing"

	"github.com/markkurossi/mpc/ot"
)

// TestLabelForBit verifies the helper selects the correct label based on the bit.
func TestLabelForBit(t *testing.T) {
	wire := ot.Wire{
		L0: ot.Label{D0: 1},
		L1: ot.Label{D0: 2},
	}
	if LabelForBit(wire, false).D0 != 1 {
		t.Fatalf("expected L0 label")
	}
	if LabelForBit(wire, true).D0 != 2 {
		t.Fatalf("expected L1 label")
	}
}

// TestBitFromLabel ensures wires round-trip between labels and boolean values.
func TestBitFromLabel(t *testing.T) {
	wire := ot.Wire{
		L0: ot.Label{D0: 3},
		L1: ot.Label{D0: 4},
	}
	if bit, err := BitFromLabel(wire, wire.L0); err != nil || bit {
		t.Fatalf("expected false, got %v (err=%v)", bit, err)
	}
	if bit, err := BitFromLabel(wire, wire.L1); err != nil || !bit {
		t.Fatalf("expected true, got %v (err=%v)", bit, err)
	}
	if _, err := BitFromLabel(wire, ot.Label{D0: 5}); err == nil {
		t.Fatalf("expected error for unknown label")
	}
}
