//
// Copyright (c) 2022-2024 Markku Rossi
//
// All rights reserved.
//

package bmr

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"github.com/markkurossi/mpc/ot"
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

func (l Label) String() string {
	return fmt.Sprintf("%x", [k / 8]byte(l))
}

// Equal tests if the label is equal with the argument label.
func (l Label) Equal(o Label) bool {
	return bytes.Equal(l[:], o[:])
}

// Xor sets l to l^o.
func (l *Label) Xor(o Label) {
	for i := 0; i < len(l); i++ {
		l[i] ^= o[i]
	}
}

// ToOT converts the label to ot.Label.
func (l *Label) ToOT() ot.Label {
	var label ot.Label
	label.D0 = uint64(binary.BigEndian.Uint32(l[:]))
	return label
}

// FromOT sets the label to the ot.Label.
func (l *Label) FromOT(label ot.Label) {
	binary.BigEndian.PutUint32(l[:], uint32(label.D0))
}
