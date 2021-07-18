//
// ast.go
//
// Copyright (c) 2019-2021 Markku Rossi
//
// All rights reserved.
//

package ast

import (
	"fmt"
	"io"
	"math"
	"math/big"
	"regexp"
	"strconv"

	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
)

var (
	_ AST = &List{}
	_ AST = &Func{}
	_ AST = &ConstantDef{}
	_ AST = &VariableDef{}
	_ AST = &Assign{}
	_ AST = &If{}
	_ AST = &Call{}
	_ AST = &Return{}
	_ AST = &For{}
	_ AST = &Binary{}
	_ AST = &Unary{}
	_ AST = &Slice{}
	_ AST = &Index{}
	_ AST = &VariableRef{}
	_ AST = &BasicLit{}
	_ AST = &CompositeLit{}
)

func indent(w io.Writer, indent int) {
	for i := 0; i < indent; i++ {
		fmt.Fprint(w, " ")
	}
}

// Type specifies AST types.
type Type int

// AST types.
const (
	TypeName Type = iota
	TypeArray
	TypeSlice
	TypeStruct
	TypePointer
	TypeAlias
)

// TypeInfo contains AST type information.
type TypeInfo struct {
	utils.Point
	Type         Type
	Name         Identifier
	ElementType  *TypeInfo
	ArrayLength  AST
	TypeName     string
	StructFields []StructField
	AliasType    *TypeInfo
}

// StructField contains AST structure field information.
type StructField struct {
	Name string
	Type *TypeInfo
}

func (ti *TypeInfo) String() string {
	switch ti.Type {
	case TypeName:
		return ti.Name.String()

	case TypeArray:
		return fmt.Sprintf("[%s]%s", ti.ArrayLength, ti.ElementType)

	case TypeSlice:
		return fmt.Sprintf("[]%s", ti.ElementType)

	case TypeStruct:
		name := fmt.Sprintf("struct %s {", ti.TypeName)
		for idx, field := range ti.StructFields {
			if idx > 0 {
				name += ", "
			}
			name += fmt.Sprintf("%s %s", field.Name, field.Type.String())
		}
		return name + "}"

	case TypeAlias:
		return fmt.Sprintf("%s=%s", ti.TypeName, ti.AliasType)

	case TypePointer:
		return fmt.Sprintf("*%s", ti.ElementType)

	default:
		return fmt.Sprintf("{TypeInfo %d}", ti.Type)
	}
}

// IsIdentifier returns true if the type info specifies a type name
// without package.
func (ti *TypeInfo) IsIdentifier() bool {
	return ti.Type == TypeName && len(ti.Name.Package) == 0
}

var reSizedType = regexp.MustCompilePOSIX(
	`^(uint|int|float|string)([[:digit:]]*)$`)

