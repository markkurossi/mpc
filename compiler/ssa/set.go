//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"sort"
)

// Set implements a set of string values.
type Set map[string]bool

// NewSet creates a new string value set.
func NewSet() Set {
	return make(map[string]bool)
}

// Contains tests if the argument value exists in the set.
func (set Set) Contains(val string) bool {
	_, ok := set[val]
	return ok
}

// Add adds a value to the set.
func (set Set) Add(val string) {
	set[val] = true
}

// Remove removes a value from set set. The operation does nothing if
// the value did not exist in the set.
func (set Set) Remove(val string) {
	delete(set, val)
}

// Copy creates a copy of the set.
func (set Set) Copy() Set {
	result := make(map[string]bool)
	for k, v := range set {
		result[k] = v
	}
	return result
}

// Subtract removes the values of the argument set from the set.
func (set Set) Subtract(o Set) {
	for k := range o {
		set.Remove(k)
	}
}

// Array returns the values of the set as an array.
func (set Set) Array() []string {
	var result []string
	for k := range set {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
