//
// lexer.go
//
// Copyright (c) 2019 Markku Rossi
//
// All rights reserved.
//

package compiler

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"unicode"

	"github.com/markkurossi/mpc/compiler/ast"
)

type TokenType int

const (
	T_Identifier TokenType = iota
	T_Symbol
	T_SymPackage
	T_SymFunc
	T_SymIf
	T_SymElse
	T_SymReturn
	T_Type
	T_Assign
	T_Mult
	T_MultEq
	T_Div
	T_DivEq
	T_Mod
	T_Lshift
	T_Rshift
	T_Plus
	T_PlusPlus
	T_PlusEq
	T_Minus
	T_MinusMinus
	T_MinusEq
	T_LParen
	T_RParen
	T_LBrace
	T_RBrace
	T_Comma
	T_Lt
	T_Le
	T_Gt
	T_Ge
	T_Eq
	T_Neq
	T_And
	T_Or
	T_BitAnd
	T_BitOr
	T_BitXor
	T_BitClear
)

var tokenTypes = map[TokenType]string{
	T_Identifier: "identifier",
	T_Symbol:     "symbol",
	T_SymPackage: "package",
	T_SymFunc:    "func",
	T_SymIf:      "if",
	T_SymElse:    "else",
	T_SymReturn:  "return",
	T_Type:       "type",
	T_Assign:     "=",
	T_Mult:       "*",
	T_MultEq:     "*=",
	T_Div:        "/",
	T_DivEq:      "/=",
	T_Mod:        "%",
	T_Lshift:     "<<",
	T_Rshift:     ">>",
	T_Plus:       "+",
	T_PlusPlus:   "++",
	T_PlusEq:     "+=",
	T_Minus:      "-",
	T_MinusMinus: "--",
	T_MinusEq:    "-=",
	T_LParen:     "(",
	T_RParen:     ")",
	T_LBrace:     "{",
	T_RBrace:     "}",
	T_Comma:      ",",
	T_Lt:         "<",
	T_Le:         "<=",
	T_Gt:         ">",
	T_Ge:         ">=",
	T_Eq:         "==",
	T_Neq:        "!=",
	T_And:        "&&",
	T_Or:         "||",
	T_BitAnd:     "&",
	T_BitOr:      "|",
	T_BitXor:     "^",
	T_BitClear:   "&^",
}

func (t TokenType) String() string {
	name, ok := tokenTypes[t]
	if ok {
		return name
	}
	return fmt.Sprintf("{TokenType %d}", t)
}

var binaryTypes = map[TokenType]ast.BinaryType{
	T_Mult:  ast.BinaryMult,
	T_Plus:  ast.BinaryPlus,
	T_Minus: ast.BinaryMinus,
	T_Div:   ast.BinaryDiv,
	T_Lt:    ast.BinaryLt,
	T_Le:    ast.BinaryLe,
	T_Gt:    ast.BinaryGt,
	T_Ge:    ast.BinaryGe,
	T_Eq:    ast.BinaryEq,
	T_Neq:   ast.BinaryNeq,
	T_And:   ast.BinaryAnd,
	T_Or:    ast.BinaryOr,
}

func (t TokenType) BinaryType() ast.BinaryType {
	bt, ok := binaryTypes[t]
	if ok {
		return bt
	}
	panic(fmt.Sprintf("Invalid binary operator %s", t))
}

var symbols = map[string]TokenType{
	"package": T_SymPackage,
	"func":    T_SymFunc,
	"if":      T_SymIf,
	"else":    T_SymElse,
	"return":  T_SymReturn,
}

var reType = regexp.MustCompilePOSIX(`^(int|float)([[:digit:]]*)$`)

type Token struct {
	Type     TokenType
	From     ast.Point
	To       ast.Point
	StrVal   string
	TypeInfo ast.TypeInfo
}

func (t *Token) String() string {
	var str string
	if len(t.StrVal) > 0 {
		str = t.StrVal
	} else {
		str = t.Type.String()
	}
	return str
}

type Lexer struct {
	in          *bufio.Reader
	point       ast.Point
	tokenStart  ast.Point
	ungot       *Token
	unread      bool
	unreadRune  rune
	unreadPoint ast.Point
	history     map[int][]rune
}

func NewLexer(in io.Reader) *Lexer {
	return &Lexer{
		in: bufio.NewReader(in),
		point: ast.Point{
			Line: 1,
			Col:  0,
		},
		history: make(map[int][]rune),
	}
}

func (l *Lexer) ReadRune() (rune, error) {
	if l.unread {
		l.point, l.unreadPoint = l.unreadPoint, l.point
		l.unread = false
		return l.unreadRune, nil
	}
	r, _, err := l.in.ReadRune()
	if err != nil {
		return 0, err
	}

	l.unreadPoint = l.point
	if r == '\n' {
		l.point.Line++
		l.point.Col = 0
	} else {
		l.point.Col++
		l.history[l.point.Line] = append(l.history[l.point.Line], r)
	}

	return r, nil
}