// Resolve resolves the type information in the environment.
func (ti *TypeInfo) Resolve(env *Env, ctx *Codegen, gen *ssa.Generator) (
	types.Info, error) {

	var result types.Info
	if ti == nil {
		return result, nil
	}
	switch ti.Type {
	case TypeName:
		// XXX package
		matches := reSizedType.FindStringSubmatch(ti.Name.Name)
		if matches != nil {
			tt, ok := types.Types[matches[1]]
			if ok {
				var bits int
				if len(matches[2]) > 0 {
					bits64, err := strconv.ParseUint(matches[2], 10, 64)
					if err != nil {
						return result, err
					}
					if bits64 > math.MaxInt32 {
						bits = math.MaxInt32
					} else {
						bits = int(bits64)
					}
				} else {
					// Undefined size.
					bits = 0
				}
				if bits > gen.Params.MaxVarBits {
					return result, fmt.Errorf("bit size too large: %d > %d",
						bits, gen.Params.MaxVarBits)
				}
				return types.Info{
					Type: tt,
					Bits: bits,
				}, nil
			}
		}
		if ti.Name.Name == "bool" {
			return types.Bool, nil
		}
		// Check dynamic types from the env.
		b, ok := env.Get(ti.Name.Name)
		if ok {
			val, ok := b.Bound.(*ssa.Value)
			if ok && val.TypeRef {
				return val.Type, nil
			}
		}
		// Check dynamic types from the pkg.
		b, ok = ctx.Package.Bindings.Get(ti.Name.Name)
		if ok {
			val, ok := b.Bound.(*ssa.Value)
			if ok && val.TypeRef {
				return val.Type, nil
			}
		}
		return result, fmt.Errorf("undefined name: %s", ti)

	case TypeArray:
		// Array length.
		constLength, ok, err := ti.ArrayLength.Eval(env, ctx, gen)
		if err != nil {
			return result, err
		}
		if !ok {
			return result, ctx.Errorf(ti.ArrayLength,
				"array length is not constant: %s", ti.ArrayLength)
		}
		var length int
		switch l := constLength.ConstValue.(type) {
		case int:
			length = l
		case int32:
			length = int(l)
		case uint64:
			length = int(l)
		default:
			return result, ctx.Errorf(ti.ArrayLength,
				"invalid array length: %s", ti.ArrayLength)
		}

		// Element type.
		elInfo, err := ti.ElementType.Resolve(env, ctx, gen)
		if err != nil {
			return result, err
		}
		return types.Info{
			Type:        types.TArray,
			Bits:        length * elInfo.Bits,
			MinBits:     length * elInfo.MinBits,
			ElementType: &elInfo,
			ArraySize:   length,
		}, nil

	case TypePointer:
		// Element type.
		elInfo, err := ti.ElementType.Resolve(env, ctx, gen)
		if err != nil {
			return result, err
		}
		return types.Info{
			Type:        types.TPtr,
			ElementType: &elInfo,
		}, nil

	default:
		return result, fmt.Errorf("unsupported type %s", ti)
	}
}

// Identifier implements an AST identifier.
type Identifier struct {
	Defined string
	Package string
	Name    string
}

func (i Identifier) String() string {
	if len(i.Package) == 0 {
		return i.Name
	}
	return fmt.Sprintf("%s.%s", i.Package, i.Name)
}

// AST implements abstract syntax tree nodes.
type AST interface {
	utils.Locator

	String() string
	// SSA generates SSA code from the AST node. The code is appended
	// into the basic block `block'. The function returns the next
	// sequential basic block. The `ssa.Dead' is set to `true' if the
	// code terminates i.e. all following AST nodes are dead code.
	SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
		*ssa.Block, []ssa.Value, error)
	// Eval evaluates the AST node during constant propagation.
	Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
		value ssa.Value, isConstant bool, err error)
}

// NewEnv creates a new environment based on the current environment
// bindings in the block.
func NewEnv(block *ssa.Block) *Env {
	return &Env{
		Bindings: block.Bindings.Clone(),
	}
}

// Env implements a value bindings environment.
type Env struct {
	Bindings *ssa.Bindings
}

// Get gets the value binding from the environment.
func (env *Env) Get(name string) (ssa.Binding, bool) {
	return env.Bindings.Get(name)
}

// Set sets the value binding to the environment.
func (env *Env) Set(v ssa.Value, val *ssa.Value) {
	env.Bindings.Set(v, val)
}

// List implements an AST list statement.
type List []AST

func (ast List) String() string {
	result := "{\n"
	for _, a := range ast {
		result += fmt.Sprintf("\t%s\n", a)
	}
	return result + "}\n"
}

// Location implements the compiler.utils.Locator interface.
func (ast List) Location() utils.Point {
	if len(ast) > 0 {
		return ast[0].Location()
	}
	return utils.Point{}
}

// Variable implements an AST variable.
type Variable struct {
	utils.Point
	Name string
	Type *TypeInfo
}

// Func implements an AST function.
type Func struct {
	utils.Point
	Name         string
	This         *Variable
	Args         []*Variable
	Return       []*Variable
	NamedReturn  bool
	Body         List
	End          utils.Point
	NumInstances int
	Annotations  Annotations
}

