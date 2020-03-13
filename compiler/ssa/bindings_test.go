//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"testing"
)

func TestBindings(t *testing.T) {
	var a, b Bindings

	a.Set(Variable{
		Name: "a",
	}, nil)
	b.Set(Variable{
		Name: "a",
	}, nil)
	merged := a.Merge(Variable{
		Name: "c",
	}, b)
	if len(merged) != 1 {
		t.Errorf("Bindings.Merge failed")
	}
}
