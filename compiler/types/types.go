//
// types.go
//
// Copyright (c) 2020 Markku Rossi
//
// All rights reserved.
//

package types

import (
	"fmt"
)

type Type int

func (t Type) String() string {
	for k, v := range Types {
		if v == t {
			return k
		}
	}
	return fmt.Sprintf("{Type %d}", t)
}

const (
	Undefined Type = iota
	Int
	Uint
	Float
)

var Types = map[string]Type{
	"<Untyped>": Undefined,
	"int":       Int,
	"uint":      Uint,
	"float":     Float,
}

type Info struct {
	Type Type
	Bits int
}

func (i Info) String() string {
	if i.Bits == 0 {
		return i.Type.String()
	}
	return fmt.Sprintf("%s%d", i.Type, i.Bits)
}
