//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package utils

import (
	"testing"
)

func TestPoint(t *testing.T) {
	p := Point{}
	if !p.Undefined() {
		t.Errorf("Undefined point is not undefined")
	}
}
