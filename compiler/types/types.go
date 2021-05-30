//
// types.go
//
// Copyright (c) 2020-2021 Markku Rossi
//
// All rights reserved.
//

package types

import (
	"fmt"
)

// Type specifies an MPCL type.
type Type int

func (t Type) String() string {
	for k, v := range Types {
		if v == t {
			return k
		}
	}
	return fmt.Sprintf("{Type %d}", t)
}

// ShortString returns a short string name for the type.
func (t Type) ShortString() string {
	name, ok := shortTypes[t]
	if ok {
		return name
	}
	return t.String()
}

// MPCL types.
const (
	Undefined Type = iota
	Bool
	Int
	Uint
	Float
	String
	Struct
	Array
)

// Types define MPCL types and their names.
var Types = map[string]Type{
	"<Undefined>": Undefined,
	"bool":        Bool,
	"int":         Int,
	"uint":        Uint,
	"float":       Float,
	"string":      String,
	"struct":      Struct,
	"array":       Array,
}

var shortTypes = map[Type]string{
	Undefined: "?",
	Bool:      "b",
	Int:       "i",
	Uint:      "u",
	Float:     "f",
	String:    "str",
	Struct:    "struct",
	Array:     "arr",
}

// Info specifies information about a type.
type Info struct {
	Type    Type
	Bits    int
	MinBits int
	Struct  []StructField
	Element *Info
	Offset  int
}

// StructField defines a structure field name and type.
type StructField struct {
	Name string
	Type Info
}

func (f StructField) String() string {
	return fmt.Sprintf("%s[%d:%d]",
		f.Type.Type, f.Type.Offset, f.Type.Offset+f.Type.Bits)
}

func (i Info) String() string {
	if i.Bits == 0 {
		return i.Type.String()
	}
	return fmt.Sprintf("%s%d", i.Type, i.Bits)
}

// ShortString returns a short string name for the type info.
func (i Info) ShortString() string {
	if i.Bits == 0 {
		return i.Type.ShortString()
	}
	return fmt.Sprintf("%s%d", i.Type.ShortString(), i.Bits)
}

// Undefined tests if type is undefined.
func (i Info) Undefined() bool {
	return i.Type == Undefined
}

// Equal tests if the argument type is equal to this type info.
func (i Info) Equal(o Info) bool {
	return i.Type == o.Type && i.Bits == o.Bits
}

// CanAssignConst tests if the argument const type can be assigned to
// this type.
func (i Info) CanAssignConst(o Info) bool {
	switch i.Type {
	case Int:
		return (o.Type == Int || o.Type == Uint) && i.Bits >= o.MinBits

	default:
		return i.Type == o.Type && i.Bits >= o.MinBits
	}
}

// BoolType returns type information for the boolean type.
func BoolType() Info {
	return Info{
		Type: Bool,
		Bits: 1,
	}
}
