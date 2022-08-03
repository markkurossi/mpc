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

func (l Label) String() string {
	return fmt.Sprintf("%x", [k / 8]byte(l))
}

// Xor sets l to l^o.
func (l *Label) Xor(o Label) {
	for i := 0; i < len(l); i++ {
		l[i] ^= o[i]
	}
}
