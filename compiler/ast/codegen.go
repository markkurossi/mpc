//
// ast.go
//
// Copyright (c) 2019-2022 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/types"
)

// Codegen implements compilation stack.
type Codegen struct {
	logger   *utils.Logger
	Params   *utils.Params
	Verbose  bool
	Package  *Package
	Packages map[string]*Package
	Stack    []Compilation
	Types    map[types.ID]*TypeInfo
	Native   map[string]*circuit.Circuit
	HeapID   int
}

// NewCodegen creates a new compilation.
func NewCodegen(logger *utils.Logger, pkg *Package,
	packages map[string]*Package, params *utils.Params) *Codegen {
	return &Codegen{
		logger:   logger,
		Params:   params,
		Verbose:  params.Verbose,
		Package:  pkg,
		Packages: packages,
		Types:    make(map[types.ID]*TypeInfo),
		Native:   make(map[string]*circuit.Circuit),
	}
}

// Errorf logs an error message.
func (ctx *Codegen) Errorf(locator utils.Locator, format string,
	a ...interface{}) error {
	return ctx.logger.Errorf(locator.Location(), format, a...)
}

// DefineType defines the argument type and assigns it an unique type
// ID.
func (ctx *Codegen) DefineType(t *TypeInfo) types.ID {
	id := types.ID(len(ctx.Types) + 0x80000000)
	ctx.Types[id] = t
	return id
}

// LookupFunc resolves the named function from the context.
func (ctx *Codegen) LookupFunc(block *ssa.Block, ref *VariableRef) (
	*Func, error) {

	// First, check method calls.
	if len(ref.Name.Package) > 0 {
		// Check if package name is bound to a value.
		var b ssa.Binding
		var ok bool

		b, ok = block.Bindings.Get(ref.Name.Package)
		if !ok {
			// Check names in the current package.
			b, ok = ctx.Package.Bindings.Get(ref.Name.Package)
		}
		if ok {
			var typeInfo types.Info
			if b.Type.Type == types.TPtr {
				typeInfo = *b.Type.ElementType
			} else {
				typeInfo = b.Type
			}

			info, ok := ctx.Types[typeInfo.ID]
			if !ok {
				return nil, ctx.Errorf(ref, "%s undefined", ref)
			}
			method, ok := info.Methods[ref.Name.Name]
			if !ok {
				return nil, ctx.Errorf(ref, "%s undefined", ref)
			}
			return method, nil
		}
	}

	// Next, check function calls.
	var pkgName string
	if len(ref.Name.Package) > 0 {
		pkgName = ref.Name.Package
	} else {
		pkgName = ref.Name.Defined
	}
	pkg, ok := ctx.Packages[pkgName]
	if !ok {
		return nil, ctx.Errorf(ref, "package '%s' not found", pkgName)
	}
	called, ok := pkg.Functions[ref.Name.Name]
	if !ok {
		return nil, nil
	}
	return called, nil
}

// LRValue implements value as l-value or r-value. The LRValues have
// two types: 1) base type that specifies the base memory location
// containing the value, and 2) value type that specifies the wires of
// the value. Both types can be the same.
type LRValue struct {
	ctx         *Codegen
	ast         AST
	block       *ssa.Block
	gen         *ssa.Generator
	baseInfo    *ssa.PtrInfo
	baseValue   ssa.Value
	valueType   types.Info
	value       ssa.Value
	structField *types.StructField
}

func (lrv LRValue) String() string {
	offset := lrv.baseInfo.Offset + lrv.valueType.Offset
	return fmt.Sprintf("%s[%d-%d]@%s{%d}%s",
		lrv.valueType, offset, offset+lrv.valueType.Bits,
		lrv.baseInfo.Name, lrv.baseInfo.Scope, lrv.baseInfo.ContainerType)
}

// BaseType returns the base type of the LRValue.
func (lrv *LRValue) BaseType() types.Info {
	return lrv.baseInfo.ContainerType
}

// BaseValue returns the base value of the LRValue
func (lrv *LRValue) BaseValue() ssa.Value {
	return lrv.baseValue
}

// BasePtrInfo returns the base value as PtrInfo.
func (lrv *LRValue) BasePtrInfo() *ssa.PtrInfo {
	return lrv.baseInfo
}

