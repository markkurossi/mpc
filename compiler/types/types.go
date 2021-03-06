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

// ByteBits defines the byte size in bits.
const ByteBits = 8

// MPCL types.
const (
	TUndefined Type = iota
	TBool
	TInt
	TUint
	TFloat
	TString
	TStruct
	TArray
	TPtr
)

// Types define MPCL types and their names.
var Types = map[string]Type{
	"<Undefined>": TUndefined,
	"bool":        TBool,
	"int":         TInt,
	"uint":        TUint,
	"float":       TFloat,
	"string":      TString,
	"struct":      TStruct,
	"array":       TArray,
	"ptr":         TPtr,
}

var shortTypes = map[Type]string{
	TUndefined: "?",
	TBool:      "b",
	TInt:       "i",
	TUint:      "u",
	TFloat:     "f",
	TString:    "str",
	TStruct:    "struct",
	TArray:     "arr",
	TPtr:       "*",
}

// Info specifies information about a type.
type Info struct {
	Type        Type
	Bits        int
	MinBits     int
	Struct      []StructField
	ElementType *Info
	ArraySize   int
	Offset      int
}

// Undefined defines type info for undefined types.
var Undefined = Info{
	Type: TUndefined,
}

// Bool defines type info for boolean values.
var Bool = Info{
	Type:    TBool,
	Bits:    1,
	MinBits: 1,
}

// Int32 defines type info for signed 32bit integers.
var Int32 = Info{
	Type:    TInt,
	Bits:    32,
	MinBits: 32,
}

// Uint32 defines type info for unsigned 32bit integers.
var Uint32 = Info{
	Type:    TUint,
	Bits:    32,
	MinBits: 32,
}

// Uint64 defines type info for unsigned 64bit integers.
var Uint64 = Info{
	Type:    TUint,
	Bits:    64,
	MinBits: 64,
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
	switch i.Type {
	case TArray:
		return fmt.Sprintf("[%d]%s", i.ArraySize, i.ElementType)

	case TPtr:
		return fmt.Sprintf("*%s", i.ElementType)

	default:
		return fmt.Sprintf("%s%d", i.Type, i.Bits)
	}
}

// ShortString returns a short string name for the type info.
func (i Info) ShortString() string {
	if i.Bits == 0 {
		return i.Type.ShortString()
	}
	if i.Type == TPtr {
		return fmt.Sprintf("*%s", i.ElementType.ShortString())
	}
	return fmt.Sprintf("%s%d", i.Type.ShortString(), i.Bits)
}

// Undefined tests if type is undefined.
func (i Info) Undefined() bool {
	return i.Type == TUndefined
}

// Instantiate instantiates template type to match parameter type.
func (i *Info) Instantiate(o Info) bool {
	if i.Type != o.Type {
		return false
	}
	if i.Bits != 0 {
		return false
	}
	switch i.Type {
	case TStruct, TArray:
		return false

	case TPtr:
		i.Bits = o.Bits
		return i.ElementType.Instantiate(*o.ElementType)

	default:
		i.Bits = o.Bits
		return true
	}
}

// Equal tests if the argument type is equal to this type info.
func (i Info) Equal(o Info) bool {
	if i.Type != o.Type {
		return false
	}
	switch i.Type {
	case TUndefined, TBool, TInt, TUint, TFloat, TString:
		return i.Bits == o.Bits

	case TStruct:
		if len(i.Struct) != len(o.Struct) || i.Bits != o.Bits {
			return false
		}
		for idx, ie := range i.Struct {
			if ie.Type.Equal(o.Struct[idx].Type) {
				return false
			}
		}
		return true

	case TArray:
		if i.ArraySize != o.ArraySize || i.Bits != o.Bits {
			return false
		}
		return i.ElementType.Equal(*o.ElementType)

	case TPtr:
		return i.ElementType.Equal(*o.ElementType)

	default:
		panic(fmt.Sprintf("Info.Equal called for %v (%T)", i.Type, i.Type))
	}
}

// CanAssignConst tests if the argument const type can be assigned to
// this type.
func (i Info) CanAssignConst(o Info) bool {
	switch i.Type {
	case TInt:
		return (o.Type == TInt || o.Type == TUint) && i.Bits >= o.MinBits

	default:
		return i.Type == o.Type && i.Bits >= o.MinBits
	}
}

// BoolType returns type information for the boolean type.
func BoolType() Info {
	return Info{
		Type: TBool,
		Bits: 1,
	}
}
