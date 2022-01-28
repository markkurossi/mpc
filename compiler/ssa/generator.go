//
// Copyright (c) 2020-2022 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/types"
)

const (
	anon = "%_"
)

// Generator implements code generator.
type Generator struct {
	Params    *utils.Params
	versions  map[string]Value
	blockID   BlockID
	constants map[string]ConstantInst
	nextValID ValueID
}

// ConstantInst defines a constant value instance.
type ConstantInst struct {
	Count int
	Const Value
}

// NewGenerator creates a new code generator.
func NewGenerator(params *utils.Params) *Generator {
	return &Generator{
		Params:    params,
		versions:  make(map[string]Value),
		constants: make(map[string]ConstantInst),
		nextValID: 1,
	}
}

// Constants returns the constants.
func (gen *Generator) Constants() map[string]ConstantInst {
	return gen.constants
}

func (gen *Generator) nextValueID() ValueID {
	ret := gen.nextValID
	gen.nextValID++
	return ret
}

// UndefVal creates a new undefined value.
func (gen *Generator) UndefVal() Value {
	v, ok := gen.versions[anon]
	if !ok {
		v = Value{
			Name: anon,
		}
	} else {
		v.Version = v.Version + 1
	}
	v.Type = types.Undefined
	v.ID = gen.nextValueID()
	gen.versions[anon] = v
	return v
}

// AnonVal creates a new anonymous value.
func (gen *Generator) AnonVal(t types.Info) Value {

	if t.Type == types.TPtr && t.ElementType == nil {
		panic("pointer with nil element type")
	}

	v, ok := gen.versions[anon]
	if !ok {
		v = Value{
			Name: anon,
		}
	} else {
		v.Version = v.Version + 1
	}
	v.Type = t
	v.ID = gen.nextValueID()
	gen.versions[anon] = v

	return v
}

// NewVal creates a new value with the name, type, and scope.
func (gen *Generator) NewVal(name string, t types.Info, scope Scope) Value {

	key := fmtKey(name, scope)
	v, ok := gen.versions[key]
	if !ok {
		v = Value{
			Name:  name,
			Scope: scope,
			Type:  t,
		}
	} else {
		v.Version = v.Version + 1
		v.Type = t
	}
	v.ID = gen.nextValueID()
	gen.versions[key] = v

	return v
}

// AddConstant adds a reference to the constant.
func (gen *Generator) AddConstant(c Value) {
	// Add only values which have the ConstValue set.
	if c.ConstValue == nil {
		return
	}
	inst, ok := gen.constants[c.Name]
	if !ok {
		inst = ConstantInst{
			Count: 1,
			Const: c,
		}
	} else {
		inst.Count++
	}
	gen.constants[c.Name] = inst
}

// RemoveConstant drops a reference from the constant.
func (gen *Generator) RemoveConstant(c Value) {
	inst, ok := gen.constants[c.Name]
	if !ok {
		return
	}
	inst.Count--
	if inst.Count == 0 {
		delete(gen.constants, c.Name)
	} else {
		gen.constants[c.Name] = inst
	}
}

func fmtKey(name string, scope Scope) string {
	return fmt.Sprintf("%s@%d", name, scope)
}

// Block creates a new basic block.
func (gen *Generator) Block() *Block {
	block := &Block{
		ID:       gen.blockID,
		Bindings: new(Bindings),
	}
	gen.blockID++

	return block
}

// NextBlock adds the next basic block to the argument block.
func (gen *Generator) NextBlock(b *Block) *Block {
	n := gen.Block()
	n.Bindings = b.Bindings.Clone()
	b.SetNext(n)
	return n
}

// BranchBlock creates a new branch basic block.
func (gen *Generator) BranchBlock(b *Block) *Block {
	n := gen.Block()
	n.Bindings = b.Bindings.Clone()
	b.SetBranch(n)
	return n
}

