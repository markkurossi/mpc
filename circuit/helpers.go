package circuit

import (
	"fmt"

	"github.com/markkurossi/mpc/ot"
)

// LabelForBit returns the wire label corresponding to the provided bit.
func LabelForBit(wire ot.Wire, bit bool) ot.Label {
	if bit {
		return wire.L1
	}
	return wire.L0
}

// BitFromLabel resolves a concrete label back into a boolean value.
func BitFromLabel(wire ot.Wire, label ot.Label) (bool, error) {
	switch {
	case label.Equal(wire.L0):
		return false, nil
	case label.Equal(wire.L1):
		return true, nil
	default:
		return false, fmt.Errorf("unknown label %s for wire %v", label, wire)
	}
}
