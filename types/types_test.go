//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package types

import (
	"testing"
)

func TestUndefined(t *testing.T) {
	undef := Info{}
	if !undef.Undefined() {
		t.Errorf("undef is not undefined")
	}
}
