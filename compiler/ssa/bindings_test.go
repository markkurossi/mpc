//
// Copyright (c) 2020-2022 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"testing"

	"github.com/markkurossi/mpc/types"
)

func TestSet(t *testing.T) {
	a := new(Bindings)
	a.Set(Value{
		Name: "a",
	}, nil)

	_, ok := a.Get("a")
	if !ok {
		t.Errorf("binding for value 'a' not found")
	}
	_, ok = a.Get("b")
	if ok {
		t.Errorf("non-existing binding for value 'b' found")
	}

	a.Set(Value{
		Name: "b",
	}, nil)
	_, ok = a.Get("b")
	if !ok {
		t.Errorf("binding for value 'b' not found")
	}
}

func TestClone(t *testing.T) {
	a := new(Bindings)
	a.Set(Value{
		Name: "a",
	}, nil)
	_, ok := a.Get("a")
	if !ok {
		t.Errorf("binding for value 'a' not found")
	}

	b := a.Clone()
	_, ok = b.Get("a")
	if !ok {
		t.Errorf("binding for value 'a' not found")
	}
	b.Set(Value{
		Name: "b",
	}, nil)
	_, ok = a.Get("b")
	if ok {
		t.Errorf("non-existing binding for value 'b' found")
	}
	_, ok = b.Get("b")
	if !ok {
		t.Errorf("binding for value 'b' not found")
	}
}

func TestMerge(t *testing.T) {
	a := new(Bindings)
	b := new(Bindings)

	a.Set(Value{
		Name: "a",
		Type: types.Int32,
	}, constInt(1))
	a.Set(Value{
		Name: "b",
		Type: types.Int32,
	}, constInt(42))

	b.Set(Value{
		Name: "a",
		Type: types.Int32,
	}, constInt(2))
	merged := a.Merge(Value{
		Name: "c",
	}, b)
	if len(merged.Values) != 2 {
		t.Errorf("Bindings.Merge failed: #values: %d != %d",
			len(merged.Values), 2)
	}

	bound, ok := merged.Get("b")
	if !ok {
		t.Errorf("binding for value 'b' not found")
	}
	_, ok = bound.Bound.(*Value)
	if !ok {
		t.Errorf("bindinf for value 'b' is not *Value: %T", bound.Bound)
	}
	bound, ok = merged.Get("a")
	if !ok {
		t.Errorf("binding for value 'a' not found")
	}
	fmt.Printf("merged.a: %v (%T)\n", bound, bound)
	_, ok = bound.Bound.(*Select)
	if !ok {
		t.Errorf("bindinf for value 'a' is not *Select: %T", bound.Bound)
	}
}

func constInt(i int) *Value {
	return &Value{
		Name:       fmt.Sprintf("%v/i32", i),
		Const:      true,
		Type:       types.Int32,
		ConstValue: i,
	}
}
