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
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"

	"github.com/markkurossi/mpc/compiler/ast"
	"github.com/markkurossi/mpc/compiler/types"
	"github.com/markkurossi/mpc/compiler/utils"
)

// TokenType specifies input token types.
type TokenType int

// Input tokens.
const (
	TIdentifier TokenType = iota
	TConstant
	TSymbol
	TSymPackage
	TSymImport
	TSymFunc
	TSymIf
	TSymElse
	TSymReturn
	TSymStruct
	TSymVar
	TSymConst
	TSymType
	TSymFor
	TAssign
	TDefAssign
	TMult
	TMultEq
	TDiv
	TDivEq
	TMod
	TLshift
	TRshift
	TPlus
	TPlusPlus
	TPlusEq
	TMinus
	TMinusMinus
	TMinusEq
	TLParen
	TRParen
	TLBrace
	TRBrace
	TLBracket
	TRBracket
	TComma
	TSemicolon
	TColon
	TDot
	TLt
	TLe
	TGt
	TGe
	TEq
	TNeq
	TAnd
	TOr
	TNot
	TBitAnd
	TBitOr
	TBitXor
	TBitClear
)

var tokenTypes = map[TokenType]string{
	TIdentifier: "identifier",
	TConstant:   "constant",
	TSymbol:     "symbol",
	TSymPackage: "package",
	TSymImport:  "import",
	TSymFunc:    "func",
	TSymIf:      "if",
	TSymElse:    "else",
	TSymReturn:  "return",
	TSymStruct:  "struct",
	TSymVar:     "var",
	TSymConst:   "const",
	TSymType:    "type",
	TSymFor:     "for",
	TAssign:     "=",
	TDefAssign:  ":=",
	TMult:       "*",
	TMultEq:     "*=",
	TDiv:        "/",
	TDivEq:      "/=",
	TMod:        "%",
	TLshift:     "<<",
	TRshift:     ">>",
	TPlus:       "+",
	TPlusPlus:   "++",
	TPlusEq:     "+=",
	TMinus:      "-",
	TMinusMinus: "--",
	TMinusEq:    "-=",
	TLParen:     "(",
	TRParen:     ")",
	TLBrace:     "{",
	TRBrace:     "}",
	TLBracket:   "[",
	TRBracket:   "]",
	TComma:      ",",
	TSemicolon:  ";",
	TColon:      ":",
	TDot:        ".",
	TLt:         "<",
	TLe:         "<=",
	TGt:         ">",
	TGe:         ">=",
	TEq:         "==",
	TNeq:        "!=",
	TAnd:        "&&",
	TOr:         "||",
	TNot:        "!",
	TBitAnd:     "&",
	TBitOr:      "|",
	TBitXor:     "^",
	TBitClear:   "&^",
}

func (t TokenType) String() string {
	name, ok := tokenTypes[t]
	if ok {
		return name
	}
	return fmt.Sprintf("{TokenType %d}", t)
}

var binaryTypes = map[TokenType]ast.BinaryType{
	TMult:     ast.BinaryMult,
	TPlus:     ast.BinaryPlus,
	TMinus:    ast.BinaryMinus,
	TDiv:      ast.BinaryDiv,
	TMod:      ast.BinaryMod,
	TLt:       ast.BinaryLt,
	TLe:       ast.BinaryLe,
	TGt:       ast.BinaryGt,
	TGe:       ast.BinaryGe,
	TEq:       ast.BinaryEq,
	TNeq:      ast.BinaryNeq,
	TAnd:      ast.BinaryAnd,
	TOr:       ast.BinaryOr,
	TBitAnd:   ast.BinaryBand,
	TBitOr:    ast.BinaryBor,
	TBitXor:   ast.BinaryBxor,
	TBitClear: ast.BinaryBclear,
	TLshift:   ast.BinaryLshift,
	TRshift:   ast.BinaryRshift,
}

// BinaryType returns token's binary type.
func (t TokenType) BinaryType() ast.BinaryType {
	bt, ok := binaryTypes[t]
	if ok {
		return bt
	}
	panic(fmt.Sprintf("Invalid binary operator %s", t))
}

var symbols = map[string]TokenType{
	"import":  TSymImport,
	"const":   TSymConst,
	"type":    TSymType,
	"for":     TSymFor,
	"else":    TSymElse,
	"func":    TSymFunc,
	"if":      TSymIf,
	"package": TSymPackage,
	"return":  TSymReturn,
	"struct":  TSymStruct,
	"var":     TSymVar,
}

// Token specifies an input token.
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

// Lexer implements MPCL lexical analyzer.
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
	lastComment Comment
}

// NewLexer creates a new lexer for the input.
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

// ReadRune reads the next input rune.
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

// UnreadRune unreads the last rune.
func (l *Lexer) UnreadRune() error {
	l.point, l.unreadPoint = l.unreadPoint, l.point
	l.unread = true
	return nil
}

// FlushEOL discards all remaining input from the current source code
// line.
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

