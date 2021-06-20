//
// Copyright (c) 2020-2021 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"testing"
)

func TestBindings(t *testing.T) {
	var a, b Bindings

	a.Set(Value{
		Name: "a",
	}, nil)
	b.Set(Value{
		Name: "a",
	}, nil)
	merged := a.Merge(Value{
		Name: "c",
	}, b)
	if len(merged) != 1 {
		t.Errorf("Bindings.Merge failed")
	}
}
