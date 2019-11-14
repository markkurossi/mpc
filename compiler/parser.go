//
// parser.go
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

	"github.com/markkurossi/mpc/compiler/ast"
)

var (
	SyntaxError = errors.New("Syntax error")
)

type Unit struct {
	Package   string
	Functions map[string]*ast.Func
}

type Parser struct {
	inputName string
	lexer     *Lexer
}

func NewParser(inputName string, in io.Reader) *Parser {
	return &Parser{
		inputName: inputName,
		lexer:     NewLexer(in),
	}
}

func (p *Parser) Parse() (*Unit, error) {
	pkg, err := p.parsePackage()
	if err != nil {
		return nil, err
	}

	unit := &Unit{
		Package:   pkg,
		Functions: make(map[string]*ast.Func),
	}

	err = p.parseToplevel(unit)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return unit, nil
}

func (p *Parser) err(loc ast.Point, format string, a ...interface{}) error {
	msg := fmt.Sprintf(format, a...)
	return fmt.Errorf("%s:%d:%d: %s", p.inputName, loc.Line, loc.Col, msg)
}

func (p *Parser) errUnexpected(offending *Token, expected TokenType) error {
	return p.err(offending.From, "unexpected token '%s': expected '%s'",
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

func (p *Parser) sameLine(current *Token) bool {
	t, err := p.lexer.Get()
	if err != nil {
		return false
	}
	p.lexer.Unget(t)
	return t.From.Line == current.To.Line
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
	return t.StrVal, nil
}

func (p *Parser) parseToplevel(unit *Unit) error {
	token, err := p.lexer.Get()
	if err != nil {
		return err
	}
	switch token.Type {
	case T_SymFunc:
		f, err := p.parseFunc()
		if err != nil {
			return err
		}
		_, ok := unit.Functions[f.Name]
		if ok {
			return p.err(f.Loc, "function %s already defined", f.Name)
		}
		unit.Functions[f.Name] = f

	default:
		return p.err(token.From, "unexpected token '%s'", token.Type)
	}

	return nil
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
				if arguments[i].Type.Type != ast.TypeUndefined {
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
	var returnValues []ast.TypeInfo

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
			returnValues = append(returnValues, t.TypeInfo)
			t, err = p.lexer.Get()
			if t.Type == T_RParen {
				break
			}
			if t.Type != T_Comma {
				return nil, p.errUnexpected(t, T_Comma)
			}
		}
	} else if t.Type == T_Type {
		returnValues = append(returnValues, t.TypeInfo)
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

	return &ast.Func{
		Loc:    name.From,
		Name:   name.StrVal,
		Args:   arguments,
		Return: returnValues,
		Body:   body,
	}, nil
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
	t, err := p.lexer.Get()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case T_SymReturn:
		var expr ast.AST
		if p.sameLine(t) {
			expr, err = p.parseExpr()
			if err != nil {
				return nil, err
			}
		}
		// XXX multiple return values
		return &ast.Return{
			Expr: expr,
		}, nil

	}
	return nil, SyntaxError
}

func (p *Parser) parseExpr() (ast.AST, error) {
	return p.parseExprAdditive()
}

func (p *Parser) parseExprAdditive() (ast.AST, error) {
	left, err := p.parseExprMultiplicative()
	if err != nil {
		return nil, err
	}
	t, err := p.lexer.Get()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case T_Plus:
		right, err := p.parseExprMultiplicative()
		if err != nil {
			return nil, err
		}
		return &ast.Binary{
			Left:  left,
			Op:    t.Type.BinaryType(),
			Right: right,
		}, nil
	}
	p.lexer.Unget(t)

	return left, nil
}

func (p *Parser) parseExprMultiplicative() (ast.AST, error) {
	left, err := p.parseExprPrimary()
	if err != nil {
		return nil, err
	}
	t, err := p.lexer.Get()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case T_Mult:
		right, err := p.parseExprPrimary()
		if err != nil {
			return nil, err
		}
		return &ast.Binary{
			Left:  left,
			Op:    t.Type.BinaryType(),
			Right: right,
		}, nil
	}
	p.lexer.Unget(t)

	return left, nil
}

func (p *Parser) parseExprPrimary() (ast.AST, error) {
	t, err := p.lexer.Get()
	if err != nil {
		return nil, err
	}
	switch t.Type {
	case T_Identifier:
		return &ast.VariableRef{
			Name: t.StrVal,
		}, nil
	}
	p.lexer.Unget(t)
	return nil, p.err(t.From, "unexpected token '%s' while parsing expression",
		t)
}
