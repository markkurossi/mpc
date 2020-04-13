//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"sort"
)

type Set map[string]bool

func NewSet() Set {
	return make(map[string]bool)
}

func (set Set) Contains(val string) bool {
	_, ok := set[val]
	return ok
}

func (set Set) Add(val string) {
	set[val] = true
}

func (set Set) Remove(val string) {
	delete(set, val)
}

func (set Set) Copy() Set {
	result := make(map[string]bool)
	for k, v := range set {
		result[k] = v
	}
	return result
}

func (set Set) Subtract(o Set) {
	for k, _ := range o {
		set.Remove(k)
	}
}

func (set Set) Array() []string {
	var result []string
	for k, _ := range set {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
