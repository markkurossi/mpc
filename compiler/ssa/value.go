//
// Copyright (c) 2020-2022 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"math/big"

	"github.com/markkurossi/mpc/types"
)

// Value implements SSA value binding.
type Value struct {
	Name       string
	ID         ValueID
	TypeRef    bool
	Const      bool
	Scope      Scope
	Version    int32
	Type       types.Info
	PtrInfo    *PtrInfo
	ConstValue interface{}
}

// Scope defines variable scope (max 256 levels of nested blocks).
type Scope int16

// PtrInfo defines context information for pointer values.
type PtrInfo struct {
	Name          string
	Bindings      *Bindings
	Scope         Scope
	Offset        types.Size
	ContainerType types.Info
}

func (ptr PtrInfo) String() string {
	return fmt.Sprintf("*%s@%d", ptr.Name, ptr.Scope)
}

// Undefined defines an undefined value.
var Undefined Value

// ValueID defines unique value IDs.
type ValueID uint32

// Check tests that the value type is properly set.
func (v Value) Check() {
	if v.Type.Type == types.TUndefined {
		panic(fmt.Sprintf("v.Type == TUndefined: %v", v))
	}
	if v.Type.Bits == 0 {
		panic(fmt.Sprintf("v.Type.Bits == 0: %v", v))
	}
}

// ElementType returns the pointer element type of the value. For
// non-pointer values, this returns the value type itself.
func (v Value) ElementType() types.Info {
	if v.Type.Type == types.TPtr {
		return *v.Type.ElementType
	}
	return v.Type
}

// ContainerType returs the pointer container type of the value. For
// non-pointer values, this returns the value type itself.
func (v Value) ContainerType() types.Info {
	if v.Type.Type == types.TPtr {
		return v.PtrInfo.ContainerType
	}
	return v.Type
}

func (v Value) String() string {
	if v.Const {
		return v.Name
	}
	if v.TypeRef {
		return v.Type.String()
	}
	var version string
	if v.Version >= 0 {
		version = fmt.Sprintf("%d", v.Version)
	} else {
		version = "?"
	}

	// XXX Value should have type, now we have flags and Type.Type
	if v.Type.Type == types.TPtr {
		return fmt.Sprintf("%s{%d,%s}%s{%s{%d}%s[%d-%d]}",
			v.Name, v.Scope, version, v.Type.ShortString(),
			v.PtrInfo.Name, v.PtrInfo.Scope, v.PtrInfo.ContainerType,
			v.PtrInfo.Offset, v.PtrInfo.Offset+v.Type.Bits)
	}
	return fmt.Sprintf("%s{%d,%s}%s",
		v.Name, v.Scope, version, v.Type.ShortString())
}

// ConstInt returns the value as const integer.
func (v *Value) ConstInt() (types.Size, error) {
	if !v.Const {
		return 0, fmt.Errorf("value is not constant")
	}
	switch val := v.ConstValue.(type) {
	case int:
		return types.Size(val), nil
	case int32:
		return types.Size(val), nil
	case int64:
		return types.Size(val), nil
	case uint64:
		return types.Size(val), nil

	default:
		return 0, fmt.Errorf("cannot use %v as integer", val)
	}
}

// Equal implements BindingValue.Equal.
func (v *Value) Equal(other BindingValue) bool {
	o, ok := other.(*Value)
	if !ok {
		return false
	}
	return o.Name == v.Name && o.Scope == v.Scope && o.Version == v.Version
}

// Value implements BindingValue.Value.
func (v *Value) Value(block *Block, gen *Generator) Value {
	return *v
}

// Bit tests if the argument bit is set in the value.
func (v *Value) Bit(bit types.Size) bool {
	arr, ok := v.ConstValue.([]interface{})
	if ok {
		length := types.Size(len(arr))
		elBits := v.Type.Bits / length
		idx := bit / elBits
		ofs := bit % elBits
		if idx >= length {
			return false
		}
		return isSet(arr[idx], v.Type, ofs)
	}

	return isSet(v.ConstValue, v.Type, bit)
}

func isSet(v interface{}, vt types.Info, bit types.Size) bool {
	switch val := v.(type) {
	case bool:
		if bit == 0 {
			return val
		}
		return false

	case int8:
		return (val & (1 << bit)) != 0

	case uint8:
		return (val & (1 << bit)) != 0

	case int32:
		return (val & (1 << bit)) != 0

	case uint32:
		return (val & (1 << bit)) != 0

	case int64:
		return (val & (1 << bit)) != 0

	case uint64:
		return (val & (1 << bit)) != 0

	case *big.Int:
		if bit > types.Size(val.BitLen()) {
			return false
		}
		return val.Bit(int(bit)) != 0

	case string:
		bytes := []byte(val)
		idx := bit / types.ByteBits
		mod := bit % types.ByteBits
		if idx >= types.Size(len(bytes)) {
			return false
		}
		return bytes[idx]&(1<<mod) != 0

	case Value:
		switch val.Type.Type {
		case types.TBool, types.TInt, types.TUint, types.TFloat, types.TString:
			return isSet(val.ConstValue, val.Type, bit)

		case types.TArray:
			elType := val.Type.ElementType
			idx := bit / elType.Bits
			mod := bit % elType.Bits
			if idx >= val.Type.ArraySize {
				return false
			}
			arr := val.ConstValue.([]interface{})
			return isSet(arr[idx], *elType, mod)

		case types.TStruct:
			fieldValues := val.ConstValue.([]interface{})
			for idx, f := range val.Type.Struct {
				if bit < f.Type.Bits {
					return isSet(fieldValues[idx], f.Type, bit)
				}
				bit -= f.Type.Bits
			}
			fallthrough

		default:
			panic(fmt.Sprintf("ssa.isSet: invalid Value %v (%v)",
				val, val.Type))
		}

	case types.Info:
		return false

	case []interface{}:
		switch vt.Type {
		case types.TStruct:
			for idx, f := range vt.Struct {
				if bit < f.Type.Bits {
					return isSet(val[idx], f.Type, bit)
				}
				bit -= f.Type.Bits
			}
			panic(fmt.Sprintf("ssa.isSet: bit overflow for %v", vt))

		case types.TArray:
			elType := vt.ElementType
			idx := bit / elType.Bits
			mod := bit % elType.Bits
			if idx >= vt.ArraySize {
				return false
			}
			return isSet(val[idx], *elType, mod)

		default:
			panic(fmt.Sprintf("ssa.isSet: type %v not supported", vt))
		}

	default:
		panic(fmt.Sprintf("ssa.isSet: non const %v (%T)", v, val))
	}
}

// LValueFor checks if the value `o` can be assigned for lvalue of type `l`.
func LValueFor(l types.Info, o Value) bool {
	if o.Const {
		return l.CanAssignConst(o.Type)
	}
	return l.Equal(o.Type)
}

// TypeCompatible tests if the argument value is type compatible with
// this value.
func (v Value) TypeCompatible(o Value) *types.Info {
	if v.Const && o.Const {
		if v.Type.Type == o.Type.Type {
			return &v.Type
		}
	} else if v.Const {
		if o.Type.CanAssignConst(v.Type) {
			return &o.Type
		}
	} else if o.Const {
		if v.Type.CanAssignConst(o.Type) {
			return &v.Type
		}
	}
	if v.Type.Equal(o.Type) {
		return &v.Type
	}
	return nil
}

// IntegerLike tests if the value is an integer.
func (v Value) IntegerLike() bool {
	switch v.Type.Type {
	case types.TInt, types.TUint:
		return true
	default:
		return false
	}
}
