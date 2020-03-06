//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/markkurossi/mpc/compiler/ast"
	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
)

type Parser struct {
	compiler *Compiler
	logger   *utils.Logger
	lexer    *Lexer
	pkg      *ast.Package
}

func NewParser(source string, compiler *Compiler, logger *utils.Logger, in io.Reader) *Parser {
	return &Parser{
		compiler: compiler,
		logger:   logger,
		lexer:    NewLexer(source, in),
	}
}

func (p *Parser) Parse(pkg *ast.Package) (*ast.Package, error) {
	name, err := p.parsePackage()
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		p.pkg = ast.NewPackage(name)
	} else {
		// This source file must be in the same package.
		if name != pkg.Name {
			return nil, fmt.Errorf("found packages %s and %s", name, pkg.Name)
		}
		p.pkg = pkg
	}

	token, err := p.lexer.Get()
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
		return p.pkg, nil
	}
	if token.Type == T_SymImport {
		imports := make(map[string]string)
		_, err = p.needToken(T_LParen)
		if err != nil {
			return nil, err
		}
		for {
			t, err := p.lexer.Get()
			if err != nil {
				return nil, err
			}
			if t.Type == T_RParen {
				break
			}

			var alias string
			if t.Type == T_Identifier {
				alias = t.StrVal
				t, err = p.lexer.Get()
				if err != nil {
					return nil, err
				}
			}
			if t.Type != T_Constant {
				return nil, p.errUnexpected(t, T_Constant)
			}
			str, ok := t.ConstVal.(string)
			if !ok {
				return nil, p.errUnexpected(t, T_Constant)
			}
			_, ok = imports[str]
			if ok {
				return nil, p.errf(t.From,
					"package %s imported more than once", str)
			}

			if len(alias) == 0 {
				parts := strings.Split(str, "/")
				alias = parts[len(parts)-1]
			}

			imports[alias] = str
		}
		p.pkg.Imports = imports
	} else {
		p.lexer.Unget(token)
	}

	for {
		err = p.parseToplevel()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}

	return p.pkg, nil
}

func (p *Parser) errf(loc utils.Point, format string, a ...interface{}) error {
	msg := fmt.Sprintf(format, a...)

	p.lexer.FlushEOL()

	line, ok := p.lexer.history[loc.Line]
	if ok {
		var indicator []rune
		for i := 0; i < loc.Col; i++ {
			var r rune
			if line[i] == '\t' {
				r = '\t'
			} else {
				r = ' '
			}
			indicator = append(indicator, r)
		}
		indicator = append(indicator, '^')
		p.logger.Errorf(loc, "%s\n%s\n%s\n",
			msg, string(line), string(indicator))

		return errors.New(msg)
	}
	p.logger.Errorf(loc, "%s", msg)
	return errors.New(msg)
}

func (p *Parser) errUnexpected(offending *Token, expected TokenType) error {
	return p.errf(offending.From, "unexpected token '%s': expected '%s'",
		offending, expected)
}

func (p *Parser) needToken(tt TokenType) (*Token, error) {
	token, err := p.lexer.Get()
	if err != nil {
		return nil, err
	}
	if token.Type != tt {
		p.lexer.Unget(token)
		return nil, p.errUnexpected(token, tt)
	}
	return token, nil
}

func (p *Parser) sameLine(current utils.Point) bool {
	t, err := p.lexer.Get()
	if err != nil {
		return false
	}
	p.lexer.Unget(t)
	return t.From.Line == current.Line
}

func (p *Parser) parsePackage() (string, error) {
	t, err := p.needToken(T_SymPackage)
	if err != nil {
		return "", err
	}
	t, err = p.needToken(T_Identifier)
	if err != nil {
		return "", err
	}
	parts := strings.Split(t.StrVal, "/")
	return parts[len(parts)-1], nil
}

func (p *Parser) parseToplevel() error {
	token, err := p.lexer.Get()
	if err != nil {
		return err
	}
	switch token.Type {
	case T_SymConst:
		return p.parseConst()

	case T_SymFunc:
		f, err := p.parseFunc()
		if err != nil {
			return err
		}
		_, ok := p.pkg.Functions[f.Name]
		if ok {
			return p.errf(f.Loc, "function %s already defined", f.Name)
		}
		p.pkg.Functions[f.Name] = f

	default:
		return p.errf(token.From, "unexpected token '%s'", token.Type)
	}

	return nil
}