func (l *Lexer) UnreadRune(r rune) {
	l.point, l.unreadPoint = l.unreadPoint, l.point
	l.unreadRune = r
	l.unread = true
}

func (l *Lexer) FlushEOL() error {
	for {
		r, err := l.ReadRune()
		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}
		if r == '\n' {
			return nil
		}
	}
}

func (l *Lexer) Get() (*Token, error) {
	if l.ungot != nil {
		token := l.ungot
		l.ungot = nil
		return token, nil
	}

	for {
		l.tokenStart = l.point
		r, err := l.ReadRune()
		if err != nil {
			return nil, err
		}
		if unicode.IsSpace(r) {
			continue
		}
		switch r {
		case '+':
			r, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(T_Plus), nil
				}
				return nil, err
			}
			switch r {
			case '+':
				return l.Token(T_PlusPlus), nil
			case '=':
				return l.Token(T_PlusEq), nil
			default:
				l.UnreadRune(r)
				return l.Token(T_Plus), nil
			}

		case '-':
			r, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(T_Minus), nil
				}
				return nil, err
			}
			switch r {
			case '-':
				return l.Token(T_MinusMinus), nil
			case '=':
				return l.Token(T_MinusEq), nil
			default:
				l.UnreadRune(r)
				return l.Token(T_Minus), nil
			}

		case '*':
			r, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(T_Mult), nil
				}
				return nil, err
			}
			switch r {
			case '=':
				return l.Token(T_MultEq), nil

			default:
				l.UnreadRune(r)
				return l.Token(T_Mult), nil
			}

		case '/':
			r, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(T_Div), nil
				}
				return nil, err
			}
			switch r {
			case '/':
				for {
					r, err := l.ReadRune()
					if err != nil {
						return nil, err
					}
					if r == '\n' {
						break
					}
				}
				continue

			case '=':
				return l.Token(T_DivEq), nil

			default:
				l.UnreadRune(r)
				return l.Token(T_Div), nil
			}

		case '(':
			return l.Token(T_LParen), nil
		case ')':
			return l.Token(T_RParen), nil
		case '{':
			return l.Token(T_LBrace), nil
		case '}':
			return l.Token(T_RBrace), nil
		case ',':
			return l.Token(T_Comma), nil

		case '<':
			r, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(T_Lt), nil
				}
				return nil, err
			}
			switch r {
			case '=':
				return l.Token(T_Le), nil
			default:
				l.UnreadRune(r)
				return l.Token(T_Lt), nil
			}

		case '>':
			r, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(T_Gt), nil
				}
				return nil, err
			}
			switch r {
			case '=':
				return l.Token(T_Ge), nil
			default:
				l.UnreadRune(r)
				return l.Token(T_Lt), nil
			}

		case '=':
			r, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(T_Assign), nil
				}
				return nil, err
			}
			switch r {
			case '=':
				return l.Token(T_Eq), nil
			default:
				l.UnreadRune(r)
				return l.Token(T_Assign), nil
			}

		case '|':
			r, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(T_BitOr), nil
				}
				return nil, err
			}
			switch r {
			case '|':
				return l.Token(T_Or), nil
			default:
				l.UnreadRune(r)
				return l.Token(T_BitOr), nil
			}

		case '&':
			r, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(T_BitAnd), nil
				}
				return nil, err
			}
			switch r {
			case '&':
				return l.Token(T_And), nil
			default:
				l.UnreadRune(r)
				return l.Token(T_BitAnd), nil
			}

		default:
			if unicode.IsLetter(r) {
				symbol := string(r)
				for {
					r, err := l.ReadRune()
					if err != nil {
						if err != io.EOF {
							return nil, err
						}
						break
					}
					if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
						l.UnreadRune(r)
						break
					}
					symbol += string(r)
				}
				tt, ok := symbols[symbol]
				if ok {
					return l.Token(tt), nil
				}
				matches := reType.FindStringSubmatch(symbol)
				if matches != nil {
					tt, ok := ast.Types[matches[1]]
					if ok {
						token := l.Token(T_Type)
						ival, _ := strconv.Atoi(matches[2])
						token.TypeInfo = ast.TypeInfo{
							Type: tt,
							Bits: ival,
						}
						token.StrVal = symbol
						return token, nil
					}
				}

				token := l.Token(T_Identifier)
				token.StrVal = symbol
				return token, nil
			}
			l.UnreadRune(r)
			return nil, fmt.Errorf("%s: unexpected character '%s'",
				l.point, string(r))
		}
	}
}

func (l *Lexer) Unget(t *Token) {
	l.ungot = t
}

func (l *Lexer) Token(t TokenType) *Token {
	return &Token{
		Type: t,
		From: l.tokenStart,
		To:   l.point,
	}
}
