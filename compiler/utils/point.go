//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package utils

import (
	"fmt"
)

type Point struct {
	Source string
	Line   int // 1-based
	Col    int // 0-based
}

func (p Point) String() string {
	return fmt.Sprintf("%s:%d:%d", p.Source, p.Line, p.Col)
}

func (p Point) Undefined() bool {
	return p.Line == 0
}