func (p *Parser) parseConst() error {
	token, err := p.lexer.Get()
	if err != nil {
		return err
	}
	switch token.Type {
	case T_Identifier:
		t, err := p.lexer.Get()
		if err != nil {
			return err
		}
		var constType types.Info
		if t.Type == T_Type {
			constType = t.TypeInfo
			t, err = p.lexer.Get()
			if err != nil {
				return err
			}
		} else {
			p.lexer.Unget(t)
		}
		_, err = p.needToken(T_Assign)
		if err != nil {
			return err
		}
		value, err := p.parseExprPrimary()
		if err != nil {
			return err
		}

		constVal, ok := value.(*ast.Constant)
		if !ok {
			return p.errf(value.Location(), "value %s used as constant", value)
		}
		constVar, err := constVal.Variable()
		if err != nil {
			return err
		}

		// XXX Check type compatibility
		_ = constType

		_, err = p.pkg.Bindings.Get(token.StrVal)
		if err == nil {
			return p.errf(token.From, "constant %s already defined",
				token.StrVal)
		}
		constVar.Name = token.StrVal
		p.pkg.Bindings.Set(constVar)

		return nil

	default:
		return p.errf(token.From, "unexpected token '%s'", token.Type)
	}
}

func (p *Parser) parseFunc() (*ast.Func, error) {
	name, err := p.needToken(T_Identifier)
	if err != nil {
		return nil, err
	}
	_, err = p.needToken(T_LParen)
	if err != nil {
		return nil, err
	}

	// Argument list.

	var arguments []*ast.Variable

	t, err := p.lexer.Get()
	if err != nil {
		return nil, err
	}
	if t.Type != T_RParen {
		for {
			if t.Type != T_Identifier {
				return nil, p.errUnexpected(t, T_Identifier)
			}
			arg := &ast.Variable{
				Name: t.StrVal,
			}

			t, err = p.lexer.Get()
			if err != nil {
				return nil, err
			}
			if t.Type == T_Type {
				// Type.
				arg.Type = t.TypeInfo
				t, err = p.lexer.Get()
				if err != nil {
					return nil, err
				}
			}
			// All untyped arguments get this type.
			for i := len(arguments) - 1; i >= 0; i-- {
				if arguments[i].Type.Type != types.Undefined {
					break
				}
				arguments[i].Type = arg.Type
			}

			// Append new argument.
			arguments = append(arguments, arg)

			if t.Type == T_RParen {
				break
			}
			if t.Type != T_Comma {
				return nil, p.errUnexpected(t, T_Comma)
			}
			t, err = p.lexer.Get()
			if err != nil {
				return nil, err
			}
		}
	}

	// Return values.
	var returnValues []*ast.Variable

	t, err = p.lexer.Get()
	if err != nil {
		return nil, err
	}
	if t.Type == T_LParen {
		for {
			t, err = p.needToken(T_Type)
			if err != nil {
				return nil, err
			}
			returnValues = append(returnValues, &ast.Variable{
				Type: t.TypeInfo,
			})
			t, err = p.lexer.Get()
			if t.Type == T_RParen {
				break
			}
			if t.Type != T_Comma {
				return nil, p.errUnexpected(t, T_Comma)
			}
		}
	} else if t.Type == T_Type {
		returnValues = append(returnValues, &ast.Variable{
			Type: t.TypeInfo,
		})
	} else {
		p.lexer.Unget(t)
	}

	t, err = p.lexer.Get()
	if err != nil {
		return nil, err
	}
	if t.Type != T_LBrace {
		return nil, p.errUnexpected(t, T_LBrace)
	}

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return ast.NewFunc(name.From, name.StrVal, arguments, returnValues, body),
		nil
}

func (p *Parser) parseBlock() (ast.List, error) {
	var result ast.List
	for {
		t, err := p.lexer.Get()
		if err != nil {
			return nil, err
		}
		if t.Type == T_RBrace {
			break
		}
		p.lexer.Unget(t)

		ast, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		result = append(result, ast)
	}
	return result, nil
}

