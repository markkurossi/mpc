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
	Fprint(w io.Writer, indent int)
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

func (ast List) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "{\n")
	for _, el := range ast {
		el.Fprint(w, ind+4)
		fmt.Fprintf(w, ",\n")
	}
	indent(w, ind)
	fmt.Fprintf(w, "}")
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
	Bindings     map[string]ssa.Variable
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
		Bindings:    make(map[string]ssa.Variable),
		Annotations: annotations,
	}
}

func (ast *Func) String() string {
	return fmt.Sprintf("func %s()", ast.Name)
}

func (ast *Func) Location() utils.Point {
	return ast.Loc
}

func (ast *Func) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "func %s(", ast.Name)
	for idx, arg := range ast.Args {
		if idx > 0 {
			fmt.Fprintf(w, ", ")
		}
		fmt.Fprintf(w, "%s %s", arg.Name, arg.Type)
	}

	fmt.Fprintf(w, ")")

	if len(ast.Return) > 0 {
		fmt.Fprintf(w, " ")
		if len(ast.Return) > 1 {
			fmt.Fprintf(w, "(")
		}
		for idx, ret := range ast.Return {
			if idx > 0 {
				fmt.Fprintf(w, ", ")
			}
			fmt.Fprintf(w, "%s", ret.Type)
		}
		if len(ast.Return) > 1 {
			fmt.Fprintf(w, ")")
		}
	}

	if len(ast.Body) > 0 {
		fmt.Fprintf(w, " {\n")
		for _, b := range ast.Body {
			b.Fprint(w, ind+4)
			fmt.Fprintf(w, "\n")
		}
		indent(w, ind)
		fmt.Fprintf(w, "}")
	} else {
		fmt.Fprintf(w, " {}")
	}
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

func (ast *VariableDef) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "var ")
	for idx, name := range ast.Names {
		if idx > 0 {
			fmt.Fprintf(w, ", ")
			fmt.Fprintf(w, "%s", name)
		}
	}
	fmt.Fprintf(w, " %s", ast.Type)
	if ast.Init != nil {
		fmt.Fprintf(w, " %s", ast.Init)
	}
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

func (ast *Assign) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "%s = ", ast.Name)
	ast.Expr.Fprint(w, ind)
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

func (ast *If) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "if ")
	ast.Expr.Fprint(w, 0)
	ast.True.Fprint(w, ind)
	if ast.False != nil {
		fmt.Fprintf(w, "else ")
		ast.False.Fprint(w, ind)
	}
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

func (ast *Call) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "%s(", ast.Name)
	for idx, arg := range ast.Exprs {
		if idx > 0 {
			fmt.Fprint(w, ", ")
		}
		arg.Fprint(w, ind)
	}
	fmt.Fprint(w, ")")
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

func (ast *Return) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "return")
	if len(ast.Exprs) > 0 {
		fmt.Fprintf(w, " ")
		for idx, expr := range ast.Exprs {
			if idx > 0 {
				fmt.Fprintf(w, ", ")
			}
			expr.Fprint(w, 0)
		}
	}
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

func (ast *For) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "for ")
	ast.Init.Fprint(w, ind)
	fmt.Fprintf(w, "; ")
	ast.Cond.Fprint(w, ind)
	fmt.Fprintf(w, "; ")
	ast.Inc.Fprint(w, ind)
	fmt.Fprintf(w, "{\n")

	ast.Body.Fprint(w, ind+1)
	fmt.Fprintf(w, "}\n")
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

func (ast *Binary) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	ast.Left.Fprint(w, ind)
	fmt.Fprintf(w, " %s ", ast.Op)
	ast.Right.Fprint(w, ind)
}

func (ast *Binary) FprintDebug(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "(%s ", ast.Op)
	ast.Left.Fprint(w, ind)
	fmt.Fprintf(w, " ")
	ast.Right.Fprint(w, ind)
	fmt.Fprintf(w, ")")
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

func (ast *VariableRef) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "%s", ast.Name)
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

func (ast *Constant) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "%v", ast.Value)
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

func (ast *Conversion) Fprint(w io.Writer, ind int) {
	indent(w, ind)
	fmt.Fprintf(w, "%s(%s)", ast.Type, ast.Expr)
}
