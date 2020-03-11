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
	"math/big"
	"regexp"
	"strconv"
	"unicode"

	"github.com/markkurossi/mpc/compiler/ast"
	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
)

type TokenType int

const (
	T_Identifier TokenType = iota
	T_Constant
	T_Symbol
	T_SymPackage
	T_SymImport
	T_SymFunc
	T_SymIf
	T_SymElse
	T_SymReturn
	T_SymVar
	T_SymConst
	T_SymFor
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
	T_Dot
	T_Lt
	T_Le
	T_Gt
	T_Ge
	T_Eq
	T_Neq
	T_And
	T_Or
	T_Not
	T_BitAnd
	T_BitOr
	T_BitXor
	T_BitClear
)

var tokenTypes = map[TokenType]string{
	T_Identifier: "identifier",
	T_Constant:   "constant",
	T_Symbol:     "symbol",
	T_SymPackage: "package",
	T_SymImport:  "import",
	T_SymFunc:    "func",
	T_SymIf:      "if",
	T_SymElse:    "else",
	T_SymReturn:  "return",
	T_SymVar:     "var",
	T_SymConst:   "const",
	T_SymFor:     "for",
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
	T_Dot:        ".",
	T_Lt:         "<",
	T_Le:         "<=",
	T_Gt:         ">",
	T_Ge:         ">=",
	T_Eq:         "==",
	T_Neq:        "!=",
	T_And:        "&&",
	T_Or:         "||",
	T_Not:        "!",
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
	T_Mult:     ast.BinaryMult,
	T_Plus:     ast.BinaryPlus,
	T_Minus:    ast.BinaryMinus,
	T_Div:      ast.BinaryDiv,
	T_Lt:       ast.BinaryLt,
	T_Le:       ast.BinaryLe,
	T_Gt:       ast.BinaryGt,
	T_Ge:       ast.BinaryGe,
	T_Eq:       ast.BinaryEq,
	T_Neq:      ast.BinaryNeq,
	T_And:      ast.BinaryAnd,
	T_Or:       ast.BinaryOr,
	T_BitAnd:   ast.BinaryBand,
	T_BitOr:    ast.BinaryBor,
	T_BitXor:   ast.BinaryBxor,
	T_BitClear: ast.BinaryBclear,
}

func (t TokenType) BinaryType() ast.BinaryType {
	bt, ok := binaryTypes[t]
	if ok {
		return bt
	}
	panic(fmt.Sprintf("Invalid binary operator %s", t))
}

var symbols = map[string]TokenType{
	"import":  T_SymImport,
	"const":   T_SymConst,
	"for":     T_SymFor,
	"else":    T_SymElse,
	"func":    T_SymFunc,
	"if":      T_SymIf,
	"package": T_SymPackage,
	"return":  T_SymReturn,
	"var":     T_SymVar,
}

var reType = regexp.MustCompilePOSIX(`^(uint|int|float|string)([[:digit:]]*)$`)

type Token struct {
	Type     TokenType
	From     utils.Point
	To       utils.Point
	StrVal   string
	ConstVal interface{}
	TypeInfo types.Info
}

func (t *Token) String() string {
	var str string
	if len(t.StrVal) > 0 {
		str = t.StrVal
	} else {
		switch val := t.ConstVal.(type) {
		case uint64:
			str = strconv.FormatUint(val, 10)
		case bool:
			str = fmt.Sprintf("%v", val)
		case *big.Int:
			str = val.String()
		default:
			str = t.Type.String()
		}
	}
	return str
}

type Lexer struct {
	source      string
	in          *bufio.Reader
	point       utils.Point
	tokenStart  utils.Point
	ungot       *Token
	unread      bool
	unreadRune  rune
	unreadSize  int
	unreadPoint utils.Point
	history     map[int][]rune
}

func NewLexer(source string, in io.Reader) *Lexer {
	return &Lexer{
		in: bufio.NewReader(in),
		point: utils.Point{
			Source: source,
			Line:   1,
			Col:    0,
		},
		history: make(map[int][]rune),
	}
}

func (l *Lexer) ReadRune() (rune, int, error) {
	if l.unread {
		l.point, l.unreadPoint = l.unreadPoint, l.point
		l.unread = false
		return l.unreadRune, l.unreadSize, nil
	}
	r, size, err := l.in.ReadRune()
	if err != nil {
		return r, size, err
	}

	l.unreadRune = r
	l.unreadSize = size
	l.unreadPoint = l.point
	if r == '\n' {
		l.point.Line++
		l.point.Col = 0
	} else {
		l.point.Col++
		l.history[l.point.Line] = append(l.history[l.point.Line], r)
	}

	return r, size, nil
}

func (l *Lexer) UnreadRune() error {
	l.point, l.unreadPoint = l.unreadPoint, l.point
	l.unread = true
	return nil
}

