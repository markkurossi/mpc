//
// Copyright (c) 2022 Markku Rossi
//
// All rights reserved.
//

package bmr

import (
	"crypto/rand"
	"fmt"
)

// Wire implements a circuit wire.
type Wire struct {
	L0 Label
	L1 Label
}

func (w Wire) String() string {
	return fmt.Sprintf("%v|%v", w.L0, w.L1)
}

// Label implements a wire 0 or 1 label.
type Label [k / 8]byte

// NewLabel creates a new random label.
func NewLabel() (Label, error) {
	var l Label

	_, err := rand.Read(l[:])
	if err != nil {
		return l, err
	}
	return l, nil
}

// NewLabelFromData creates a new label from the argument data.
func NewLabelFromData(data []byte) (Label, error) {
	var l Label

	switch len(data) {
	case 1:
		switch data[0] {
		case 0:

		case 1:
			for i := 0; i < len(l); i++ {
				l[i] = 0xff
			}

		default:
			return l, fmt.Errorf("invalid bit label data: 0x%x", data[0])
		}

	case k / 8:
		copy(l[:], data)

	default:
		return l, fmt.Errorf("invalid data length: %v != %v", len(data), k/8)
	}
	return l, nil
}

func (l Label) String() string {
	return fmt.Sprintf("%x", [k / 8]byte(l))
}

// Copy creates a new copy of the label.
func (l Label) Copy() Label {
	var r Label

	copy(r[:], l[:])
	return r
}

// Equal tests if the argument label is equal to this label.
func (l Label) Equal(o Label) bool {
	for i := 0; i < len(l); i++ {
		if l[i] != o[i] {
			return false
		}
	}
	return true
}

// Xor sets l to l^o.
func (l *Label) Xor(o Label) {
	for i := 0; i < len(l); i++ {
		l[i] ^= o[i]
	}
}

// BitXor sets each label bit to labelBit^bit.
func (l *Label) BitXor(bit uint) {
	var mask byte
	if bit != 0 {
		mask = 0xff
	}
	for i := 0; i < len(l); i++ {
		l[i] ^= mask
	}
}

// Mult multiplies each label bit with bit.
func (l *Label) Mult(bit uint) {
	var mask byte
	if bit != 0 {
		mask = 0xff
	}
	for i := 0; i < len(l); i++ {
		l[i] &= mask
	}
}
