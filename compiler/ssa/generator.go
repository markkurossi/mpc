//
// Copyright (c) 2020, 2021 Markku Rossi
//
// All rights reserved.
//

package ssa

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
)

const (
	anon = "%_"
)

// Generator implements code generator.
type Generator struct {
	Params    *utils.Params
	versions  map[string]Variable
	blockID   int
	constants map[string]ConstantInst
	nextVarID VariableID
}

// ConstantInst defines a constant variable instance.
type ConstantInst struct {
	Count int
	Const Variable
}

// NewGenerator creates a new code generator.
func NewGenerator(params *utils.Params) *Generator {
	return &Generator{
		Params:    params,
		versions:  make(map[string]Variable),
		constants: make(map[string]ConstantInst),
		nextVarID: 1,
	}
}

// Constants returns the constants.
func (gen *Generator) Constants() map[string]ConstantInst {
	return gen.constants
}

func (gen *Generator) nextVariableID() VariableID {
	ret := gen.nextVarID
	gen.nextVarID++
	return ret
}

// UndefVar creates a new undefined variable.
func (gen *Generator) UndefVar() Variable {
	v, ok := gen.versions[anon]
	if !ok {
		v = Variable{
			Name: anon,
		}
	} else {
		v.Version = v.Version + 1
	}
	v.Type = types.Undefined
	v.ID = gen.nextVariableID()
	gen.versions[anon] = v
	return v
}

// AnonVar creates a new anonymous variable.
func (gen *Generator) AnonVar(t types.Info) Variable {
	v, ok := gen.versions[anon]
	if !ok {
		v = Variable{
			Name: anon,
		}
	} else {
		v.Version = v.Version + 1
	}
	v.Type = t
	v.ID = gen.nextVariableID()
	gen.versions[anon] = v

	return v
}

// NewVar creates a new variable with the name, type, and scope.
func (gen *Generator) NewVar(name string, t types.Info, scope int) (
	Variable, error) {

	key := fmtKey(name, scope)
	v, ok := gen.versions[key]
	if !ok {
		v = Variable{
			Name:  name,
			Scope: scope,
			Type:  t,
		}
	} else {
		v.Version = v.Version + 1
		v.Type = t
	}
	v.ID = gen.nextVariableID()
	gen.versions[key] = v

	return v, nil
}

// AddConstant adds a reference to the constant.
func (gen *Generator) AddConstant(c Variable) {
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
func (gen *Generator) RemoveConstant(c Variable) {
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

func fmtKey(name string, scope int) string {
	return fmt.Sprintf("%s@%d", name, scope)
}

// Block creates a new basic block.
func (gen *Generator) Block() *Block {
	block := &Block{
		ID: fmt.Sprintf("l%d", gen.blockID),
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

// Constant creates a constant variable for the argument value. Type
// info is optional. If it is undefined, the type info will be
// resolved from the constant value.
func (gen *Generator) Constant(value interface{}, ti types.Info) (
	Variable, bool, error) {

	v := Variable{
		Const:      true,
		ConstValue: value,
	}
	switch val := value.(type) {
	case int32:
		var minBits int
		// Count minimum bits needed to represent the value.
		for minBits = 1; minBits < 32; minBits++ {
			if (0xffffffff<<minBits)&uint64(val) == 0 {
				break
			}
		}

		v.Name = fmt.Sprintf("$%d", val)
		v.Type = types.Info{
			Type:    types.TUint,
			Bits:    32,
			MinBits: minBits,
		}

	case int64:
		var minBits int
		// Count minimum bits needed to represent the value.
		for minBits = 1; minBits < 64; minBits++ {
			if (0xffffffffffffffff<<minBits)&uint64(val) == 0 {
				break
			}
		}

		var bits int
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

	case uint64:
		var minBits int
		// Count minimum bits needed to represent the value.
		for minBits = 1; minBits < 64; minBits++ {
			if (0xffffffffffffffff<<minBits)&val == 0 {
				break
			}
		}

		var bits int
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
		minBits := val.BitLen()
		var bits int
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
		bits := len([]byte(val)) * types.ByteBits

		v.Type = types.Info{
			Type:    types.TString,
			Bits:    bits,
			MinBits: bits,
		}

	case []interface{}:
		var bits int
		var length string
		var name string

		if len(val) > 0 {
			ev, ok, err := gen.Constant(val[0], types.Undefined)
			if err != nil {
				return v, false, err
			}
			if !ok {
				return v, false, fmt.Errorf("array element is not constant")
			}
			bits = ev.Type.Bits * len(val)
			name = ev.Type.String()
			length = fmt.Sprintf("%d", len(val))
		} else {
			name = "interface{}"
		}

		v.Name = fmt.Sprintf("$[%s]%s{%v}", length, name, arrayString(val))
		if ti.Undefined() {
			v.Type = types.Info{
				Type:         types.TArray,
				Bits:         bits,
				MinBits:      bits,
				ArrayElement: ti.ArrayElement,
				ArraySize:    len(val),
			}
		} else {
			v.Type = ti
			v.Type.Bits = ti.ArraySize * ti.ArrayElement.Bits
			v.Type.MinBits = ti.Bits
		}

	case types.Info:
		v.Name = fmt.Sprintf("$%s", val)
		v.Type = val
		v.TypeRef = true

	case Variable:
		if !val.Const {
			return v, false, fmt.Errorf("value %v (%T) is not constant",
				val, val)
		}
		v = val

	default:
		return v, false,
			fmt.Errorf("Generator.Constant: %v (%T) not implemented yet",
				val, val)
	}
	v.ID = gen.nextVariableID()

	return v, true, nil
}

func arrayString(arr []interface{}) string {
	var parts []string
	for _, part := range arr {
		parts = append(parts, fmt.Sprintf("%v", part))
	}
	return strings.Join(parts, ",")
}
