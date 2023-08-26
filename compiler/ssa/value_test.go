//
// Copyright (c) 2023 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"testing"
)

var inputs = []string{
	"$127", "$126", "$125", "$124", "$123", "$122", "$121", "$119",
	"$118", "$117", "$116", "$115", "$114", "$113", "$111", "$110",
	"$109", "$108", "$107", "$106", "$105", "$103", "$102", "$101",
	"$100",
}

func TestHashCode(t *testing.T) {
	counts := make(map[int]int)
	for _, input := range inputs {
		v := Value{
			Name:  input,
			Const: true,
		}
		counts[v.HashCode()]++
	}

	for k, v := range counts {
		if v > 1 {
			t.Errorf("HashCode %v: count=%v\n", k, v)
		}
	}
}