func (p *Parser) parseStatement() (ast.AST, error) {
	tStmt, err := p.lexer.Get()
	if err != nil {
		return nil, err
	}
	switch tStmt.Type {
	case T_SymVar:
		var names []string
		for {
			tName, err := p.needToken(T_Identifier)
			if err != nil {
				return nil, err
			}
			names = append(names, tName.StrVal)
			t, err := p.lexer.Get()
			if err != nil {
				return nil, err
			}
			if t.Type != T_Comma {
				p.lexer.Unget(t)
				break
			}
		}
		t, err := p.lexer.Get()
		if err != nil {
			return nil, err
		}
		if t.Type != T_Type {
			p.lexer.Unget(t)
			return nil, p.errUnexpected(t, T_Type)
		}
		return &ast.VariableDef{
			Loc:   tStmt.From,
			Names: names,
			Type:  t.TypeInfo,
		}, nil

	case T_SymIf:
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		_, err = p.needToken(T_LBrace)
		if err != nil {
			return nil, err
		}

		var b1, b2 ast.List
		b1, err = p.parseBlock()
		if err != nil {
			return nil, err
		}
		t, err := p.lexer.Get()
		if err != nil {
			return nil, err
		}
		if t.Type == T_SymElse {
			_, err = p.needToken(T_LBrace)
			if err != nil {
				return nil, err
			}
			b2, err = p.parseBlock()
			if err != nil {
				return nil, err
			}
		} else {
			p.lexer.Unget(t)
		}
		return &ast.If{
			Expr:  expr,
			True:  b1,
			False: b2,
		}, nil

	case T_SymReturn:
		var exprs []ast.AST
		if p.sameLine(tStmt.To) {
			expr, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, expr)
			for {
				t, err := p.lexer.Get()
				if err != nil {
					return nil, err
				}
				if t.Type != T_Comma {
					p.lexer.Unget(t)
					break
				}
				expr, err = p.parseExpr()
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
		}
		return &ast.Return{
			Loc:   tStmt.From,
			Exprs: exprs,
		}, nil

	case T_Identifier:
		t, err := p.lexer.Get()
		if err != nil {
			return nil, err
		}
		switch t.Type {
		case T_Assign:
			expr, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			return &ast.Assign{
				Loc:  tStmt.From,
				Name: tStmt.StrVal,
				Expr: expr,
			}, nil

		default:
			p.lexer.Unget(t)
			return nil, p.errf(t.From, "syntax error")
		}
	}
	return nil, p.errf(tStmt.From, "syntax error")
}

func (p *Parser) parseExpr() (ast.AST, error) {
	// Precedence Operator
	// -----------------------------
	//   5          * / % << >> & &^
	//   4          + - | ^
	//   3          == != < <= > >=
	//   2          &&
	//   1          ||
	return p.parseExprLogicalOr()
}

func (p *Parser) parseExprLogicalOr() (ast.AST, error) {
	left, err := p.parseExprLogicalAnd()
	if err != nil {
		return nil, err
	}
	for {
		t, err := p.lexer.Get()
		if err != nil {
			return nil, err
		}
		if t.Type != T_Or {
			p.lexer.Unget(t)
			return left, nil
		}
		right, err := p.parseExprLogicalAnd()
		if err != nil {
			return nil, err
		}
		left = &ast.Binary{
			Loc:   t.From,
			Left:  left,
			Op:    t.Type.BinaryType(),
			Right: right,
		}
	}
}

func (p *Parser) parseExprLogicalAnd() (ast.AST, error) {
	left, err := p.parseExprComparative()
	if err != nil {
		return nil, err
	}
	for {
		t, err := p.lexer.Get()
		if err != nil {
			return nil, err
		}
		if t.Type != T_And {
			p.lexer.Unget(t)
			return left, nil
		}
		right, err := p.parseExprComparative()
		if err != nil {
			return nil, err
		}
		left = &ast.Binary{
			Loc:   t.From,
			Left:  left,
			Op:    t.Type.BinaryType(),
			Right: right,
		}
	}
}

func (p *Parser) parseExprComparative() (ast.AST, error) {
	left, err := p.parseExprAdditive()
	if err != nil {
		return nil, err
	}
	for {
		t, err := p.lexer.Get()
		if err != nil {
			return nil, err
		}
		switch t.Type {
		case T_Eq, T_Neq, T_Lt, T_Le, T_Gt, T_Ge:
			right, err := p.parseExprAdditive()
			if err != nil {
				return nil, err
			}
			left = &ast.Binary{
				Loc:   t.From,
				Left:  left,
				Op:    t.Type.BinaryType(),
				Right: right,
			}

		default:
			p.lexer.Unget(t)
			return left, nil
		}
	}
}

func (p *Parser) parseExprAdditive() (ast.AST, error) {
	left, err := p.parseExprMultiplicative()
	if err != nil {
		return nil, err
	}
	for {
		t, err := p.lexer.Get()
		if err != nil {
			return nil, err
		}
		switch t.Type {
		case T_Plus, T_Minus, T_BitOr, T_BitXor:
			right, err := p.parseExprMultiplicative()
			if err != nil {
				return nil, err
			}
			left = &ast.Binary{
				Loc:   t.From,
				Left:  left,
				Op:    t.Type.BinaryType(),
				Right: right,
			}

		default:
			p.lexer.Unget(t)
			return left, nil
		}
	}
}

