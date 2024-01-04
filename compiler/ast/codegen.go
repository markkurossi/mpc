//
// ast.go
//
// Copyright (c) 2019-2023 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"
	"runtime"

	"github.com/markkurossi/mpc/circuit"
	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/utils"
	"github.com/markkurossi/mpc/types"
)

// Codegen implements compilation stack.
type Codegen struct {
	logger         *utils.Logger
	Params         *utils.Params
	Verbose        bool
	Package        *Package
	Packages       map[string]*Package
	MainInputSizes [][]int
	Stack          []Compilation
	Types          map[types.ID]*TypeInfo
	Native         map[string]*circuit.Circuit
	HeapID         int
}

// NewCodegen creates a new compilation.
func NewCodegen(logger *utils.Logger, pkg *Package,
	packages map[string]*Package, params *utils.Params,
	mainInputSizes [][]int) *Codegen {

	return &Codegen{
		logger:         logger,
		Params:         params,
		Verbose:        params.Verbose,
		Package:        pkg,
		Packages:       packages,
		MainInputSizes: mainInputSizes,
		Types:          make(map[types.ID]*TypeInfo),
		Native:         make(map[string]*circuit.Circuit),
	}
}

func (ctx *Codegen) errorLoc(err error) error {
	if !ctx.Params.MPCLCErrorLoc {
		return err
	}
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return err
	}
	fmt.Printf("%s:%d: MCPLC error:\n\u2514\u2574%s\n", file, line, err)
	return err
}

// Error logs an error message.
func (ctx *Codegen) Error(locator utils.Locator, msg string) error {
	return ctx.errorLoc(ctx.logger.Errorf(locator.Location(), "%s", msg))
}

// Errorf logs an error message.
func (ctx *Codegen) Errorf(locator utils.Locator, format string,
	a ...interface{}) error {
	return ctx.errorLoc(ctx.logger.Errorf(locator.Location(), format, a...))
}

// Warningf logs a warning message
func (ctx *Codegen) Warningf(locator utils.Locator, format string,
	a ...interface{}) {
	ctx.logger.Warningf(locator.Location(), format, a...)
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