// Annotations specify function annotations.
type Annotations []string

// NewFunc creates a new function definition.
func NewFunc(loc utils.Point, name string, args []*Variable, ret []*Variable,
	namedReturn bool, body List, end utils.Point,
	annotations Annotations) *Func {
	return &Func{
		Point:       loc,
		Name:        name,
		Args:        args,
		Return:      ret,
		NamedReturn: namedReturn,
		Body:        body,
		End:         end,
		Annotations: annotations,
	}
}

func (ast *Func) String() string {
	if ast.This != nil {
		return fmt.Sprintf("func (%s %s) %s()",
			ast.This.Name, ast.This.Type, ast.Name)
	}
	return fmt.Sprintf("func %s()", ast.Name)
}

// ConstantDef implements an AST constant definition.
type ConstantDef struct {
	utils.Point
	Name string
	Type *TypeInfo
	Init AST
}

func (ast *ConstantDef) String() string {
	result := fmt.Sprintf("const %s", ast.Name)
	if ast.Type != nil {
		result += fmt.Sprintf(" %s", ast.Type)
	}
	if ast.Init != nil {
		result += fmt.Sprintf(" = %s", ast.Init)
	}
	return result
}

// VariableDef implements an AST variable definition.
type VariableDef struct {
	utils.Point
	Names []string
	Type  *TypeInfo
	Init  AST
}

func (ast *VariableDef) String() string {
	result := fmt.Sprintf("var %v %s", ast.Names, ast.Type)
	if ast.Init != nil {
		result += fmt.Sprintf(" = %s", ast.Init)
	}
	return result
}

// Assign implements an AST assignment expression.
type Assign struct {
	utils.Point
	LValues []AST
	Exprs   []AST
	Define  bool
}

func (ast *Assign) String() string {
	var op string
	if ast.Define {
		op = ":="
	} else {
		op = "="
	}
	return fmt.Sprintf("%v %s %v", ast.LValues, op, ast.Exprs)
}

// If implements an AST if statement.
type If struct {
	utils.Point
	Expr  AST
	True  AST
	False AST
}

func (ast *If) String() string {
	return fmt.Sprintf("if %s", ast.Expr)
}

// Call implements an AST call expression.
type Call struct {
	utils.Point
	Ref   *VariableRef
	Exprs []AST
}

func (ast *Call) String() string {
	return fmt.Sprintf("%s()", ast.Ref)
}

// Return implements an AST return statement.
type Return struct {
	utils.Point
	Exprs         []AST
	AutoGenerated bool
}

func (ast *Return) String() string {
	return fmt.Sprintf("return %v", ast.Exprs)
}

// For implements an AST for statement.
type For struct {
	utils.Point
	Init AST
	Cond AST
	Inc  AST
	Body List
}

func (ast *For) String() string {
	return fmt.Sprintf("for %s; %s; %s %s",
		ast.Init, ast.Cond, ast.Inc, ast.Body)
}

// BinaryType defines binary expression types.
type BinaryType int

// Binary expression types.
const (
	BinaryMult BinaryType = iota
	BinaryDiv
	BinaryMod
	BinaryLshift
	BinaryRshift
	BinaryBand
	BinaryBclear
	BinaryPlus
	BinaryMinus
	BinaryBor
	BinaryBxor
	BinaryEq
	BinaryNeq
	BinaryLt
	BinaryLe
	BinaryGt
	BinaryGe
	BinaryAnd
	BinaryOr
)

var binaryTypes = map[BinaryType]string{
	BinaryMult:   "*",
	BinaryDiv:    "/",
	BinaryMod:    "%",
	BinaryLshift: "<<",
	BinaryRshift: ">>",
	BinaryBand:   "&",
	BinaryBclear: "&^",
	BinaryPlus:   "+",
	BinaryMinus:  "-",
	BinaryBor:    "|",
	BinaryBxor:   "^",
	BinaryEq:     "==",
	BinaryNeq:    "!=",
	BinaryLt:     "<",
	BinaryLe:     "<=",
	BinaryGt:     ">",
	BinaryGe:     ">=",
	BinaryAnd:    "&&",
	BinaryOr:     "||",
}