// LValue returns the l-valiue of the LRValue.
func (lrv *LRValue) LValue() ssa.Value {
	return lrv.gen.NewVal(lrv.baseInfo.Name, lrv.baseInfo.ContainerType,
		lrv.baseInfo.Scope)
}

// RValue returns the r-value of the LRValue.
func (lrv *LRValue) RValue() ssa.Value {
	if lrv.value.Type.Undefined() && lrv.structField != nil {
		fieldType := lrv.valueType
		fieldType.Offset = 0

		lrv.value = lrv.gen.AnonVal(fieldType)

		fromConst := lrv.gen.Constant(int32(lrv.valueType.Offset), types.Int32)
		toConst := lrv.gen.Constant(int32(lrv.valueType.Offset+
			lrv.valueType.Bits), types.Int32)

		lrv.block.AddInstr(ssa.NewSliceInstr(lrv.baseValue, fromConst, toConst,
			lrv.value))
	}
	return lrv.value
}

// ValueType returns the value type of the LRValue.
func (lrv *LRValue) ValueType() types.Info {
	return lrv.valueType
}

func (lrv *LRValue) ptrBaseValue() (ssa.Value, error) {
	b, ok := lrv.baseInfo.Bindings.Get(lrv.baseInfo.Name)
	if !ok {
		return ssa.Undefined, lrv.ctx.Errorf(lrv.ast, "undefined: %s",
			lrv.baseInfo.Name)
	}
	return b.Value(lrv.block, lrv.gen), nil
}

// ConstValue returns the constant value of the LRValue if available.
func (lrv *LRValue) ConstValue() (ssa.Value, bool, error) {
	switch lrv.value.Type.Type {
	case types.TUndefined:
		return lrv.value, false, nil

	case types.TBool, types.TInt, types.TUint, types.TFloat, types.TString,
		types.TStruct, types.TArray:
		return lrv.value, true, nil

	default:
		return ssa.Undefined, false, lrv.ctx.Errorf(lrv.ast,
			"LRValue.ConstValue: %s not supported yet: %v",
			lrv.value.Type, lrv.value)
	}
}

