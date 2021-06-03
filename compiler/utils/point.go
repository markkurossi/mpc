//
// Copyright (c) 2020-2021 Markku Rossi
//
// All rights reserved.
//

package utils

import (
	"fmt"
)

// Locator is an interface that implements Location method for
// returning item's input data position.
type Locator interface {
	Location() Point
}

// Point specifies a position in the compiler input data.
type Point struct {
	Source string
	Line   int // 1-based
	Col    int // 0-based
}

// Location implements the Locator interface.
func (p Point) Location() Point {
	return p
}

func (p Point) String() string {
	return fmt.Sprintf("%s:%d:%d", p.Source, p.Line, p.Col)
}

// Undefined tests if the input position is undefined.
func (p Point) Undefined() bool {
	return p.Line == 0
}
