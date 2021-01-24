//
// ast.go
//
// Copyright (c) 2019 Markku Rossi
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
	_ AST = &Slice{}
	_ AST = &Index{}
	_ AST = &VariableRef{}
	_ AST = &Constant{}
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
	TypeAlias
)

// TypeInfo contains AST type information.
type TypeInfo struct {
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
			return types.Info{
				Type: types.Bool,
				Bits: 1,
			}, nil
		}
		// Check dynamic types from the env.
		b, ok := env.Get(ti.Name.Name)
		if ok {
			val, ok := b.Bound.(*ssa.Variable)
			if ok && val.TypeRef {
				return val.Type, nil
			}
		}
		return result, fmt.Errorf("undefined name: %s", ti)

	default:
		return result, fmt.Errorf("unsupported type %s", ti)
	}
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

	default:
		return fmt.Sprintf("{TypeInfo %d}", ti.Type)
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
	String() string
	// Location returns the location information of the node.
	Location() utils.Point
	// SSA generates SSA code from the AST node. The code is appended
	// into the basic block `block'. The function returns the next
	// sequential basic. The `ssa.Dead' is set to `true' if the code
	// terminates i.e. all following AST nodes are dead code.
	SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
		*ssa.Block, []ssa.Variable, error)
	// Eval evaluates the AST node during constant propagation.
	Eval(env *Env, ctx *Codegen, gen *ssa.Generator) (
		interface{}, bool, error)
}

// NewEnv creates a new environment based on the current environment
// bindings in the block.
func NewEnv(block *ssa.Block) *Env {
	return &Env{
		Bindings: block.Bindings.Clone(),
	}
}

// Env implements a variable bindings environment.
type Env struct {
	Bindings ssa.Bindings
}

// Get gets the variable binding from the environment.
func (env *Env) Get(name string) (ssa.Binding, bool) {
	return env.Bindings.Get(name)
}

// Set sets the variable binding to the environment.
func (env *Env) Set(v ssa.Variable, val *ssa.Variable) {
	env.Bindings.Set(v, val)
}

// List implements an AST list statement.
type List []AST

func (ast List) String() string {
	return fmt.Sprintf("%v", []AST(ast))
}

// Location implements the compiler.ast.AST.Location for list
// statements.
func (ast List) Location() utils.Point {
	if len(ast) > 0 {
		return ast[0].Location()
	}
	return utils.Point{}
}

// Variable implements an AST variable.
type Variable struct {
	Loc  utils.Point
	Name string
	Type *TypeInfo
}

// Func implements an AST function.
type Func struct {
	Loc          utils.Point
	Name         string
	Args         []*Variable
	Return       []*Variable
	Body         List
	NumInstances int
	Annotations  Annotations
}

// Annotations specify function annotations.
type Annotations []string

// NewFunc creates a new function definition.
func NewFunc(loc utils.Point, name string, args []*Variable, ret []*Variable,
	body List, annotations Annotations) *Func {
	return &Func{
		Loc:         loc,
		Name:        name,
		Args:        args,
		Return:      ret,
		Body:        body,
		Annotations: annotations,
	}
}

func (ast *Func) String() string {
	return fmt.Sprintf("func %s()", ast.Name)
}

// Location implements the compiler.ast.AST.Location for function
// definitions.
func (ast *Func) Location() utils.Point {
	return ast.Loc
}

// ConstantDef implements an AST constant definition.
type ConstantDef struct {
	Loc  utils.Point
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

// Location implements the compiler.ast.AST.Location for constant
// definitions.
func (ast *ConstantDef) Location() utils.Point {
	return ast.Loc
}

// VariableDef implements an AST variable definition.
type VariableDef struct {
	Loc   utils.Point
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

// Location implements the compiler.ast.AST.Location for variable
// definitions.
func (ast *VariableDef) Location() utils.Point {
	return ast.Loc
}

// Assign implements an AST assignment expression.
type Assign struct {
	Loc     utils.Point
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

// Location implements the compiler.ast.AST.Location for assignment
// expressions.
func (ast *Assign) Location() utils.Point {
	return ast.Loc
}

// If implements an AST if statement.
type If struct {
	Loc   utils.Point
	Expr  AST
	True  List
	False List
}

func (ast *If) String() string {
	return fmt.Sprintf("if %s", ast.Expr)
}

// Location implements the compiler.ast.AST.Location for if
// statements.
func (ast *If) Location() utils.Point {
	return ast.Loc
}

// Call implements an AST call expression.
type Call struct {
	Loc   utils.Point
	Ref   *VariableRef
	Exprs []AST
}

func (ast *Call) String() string {
	return fmt.Sprintf("%s()", ast.Ref)
}

// Location implements the compiler.ast.AST.Location for call
// expressions.
func (ast *Call) Location() utils.Point {
	return ast.Loc
}

// Return implements an AST return statement.
type Return struct {
	Loc   utils.Point
	Exprs []AST
}

func (ast *Return) String() string {
	return fmt.Sprintf("return %v", ast.Exprs)
}

// Location implements the compiler.ast.AST.Location for return
// statements.
func (ast *Return) Location() utils.Point {
	return ast.Loc
}

// For implements an AST for statement.
type For struct {
	Loc  utils.Point
	Init AST
	Cond AST
	Inc  AST
	Body List
}

func (ast *For) String() string {
	return fmt.Sprintf("for %s; %s; %s %s",
		ast.Init, ast.Cond, ast.Inc, ast.Body)
}

// Location implements the compiler.ast.AST.Location for for
// statements.
func (ast *For) Location() utils.Point {
	return ast.Loc
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
	Loc   utils.Point
	Left  AST
	Op    BinaryType
	Right AST
}

func (ast *Binary) String() string {
	return fmt.Sprintf("%s %s %s", ast.Left, ast.Op, ast.Right)
}

// Location implements the compiler.ast.AST.Location for binary
// expressions.
func (ast *Binary) Location() utils.Point {
	return ast.Loc
}

// Slice implements an AST slice expression.
type Slice struct {
	Loc  utils.Point
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

// Location implements the compiler.ast.AST.Location for slice
// expressions.
func (ast *Slice) Location() utils.Point {
	return ast.Loc
}

// Index implements an AST array index expression.
type Index struct {
	Loc   utils.Point
	Expr  AST
	Index AST
}

func (ast *Index) String() string {
	return fmt.Sprintf("%s[%s]", ast.Expr, ast.Index)
}

// Location implements the compiler.ast.AST.Location for index
// expressions.
func (ast *Index) Location() utils.Point {
	return ast.Loc
}

// VariableRef implements an AST variable reference.
type VariableRef struct {
	Loc  utils.Point
	Name Identifier
}

func (ast *VariableRef) String() string {
	return ast.Name.String()
}

// Location implements the compiler.ast.AST.Location for variable
// references.
func (ast *VariableRef) Location() utils.Point {
	return ast.Loc
}

// Constant implements an AST constant value.
type Constant struct {
	Loc   utils.Point
	Value interface{}
}

func (ast *Constant) String() string {
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
	default:
		return fmt.Sprintf("{undefined constant %v (%T)}", val, val)
	}
}

// Location implements the compiler.ast.AST.Location for constant
// values.
func (ast *Constant) Location() utils.Point {
	return ast.Loc
}