// Get gets the next token.
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
					return l.Token(TPlus), nil
				}
				return nil, err
			}
			switch r {
			case '+':
				return l.Token(TPlusPlus), nil
			case '=':
				return l.Token(TPlusEq), nil
			default:
				l.UnreadRune()
				return l.Token(TPlus), nil
			}

		case '-':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(TMinus), nil
				}
				return nil, err
			}
			switch r {
			case '-':
				return l.Token(TMinusMinus), nil
			case '=':
				return l.Token(TMinusEq), nil
			default:
				l.UnreadRune()
				return l.Token(TMinus), nil
			}

		case '*':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(TMult), nil
				}
				return nil, err
			}
			switch r {
			case '=':
				return l.Token(TMultEq), nil

			default:
				l.UnreadRune()
				return l.Token(TMult), nil
			}

		case '/':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(TDiv), nil
				}
				return nil, err
			}
			switch r {
			case '/':
				var comment []rune
				start := l.point
				for {
					r, _, err := l.ReadRune()
					if err != nil {
						return nil, err
					}
					if r == '\n' {
						break
					}
					comment = append(comment, r)
				}
				l.commentLine(string(comment), start)
				continue

			case '=':
				return l.Token(TDivEq), nil

			default:
				l.UnreadRune()
				return l.Token(TDiv), nil
			}
		case '%':
			return l.Token(TMod), nil
		case '(':
			return l.Token(TLParen), nil
		case ')':
			return l.Token(TRParen), nil
		case '{':
			return l.Token(TLBrace), nil
		case '}':
			return l.Token(TRBrace), nil
		case '[':
			return l.Token(TLBracket), nil
		case ']':
			return l.Token(TRBracket), nil
		case ',':
			return l.Token(TComma), nil
		case ';':
			return l.Token(TSemicolon), nil
		case '.':
			return l.Token(TDot), nil

		case '<':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(TLt), nil
				}
				return nil, err
			}
			switch r {
			case '=':
				return l.Token(TLe), nil
			case '<':
				return l.Token(TLshift), nil
			default:
				l.UnreadRune()
				return l.Token(TLt), nil
			}

		case '>':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(TGt), nil
				}
				return nil, err
			}
			switch r {
			case '=':
				return l.Token(TGe), nil
			case '>':
				return l.Token(TRshift), nil
			default:
				l.UnreadRune()
				return l.Token(TGt), nil
			}

		case '=':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(TAssign), nil
				}
				return nil, err
			}
			switch r {
			case '=':
				return l.Token(TEq), nil
			default:
				l.UnreadRune()
				return l.Token(TAssign), nil
			}

		case ':':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(TColon), nil
				}
			}
			switch r {
			case '=':
				return l.Token(TDefAssign), nil
			default:
				l.UnreadRune()
				return l.Token(TColon), nil
			}

		case '|':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(TBitOr), nil
				}
				return nil, err
			}
			switch r {
			case '|':
				return l.Token(TOr), nil
			default:
				l.UnreadRune()
				return l.Token(TBitOr), nil
			}

		case '&':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(TBitAnd), nil
				}
				return nil, err
			}
			switch r {
			case '&':
				return l.Token(TAnd), nil
			case '^':
				return l.Token(TBitClear), nil
			default:
				l.UnreadRune()
				return l.Token(TBitAnd), nil
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
			token := l.Token(TConstant)
			token.ConstVal = val
			return token, nil

		case '!':
			r, _, err := l.ReadRune()
			if err != nil {
				if err == io.EOF {
					return l.Token(TNot), nil
				}
				return nil, err
			}
			switch r {
			case '=':
				return l.Token(TNeq), nil
			default:
				l.UnreadRune()
				return l.Token(TNot), nil
			}

		case '^':
			return l.Token(TBitXor), nil

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
					if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
						l.UnreadRune()
						break
					}
					symbol += string(r)
				}
				tt, ok := symbols[symbol]
				if ok {
					return l.Token(tt), nil
				}
				if symbol == "true" || symbol == "false" {
					token := l.Token(TConstant)
					token.ConstVal = symbol == "true"
					return token, nil
				}

				token := l.Token(TIdentifier)
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
						token := l.Token(TConstant)
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
					}
					l.UnreadRune()
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
				token := l.Token(TConstant)

				if u <= math.MaxInt32 {
					token.ConstVal = int32(u)
				} else if u <= math.MaxInt64 {
					token.ConstVal = int64(u)
				} else {
					token.ConstVal = u
				}
				return token, nil
			}
			l.UnreadRune()
			return nil, fmt.Errorf("%s: unexpected character '%s'",
				l.point, string(r))
		}
	}
}

// Unget pushes the token back to the lexer input stream. The next
// call to Get will return it.
func (l *Lexer) Unget(t *Token) {
	l.ungot = t
}

// Token returns a new token for the argument token type.
func (l *Lexer) Token(t TokenType) *Token {
	return &Token{
		Type: t,
		From: l.tokenStart,
		To:   l.point,
	}
}

// Comment describes a comment.
type Comment struct {
	Start utils.Point
	End   utils.Point
	Lines []string
}

// Empty tests if the comment is empty.
func (c Comment) Empty() bool {
	return len(c.Lines) == 0
}

func (l *Lexer) commentLine(line string, loc utils.Point) {
	line = strings.TrimSpace(line)
	if l.lastComment.Empty() || l.lastComment.End.Line+1 != loc.Line {
		l.lastComment = Comment{
			Start: loc,
			End:   loc,
			Lines: []string{line},
		}
	} else {
		l.lastComment.End = loc
		l.lastComment.Lines = append(l.lastComment.Lines, line)
	}
}

// Annotations returns the annotations immediately preceding the
// current lexer location.
func (l *Lexer) Annotations(loc utils.Point) ast.Annotations {
	if l.lastComment.Empty() || l.lastComment.End.Line+1 != loc.Line {
		return nil
	}
	return l.lastComment.Lines
}