// LookupVar resolved the named variable from the context.
func (ctx *Codegen) LookupVar(block *ssa.Block, gen *ssa.Generator,
	bindings *ssa.Bindings, ref *VariableRef) (*LRValue, bool, error) {

	lrv := &LRValue{
		ctx:   ctx,
		ast:   ref,
		block: block,
		gen:   gen,
	}

	var err error
	var env *ssa.Bindings
	var b ssa.Binding
	var ok bool

	if len(ref.Name.Package) > 0 {
		// Check if package name is bound to a value.
		b, ok = bindings.Get(ref.Name.Package)
		if ok {
			env = bindings
		} else {
			// Check names in the current package.
			b, ok = ctx.Package.Bindings.Get(ref.Name.Package)
			if ok {
				env = ctx.Package.Bindings
			}
		}
		if ok {
			if block != nil {
				lrv.baseValue = b.Value(block, gen)
			} else {
				// Evaluating a const value.
				v, ok := b.Bound.(*ssa.Value)
				if !ok || !v.Const {
					// Value is not const
					return nil, false, nil
				}
				lrv.baseValue = *v
			}

			if lrv.baseValue.Type.Type == types.TPtr {
				lrv.baseInfo = lrv.baseValue.PtrInfo
				lrv.baseValue, err = lrv.ptrBaseValue()
				if err != nil {
					return nil, false, err
				}
			} else {
				lrv.baseInfo = &ssa.PtrInfo{
					Name:          ref.Name.Package,
					Bindings:      env,
					Scope:         b.Scope,
					ContainerType: b.Type,
				}
			}

			if lrv.baseValue.Type.Type != types.TStruct {
				return nil, false, ctx.Errorf(ref, "%s undefined", ref.Name)
			}

			for _, f := range lrv.baseValue.Type.Struct {
				if f.Name == ref.Name.Name {
					lrv.structField = &f
					break
				}
			}
			if lrv.structField == nil {
				return nil, false, ctx.Errorf(ref,
					"%s undefined (type %s has no field or method %s)",
					ref.Name, lrv.baseValue.Type, ref.Name.Name)
			}
			lrv.valueType = lrv.structField.Type

			return lrv, true, nil
		}
	}

	// Explicit package references.
	var pkg *Package
	if len(ref.Name.Package) > 0 {
		pkg, ok = ctx.Packages[ref.Name.Package]
		if !ok {
			return nil, false, ctx.Errorf(ref, "package '%s' not found",
				ref.Name.Package)
		}
		env = pkg.Bindings
		b, ok = env.Get(ref.Name.Name)
		if !ok {
			return nil, false, ctx.Errorf(ref, "undefined variable '%s'",
				ref.Name)
		}
	} else {
		// Check block bindings.
		env = bindings
		b, ok = env.Get(ref.Name.Name)
		if !ok {
			// Check names in the name's package.
			if len(ref.Name.Defined) > 0 {
				pkg, ok = ctx.Packages[ref.Name.Defined]
				if !ok {
					return nil, false, ctx.Errorf(ref, "package '%s' not found",
						ref.Name.Defined)
				}
				env = pkg.Bindings
				b, ok = env.Get(ref.Name.Name)
			}
		}
	}
	if !ok {
		return nil, false, ctx.Errorf(ref, "undefined variable '%s'", ref.Name)
	}

	if block != nil {
		lrv.value = b.Value(block, gen)
	} else {
		// Evaluating const value.
		v, ok := b.Bound.(*ssa.Value)
		if !ok || !v.Const {
			// Value is not const
			return nil, false, nil
		}
		lrv.value = *v
	}
	lrv.valueType = lrv.value.Type

	if lrv.value.Type.Type == types.TPtr {
		lrv.baseInfo = lrv.value.PtrInfo
		lrv.baseValue, err = lrv.ptrBaseValue()
		if err != nil {
			return nil, false, err
		}
	} else {
		lrv.baseInfo = &ssa.PtrInfo{
			Name:          ref.Name.Name,
			Bindings:      env,
			Scope:         b.Scope,
			ContainerType: b.Type,
		}
		lrv.baseValue = lrv.value
	}

	return lrv, true, nil
}

// Func returns the current function in the current compilation.
func (ctx *Codegen) Func() *Func {
	if len(ctx.Stack) == 0 {
		return nil
	}
	return ctx.Stack[len(ctx.Stack)-1].Called
}

// Scope returns the value scope in the current compilation.
func (ctx *Codegen) Scope() ssa.Scope {
	if ctx.Func() != nil {
		return 1
	}
	return 0
}

// PushCompilation pushes a new compilation to the compilation stack.
func (ctx *Codegen) PushCompilation(start, ret, caller *ssa.Block,
	called *Func) {

	ctx.Stack = append(ctx.Stack, Compilation{
		Start:  start,
		Return: ret,
		Caller: caller,
		Called: called,
	})
}

// PopCompilation pops the topmost compilation from the compilation
// stack.
func (ctx *Codegen) PopCompilation() {
	if len(ctx.Stack) == 0 {
		panic("compilation stack underflow")
	}
	ctx.Stack = ctx.Stack[:len(ctx.Stack)-1]
}

// Start returns the start block of the current compilation.
func (ctx *Codegen) Start() *ssa.Block {
	return ctx.Stack[len(ctx.Stack)-1].Start
}

// Return returns the return block of the current compilation.
func (ctx *Codegen) Return() *ssa.Block {
	return ctx.Stack[len(ctx.Stack)-1].Return
}

// Caller returns the caller block of the current compilation.
func (ctx *Codegen) Caller() *ssa.Block {
	return ctx.Stack[len(ctx.Stack)-1].Caller
}

// HeapVar returns the name of the next global heap variable.
func (ctx *Codegen) HeapVar() string {
	name := fmt.Sprintf("$heap%v", ctx.HeapID)
	ctx.HeapID++
	return name
}

// Compilation contains information about a compilation
// scope. Toplevel, each function call, and each nested block specify
// their own scope with their own variable bindings.
type Compilation struct {
	Start  *ssa.Block
	Return *ssa.Block
	Caller *ssa.Block
	Called *Func
	// XXX Bindings
	// XXX Parent scope.
}