func (t BinaryType) String() string {
	name, ok := binaryTypes[t]
	if ok {
		return name
	}
	return fmt.Sprintf("{BinaryType %d}", t)
}

// Binary implements an AST binary expression.
type Binary struct {
	utils.Point
	Left  AST
	Op    BinaryType
	Right AST
}

func (ast *Binary) String() string {
	return fmt.Sprintf("%s %s %s", ast.Left, ast.Op, ast.Right)
}

// UnaryType defines unary expression types.
type UnaryType int

// Unary expression types.
const (
	UnaryPlus UnaryType = iota
	UnaryMinus
	UnaryNot
	UnaryXor
	UnaryPtr
	UnaryAddr
	UnarySend
)

var unaryTypes = map[UnaryType]string{
	UnaryPlus:  "+",
	UnaryMinus: "-",
	UnaryNot:   "!",
	UnaryXor:   "^",
	UnaryPtr:   "*",
	UnaryAddr:  "&",
	UnarySend:  "<-",
}

func (t UnaryType) String() string {
	name, ok := unaryTypes[t]
	if ok {
		return name
	}
	return fmt.Sprintf("{UnaryType %d}", t)
}

// Unary implements an AST unary expression.
type Unary struct {
	utils.Point
	Type UnaryType
	Expr AST
}

func (ast *Unary) String() string {
	return fmt.Sprintf("%s%s", ast.Type, ast.Expr)
}

// Slice implements an AST slice expression.
type Slice struct {
	utils.Point
	Expr AST
	From AST
	To   AST
}

func (ast *Slice) String() string {
	var fromStr, toStr string
	if ast.From != nil {
		fromStr = ast.From.String()
	}
	if ast.To != nil {
		toStr = ast.To.String()
	}
	return fmt.Sprintf("%s[%s:%s]", ast.Expr, fromStr, toStr)
}

// Index implements an AST array index expression.
type Index struct {
	utils.Point
	Expr  AST
	Index AST
}

func (ast *Index) String() string {
	return fmt.Sprintf("%s[%s]", ast.Expr, ast.Index)
}

// VariableRef implements an AST variable reference.
type VariableRef struct {
	utils.Point
	Name Identifier
}

func (ast *VariableRef) String() string {
	return ast.Name.String()
}

// BasicLit implements an AST basic literal value.
type BasicLit struct {
	utils.Point
	Value interface{}
}

func (ast *BasicLit) String() string {
	return ConstantName(ast.Value)
}

// ConstantName returns the name of the constant value.
func ConstantName(value interface{}) string {
	switch val := value.(type) {
	case int, int32, uint64:
		return fmt.Sprintf("$%d", val)
	case *big.Int:
		return fmt.Sprintf("$%s", val)
	case bool:
		return fmt.Sprintf("$%v", val)
	case string:
		return fmt.Sprintf("%q", val)
	default:
		return fmt.Sprintf("{undefined constant %v (%T)}", val, val)
	}
}

// CompositeLit implements an AST composite literal value.
type CompositeLit struct {
	utils.Point
	Type  *TypeInfo
	Value []KeyedElement
}

func (ast *CompositeLit) String() string {
	str := ast.Type.String()
	str += "{"

	for idx, e := range ast.Value {
		if idx > 0 {
			str += ","
		}
		if e.Key != nil {
			str += fmt.Sprintf("%s: %s", e.Key, e.Element)
		} else {
			str += e.Element.String()
		}
	}
	return str + "}"
}

// KeyedElement implements a keyed element of composite literal.
type KeyedElement struct {
	Key     AST
	Element AST
}
