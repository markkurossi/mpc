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
	"math/big"

	"github.com/markkurossi/mpc/compiler/ssa"
	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
)

var (
	_ AST = &List{}
	_ AST = &Func{}
	_ AST = &VariableDef{}
	_ AST = &Assign{}
	_ AST = &If{}
	_ AST = &Call{}
	_ AST = &Return{}
	_ AST = &For{}
	_ AST = &Binary{}
	_ AST = &Slice{}
	_ AST = &VariableRef{}
	_ AST = &Constant{}
	_ AST = &Conversion{}
)

func indent(w io.Writer, indent int) {
	for i := 0; i < indent; i++ {
		fmt.Fprint(w, " ")
	}
}

type Identifier struct {
	Package string
	Name    string
}

func (i Identifier) String() string {
	if len(i.Package) == 0 {
		return i.Name
	}
	return fmt.Sprintf("%s.%s", i.Package, i.Name)
}

type AST interface {
	String() string
	Location() utils.Point
	// SSA generates SSA code from the AST node. The code is appended
	// into the basic block `block'. The function returns the next
	// sequential basic. The `ssa.Dead' is set to `true' if the code
	// terminates i.e. all following AST nodes are dead code.
	SSA(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
		*ssa.Block, []ssa.Variable, error)
	Eval(block *ssa.Block, ctx *Codegen, gen *ssa.Generator) (
		interface{}, bool, error)
}

type List []AST

func (ast List) String() string {
	return fmt.Sprintf("%v", []AST(ast))
}

func (ast List) Location() utils.Point {
	if len(ast) > 0 {
		return ast[0].Location()
	}
	return utils.Point{}
}

type Variable struct {
	Name string
	Type types.Info
}

type Func struct {
	Loc          utils.Point
	Name         string
	Args         []*Variable
	Return       []*Variable
	Body         List
	NumInstances int
	Annotations  Annotations
}

type Annotations []string

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

func (ast *Func) Location() utils.Point {
	return ast.Loc
}

type VariableDef struct {
	Loc   utils.Point
	Names []string
	Type  types.Info
	Init  AST
}

func (ast *VariableDef) String() string {
	result := fmt.Sprintf("var %v %s", ast.Names, ast.Type)
	if ast.Init != nil {
		result += fmt.Sprintf(" = %s", ast.Init)
	}
	return result
}

func (ast *VariableDef) Location() utils.Point {
	return ast.Loc
}

type Assign struct {
	Loc    utils.Point
	Name   string
	Expr   AST
	Define bool
}

func (ast *Assign) String() string {
	var op string
	if ast.Define {
		op = ":="
	} else {
		op = "="
	}
	return fmt.Sprintf("%s %s %s", ast.Name, op, ast.Expr)
}

func (ast *Assign) Location() utils.Point {
	return ast.Loc
}

type If struct {
	Loc   utils.Point
	Expr  AST
	True  List
	False List
}

func (ast *If) String() string {
	return fmt.Sprintf("if %s", ast.Expr)
}

func (ast *If) Location() utils.Point {
	return ast.Loc
}

type Call struct {
	Loc   utils.Point
	Name  Identifier
	Exprs []AST
}

func (ast *Call) String() string {
	return fmt.Sprintf("%s()", ast.Name)
}

func (ast *Call) Location() utils.Point {
	return ast.Loc
}

type Return struct {
	Loc   utils.Point
	Exprs []AST
}

func (ast *Return) String() string {
	return fmt.Sprintf("return %v", ast.Exprs)
}

func (ast *Return) Location() utils.Point {
	return ast.Loc
}

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

func (ast *For) Location() utils.Point {
	return ast.Loc
}

type BinaryType int

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

type Binary struct {
	Loc   utils.Point
	Left  AST
	Op    BinaryType
	Right AST
}

func (ast *Binary) String() string {
	return fmt.Sprintf("%s %s %s", ast.Left, ast.Op, ast.Right)
}

func (ast *Binary) Location() utils.Point {
	return ast.Loc
}

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

func (ast *Slice) Location() utils.Point {
	return ast.Loc
}

type VariableRef struct {
	Loc  utils.Point
	Name Identifier
}

func (ast *VariableRef) String() string {
	return ast.Name.String()
}

func (ast *VariableRef) Location() utils.Point {
	return ast.Loc
}

type Constant struct {
	Loc   utils.Point
	Value interface{}
}

func (ast *Constant) String() string {
	return ConstantName(ast.Value)
}

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

func (ast *Constant) Location() utils.Point {
	return ast.Loc
}

type Conversion struct {
	Loc  utils.Point
	Type types.Info
	Expr AST
}

func (ast *Conversion) String() string {
	return fmt.Sprintf("%s(%s)", ast.Type, ast.Expr)
}

func (ast *Conversion) Location() utils.Point {
	return ast.Loc
}
