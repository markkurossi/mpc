//
// ast.go
//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
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

// Dereference dereferences the pointer value. If the value is not a
// pointer, returns argument value.
func (ctx *Codegen) Dereference(ast AST, ptr ssa.Value, block *ssa.Block,
	gen *ssa.Generator) (ssa.Value, error) {

	if ptr.Type.Type != types.TPtr {
		return ptr, nil
	}
	b, ok := ptr.PtrInfo.Bindings.Get(ptr.PtrInfo.Name)
	if !ok {
		return ssa.Undefined, ctx.Errorf(ast, "undefined: %s", ptr.PtrInfo.Name)
	}
	return b.Value(block, gen), nil
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

// Func returns the current function in the current compilation.
func (ctx *Codegen) Func() *Func {
	if len(ctx.Stack) == 0 {
		return nil
	}
	return ctx.Stack[len(ctx.Stack)-1].Called
}

// Scope returns the value scope in the current compilation.
func (ctx *Codegen) Scope() int {
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