func (l *Lexer) FlushEOL() error {
	for {
		r, _, err := l.ReadRune()
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
		r, _, err := l.ReadRune()
		if err != nil {
			return nil, err
		}
		if unicode.IsSpace(r) {
			continue
		}
		switch r {
		case '+':
			r, _, err := l.ReadRune()
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
				l.UnreadRune()
				return l.Token(T_Plus), nil
			}

		case '-':
			r, _, err := l.ReadRune()
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
				l.UnreadRune()
				return l.Token(T_Minus), nil
			}

		case '*':
			r, _, err := l.ReadRune()
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
				l.UnreadRune()
				return l.Token(T_Mult), nil
			}

		case '/':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(T_Div), nil
				}
				return nil, err
			}
			switch r {
			case '/':
				for {
					r, _, err := l.ReadRune()
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
				l.UnreadRune()
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
		case '.':
			return l.Token(T_Dot), nil

		case '<':
			r, _, err := l.ReadRune()
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
				l.UnreadRune()
				return l.Token(T_Lt), nil
			}

		case '>':
			r, _, err := l.ReadRune()
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
				l.UnreadRune()
				return l.Token(T_Gt), nil
			}

		case '=':
			r, _, err := l.ReadRune()
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
				l.UnreadRune()
				return l.Token(T_Assign), nil
			}

		case '|':
			r, _, err := l.ReadRune()
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
				l.UnreadRune()
				return l.Token(T_BitOr), nil
			}

		case '&':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(T_BitAnd), nil
				}
				return nil, err
			}
			switch r {
			case '&':
				return l.Token(T_And), nil
			case '^':
				return l.Token(T_BitClear), nil
			default:
				l.UnreadRune()
				return l.Token(T_BitAnd), nil
			}

		case '"':
			var val string
			for {
				r, _, err := l.ReadRune()
				if err != nil {
					return nil, err
				}
				if r == '"' {
					break
				}
				// XXX escapes
				val += string(r)
			}
			token := l.Token(T_Constant)
			token.ConstVal = val
			return token, nil

		case '!':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(T_Not), nil
				}
				return nil, err
			}
			switch r {
			case '=':
				return l.Token(T_Neq), nil
			default:
				l.UnreadRune()
				return l.Token(T_Not), nil
			}

		case '^':
			return l.Token(T_BitXor), nil

		default:
			if unicode.IsLetter(r) {
				symbol := string(r)
				for {
					r, _, err := l.ReadRune()
					if err != nil {
						if err != io.EOF {
							return nil, err
						}
						break
					}
					if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
						l.UnreadRune()
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
					tt, ok := types.Types[matches[1]]
					if ok {
						token := l.Token(T_Type)
						var bits int
						if len(matches[2]) > 0 {
							bits, err = strconv.Atoi(matches[2])
							if err != nil {
								return nil, err
							}
						} else {
							// Default size for types.
							switch tt {
							case types.Int, types.Uint:
								bits = 32
							case types.Float:
								return nil, fmt.Errorf("invalid type %s", tt)
							}
						}
						token.TypeInfo = types.Info{
							Type: tt,
							Bits: bits,
						}
						token.StrVal = symbol
						return token, nil
					}
				} else if symbol == "bool" {
					token := l.Token(T_Type)
					token.TypeInfo = types.Info{
						Type: types.Bool,
						Bits: 1,
					}
					token.StrVal = symbol
					return token, nil
				} else if symbol == "true" || symbol == "false" {
					token := l.Token(T_Constant)
					token.ConstVal = symbol == "true"
					return token, nil
				}

				token := l.Token(T_Identifier)
				token.StrVal = symbol
				return token, nil
			}
			if unicode.IsDigit(r) {
				var input string

				if r == '0' {
					// XXX 0b, 0x, etc.
					r, _, err := l.ReadRune()
					if err != nil {
						return nil, err
					}
					if r == 'x' {
						for {
							r, _, err := l.ReadRune()
							if err != nil {
								if err != io.EOF {
									return nil, err
								}
								break
							}
							if !unicode.Is(unicode.ASCII_Hex_Digit, r) {
								l.UnreadRune()
								break
							}
							input += string(r)
						}
						token := l.Token(T_Constant)
						if len(input) > 16 {
							val := new(big.Int)
							_, ok := val.SetString(input, 16)
							if !ok {
								return nil,
									fmt.Errorf("malformed constant '%s'", input)
							}
							token.ConstVal = val
						} else {
							u, err := strconv.ParseUint(input, 16, 64)
							if err != nil {
								return nil, err
							}
							token.ConstVal = u
						}
						return token, nil
					} else {
						l.UnreadRune()
					}
					input += "0"
				} else {
					input += string(r)
				}

				for {
					r, _, err := l.ReadRune()
					if err != nil {
						if err != io.EOF {
							return nil, err
						}
						break
					}
					if !unicode.IsDigit(r) {
						l.UnreadRune()
						break
					}
					input += string(r)
				}
				u, err := strconv.ParseUint(input, 10, 64)
				if err != nil {
					// XXX bigint constants
					return nil, err
				}
				token := l.Token(T_Constant)
				token.ConstVal = u
				return token, nil
			}
			l.UnreadRune()
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