func (p *Parser) parseExprMultiplicative() (ast.AST, error) {
	left, err := p.parseExprPrimary()
	if err != nil {
		return nil, err
	}
	for {
		t, err := p.lexer.Get()
		if err != nil {
			return nil, err
		}
		switch t.Type {
		case T_Mult, T_Div, T_Mod, T_Lshift, T_Rshift, T_BitAnd, T_BitClear:
			right, err := p.parseExprPrimary()
			if err != nil {
				return nil, err
			}
			left = &ast.Binary{
				Loc:   t.From,
				Left:  left,
				Op:    t.Type.BinaryType(),
				Right: right,
			}

		default:
			p.lexer.Unget(t)
			return left, nil
		}
	}
}

// PrimaryExpr =
//     Operand |
//     Conversion |
//     MethodExpr |
//     PrimaryExpr Selector |
//     PrimaryExpr Index |
//     PrimaryExpr Slice |
//     PrimaryExpr TypeAssertion |
//     PrimaryExpr Arguments .
//
// Selector       = "." identifier .
// Index          = "[" Expression "]" .
// Slice          = "[" [ Expression ] ":" [ Expression ] "]" |
//                  "[" [ Expression ] ":" Expression ":" Expression "]" .
// TypeAssertion  = "." "(" Type ")" .
// Arguments      = "(" [ ( ExpressionList | Type [ "," ExpressionList ] ) [ "..." ] [ "," ] ] ")" .

func (p *Parser) parseExprPrimary() (ast.AST, error) {
	primary, err := p.parseOperand()
	if err != nil {
		return nil, err
	}

	for {
		t, err := p.lexer.Get()
		if err != nil {
			if err == io.EOF {
				return primary, nil
			}
			return nil, err
		}
		switch t.Type {
		case T_Dot:
			// Selector.
			return nil, fmt.Errorf("Selector not implemented yet")

		case T_LParen:
			// Arguments.
			var arguments []ast.AST
			for {
				expr, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				arguments = append(arguments, expr)

				n, err := p.lexer.Get()
				if n.Type == T_RParen {
					break
				} else if n.Type != T_Comma {
					return nil, p.errf(n.From, "unexpected token %s", n)
				}
			}
			vr, ok := primary.(*ast.VariableRef)
			if !ok {
				return nil, p.errf(primary.Location(),
					"non-function %s used as function", primary)
			}
			if len(vr.Name.Package) == 0 {
				vr.Name.Package = p.pkg.Name
			}
			return &ast.Call{
				Loc:   primary.Location(),
				Name:  vr.Name,
				Exprs: arguments,
			}, nil

		default:
			p.lexer.Unget(t)
			return primary, nil
		}
	}
}

// Operand     = Literal | OperandName | "(" Expression ")" .
// Literal     = BasicLit | CompositeLit | FunctionLit .
// BasicLit    = int_lit | float_lit | imaginary_lit | rune_lit | string_lit .
// OperandName = identifier | QualifiedIdent .
//
// QualifiedIdent = PackageName "." identifier .

func (p *Parser) parseOperand() (ast.AST, error) {
	t, err := p.lexer.Get()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case T_Constant: // Literal
		return &ast.Constant{
			Loc:   t.From,
			Value: t.ConstVal,
		}, nil

	case T_Identifier: // OperandName
		n, err := p.lexer.Get()
		if err != nil {
			return nil, err
		}
		if n.Type == T_Dot {
			// Check that package is imported.
			name, ok := p.pkg.Imports[t.StrVal]
			if !ok {
				return nil, p.errf(t.From, "undefined: %s", t.StrVal)
			}
			p.pkg.References[t.StrVal] = name

			id, err := p.needToken(T_Identifier)
			if err != nil {
				return nil, err
			}
			// QualifiedIdent.
			return &ast.VariableRef{
				Loc: t.From,
				Name: ast.Identifier{
					Package: t.StrVal,
					Name:    id.StrVal,
				},
			}, nil
		} else {
			// Identifier in current package.
			p.lexer.Unget(n)
			return &ast.VariableRef{
				Loc: t.From,
				Name: ast.Identifier{
					Name: t.StrVal,
				},
			}, nil
		}

	case T_LParen: // '(' Expression ')'
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		_, err = p.needToken(T_RParen)
		if err != nil {
			return nil, err
		}
		return expr, nil

	default:
		p.lexer.Unget(t)
		return nil, p.errf(t.From,
			"unexpected token '%s' while parsing expression", t)
	}
}
