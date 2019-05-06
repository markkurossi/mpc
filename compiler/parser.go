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

type Unit struct {
	Package string
	AST     ast.AST
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

	ast, err := p.parseToplevel()
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &Unit{
		Package: pkg,
		AST:     ast,
	}, nil
}

func (p *Parser) err(offending *Token, format string, a ...interface{}) error {
	msg := fmt.Sprintf(format, a...)
	return fmt.Errorf("%s:%d:%d: %s",
		p.inputName, offending.From.Line, offending.From.Col, msg)
}

func (p *Parser) errUnexpected(offending *Token, expected TokenType) error {
	return p.err(offending, "unexpected token '%s': expected '%s'",
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

func (p *Parser) parseToplevel() (ast.AST, error) {
	result := new(ast.List)

	token, err := p.lexer.Get()
	if err != nil {
		return nil, err
	}
	switch token.Type {
	case T_SymFunc:
		ast, err := p.parseFunc()
		if err != nil {
			return nil, err
		}
		result.Elements = append(result.Elements, ast)

	default:
		return nil, p.err(token, "unexpected token '%s'", token.Type)
	}

	return result, nil
}

func (p *Parser) parseFunc() (ast.AST, error) {
	name, err := p.needToken(T_Identifier)
	if err != nil {
		return nil, err
	}
	_, err = p.needToken(T_LParen)
	if err != nil {
		return nil, err
	}

	// Argument list.
	t, err := p.lexer.Get()
	if err != nil {
		return nil, err
	}
	if t.Type != T_RParen {
		for {
			if t.Type != T_Identifier {
				return nil, p.errUnexpected(t, T_Identifier)
			}
			t, err = p.lexer.Get()
			if err != nil {
				return nil, err
			}
			if t.Type == T_Type {
				// Type.
				t, err = p.lexer.Get()
				if err != nil {
					return nil, err
				}
			}
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
	var returnValues []*Token

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
			returnValues = append(returnValues, t)
			t, err = p.lexer.Get()
			if t.Type == T_RParen {
				break
			}
			if t.Type != T_Comma {
				return nil, p.errUnexpected(t, T_Comma)
			}
		}
	} else if t.Type == T_Type {
		returnValues = append(returnValues, t)
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

	fmt.Printf("func %s() %v\n", name.StrVal, returnValues)

	body, err := p.parseBlock()
	if err != nil {
		return nil, err
	}

	return &ast.Func{
		Name: name.StrVal,
		Body: body,
	}, nil
}

func (p *Parser) parseBlock() ([]ast.AST, error) {
	var result []ast.AST
	for {
		t, err := p.lexer.Get()
		if err != nil {
			return nil, err
		}
		if t.Type == T_RBrace {
			break
		}
		return nil, errors.New("parseBlock not implemented yet")
	}
	return result, nil
}
