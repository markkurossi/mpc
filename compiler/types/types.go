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

func (t Type) ShortString() string {
	name, ok := shortTypes[t]
	if ok {
		return name
	}
	return t.String()
}

const (
	Undefined Type = iota
	Bool
	Int
	Uint
	Float
)

var Types = map[string]Type{
	"<Undefined>": Undefined,
	"bool":        Bool,
	"int":         Int,
	"uint":        Uint,
	"float":       Float,
}

var shortTypes = map[Type]string{
	Undefined: "?",
	Bool:      "b",
	Int:       "i",
	Uint:      "u",
	Float:     "f",
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

func (i Info) ShortString() string {
	if i.Bits == 0 {
		return i.Type.ShortString()
	}
	return fmt.Sprintf("%s%d", i.Type.ShortString(), i.Bits)
}

func (i Info) Undefined() bool {
	return i.Type == Undefined
}

func (i Info) Equal(o Info) bool {
	return i.Type == o.Type && i.Bits == o.Bits
}

func BoolType() Info {
	return Info{
		Type: Bool,
	}
}