// Constant creates a constant value for the argument value. Type info
// is optional. If it is undefined, the type info will be resolved
// from the constant value.
func (gen *Generator) Constant(value interface{}, ti types.Info) Value {

	var minBits types.Size
	var bits types.Size

	v := Value{
		Const:      true,
		ConstValue: value,
	}
	switch val := value.(type) {
	case int8:
		// Count minimum bits needed to represent the value.
		for minBits = 1; minBits < 8; minBits++ {
			if (0xffffffff<<minBits)&uint64(val) == 0 {
				break
			}
		}

		v.Name = fmt.Sprintf("$%d", val)
		v.Type = types.Info{
			Bits:    8,
			MinBits: minBits,
		}
		if val < 0 {
			v.Type.Type = types.TInt
		} else {
			v.Type.Type = types.TUint
		}

	case uint8:
		// Count minimum bits needed to represent the value.
		for minBits = 1; minBits < 8; minBits++ {
			if (0xffffffff<<minBits)&uint64(val) == 0 {
				break
			}
		}

		v.Name = fmt.Sprintf("$%d", val)
		v.Type = types.Info{
			Type:    types.TUint,
			Bits:    8,
			MinBits: minBits,
		}

	case int32:
		// Count minimum bits needed to represent the value.
		for minBits = 1; minBits < 32; minBits++ {
			if (0xffffffff<<minBits)&uint64(val) == 0 {
				break
			}
		}

		v.Name = fmt.Sprintf("$%d", val)
		v.Type = types.Info{
			Bits:    32,
			MinBits: minBits,
		}
		if ti.Undefined() {
			if val < 0 {
				v.Type.Type = types.TInt
			} else {
				v.Type.Type = types.TUint
			}
		} else {
			v.Type = ti
			v.Type.MinBits = minBits
			if v.Type.Bits == 0 {
				v.Type.Bits = minBits
			}
		}

	case int64:
		// Count minimum bits needed to represent the value.
		for minBits = 1; minBits < 64; minBits++ {
			if (0xffffffffffffffff<<minBits)&uint64(val) == 0 {
				break
			}
		}

		if minBits > 32 {
			bits = 64
		} else {
			bits = 32
		}

		v.Name = fmt.Sprintf("$%d", val)
		v.Type = types.Info{
			Bits:    bits,
			MinBits: minBits,
		}
		if val < 0 {
			v.Type.Type = types.TInt
		} else {
			v.Type.Type = types.TUint
		}

	case uint64:
		// Count minimum bits needed to represent the value.
		for minBits = 1; minBits < 64; minBits++ {
			if (0xffffffffffffffff<<minBits)&val == 0 {
				break
			}
		}

		if minBits > 32 {
			bits = 64
		} else {
			bits = 32
		}

		v.Name = fmt.Sprintf("$%d", val)
		v.Type = types.Info{
			Type:    types.TUint,
			Bits:    bits,
			MinBits: minBits,
		}

	case *big.Int:
		v.Name = fmt.Sprintf("$%s", val.String())
		if val.Sign() == -1 {
			v.Type = types.Info{
				Type: types.TInt,
			}
		} else {
			v.Type = types.Info{
				Type: types.TUint,
			}
		}
		minBits = types.Size(val.BitLen())
		if minBits > 64 {
			bits = minBits
		} else if minBits > 32 {
			bits = 64
		} else {
			bits = 32
		}

		v.Type.Bits = bits
		v.Type.MinBits = minBits

	case bool:
		v.Name = fmt.Sprintf("$%v", val)
		v.Type = types.Bool

	case string:
		v.Name = fmt.Sprintf("$%q", val)
		bits = types.Size(len([]byte(val)) * types.ByteBits)

		v.Type = types.Info{
			Type:    types.TString,
			Bits:    bits,
			MinBits: bits,
		}

	case []interface{}:
		var length string
		var name string
		var elementType types.Info

		if len(val) > 0 {
			ev := gen.Constant(val[0], types.Undefined)
			elementType = ev.Type
			bits = ev.Type.Bits * types.Size(len(val))
			name = ev.Type.String()
			length = fmt.Sprintf("%d", len(val))
		} else {
			name = "interface{}"
		}
		if !ti.Undefined() && ti.Type == types.TStruct {
			v.Name = "$" + ti.String()
			ti.Bits = bits
			ti.MinBits = bits
			v.Type = ti
			return v
		}

		v.Name = fmt.Sprintf("$[%s]%s%s", length, name, arrayString(val))
		if ti.Undefined() {
			v.Type = types.Info{
				Type:        types.TArray,
				Bits:        bits,
				MinBits:     bits,
				ElementType: &elementType,
				ArraySize:   types.Size(len(val)),
			}
		} else {
			v.Type = ti
			v.Type.Bits = ti.ArraySize * ti.ElementType.Bits
			v.Type.MinBits = ti.Bits
		}

	case types.Info:
		v.Name = fmt.Sprintf("$%s", val)
		v.Type = val
		v.TypeRef = true

	case Value:
		if !val.Const {
			panic(fmt.Sprintf("value %v (%T) is not constant", val, val))
		}
		v = val

	default:
		panic(fmt.Sprintf("Generator.Constant: %v (%T) not implemented yet",
			val, val))
	}
	v.ID = gen.nextValueID()

	return v
}

func arrayString(arr []interface{}) string {
	var parts []string

	for _, part := range arr {
		value, ok := part.(Value)
		if ok && value.Const {
			arr, ok := value.ConstValue.([]interface{})
			if ok {
				parts = append(parts, arrayString(arr))
				continue
			}
		}
		parts = append(parts, fmt.Sprintf("%v", part))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
