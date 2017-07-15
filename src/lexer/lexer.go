package lexer

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"unicode"

	"github.com/nitrogen-lang/nitrogen/src/token"
)

type Lexer struct {
	input     *bufio.Reader
	curCh     rune // current char under examination
	peekCh    rune // peek character
	lastToken token.Token
}

func New(reader io.Reader) *Lexer {
	l := &Lexer{input: bufio.NewReader(reader)}
	// Populate both current and peek char
	l.readRune()
	l.readRune()
	return l
}

func NewString(input string) *Lexer {
	return New(strings.NewReader(input))
}

func (l *Lexer) readRune() {
	l.curCh = l.peekCh

	var err error
	l.peekCh, _, err = l.input.ReadRune()
	if err != nil {
		l.peekCh = 0
	}
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.devourWhitespaceNotNewLine()

	switch l.curCh {
	case '\n':
		if l.needSemicolon() {
			tok = newToken(token.SEMICOLON, ';')
		} else {
			l.devourWhitespace()
			return l.NextToken()
		}

	// Operators
	case '+':
		tok = newToken(token.PLUS, l.curCh)
	case '-':
		tok = newToken(token.MINUS, l.curCh)
	case '*':
		tok = newToken(token.ASTERISK, l.curCh)
	case '%':
		tok = newToken(token.MODULUS, l.curCh)
	case '/':
		if l.peekChar() == '/' {
			l.readRune()
			tok = l.readSingleLineComment()
		} else if l.peekChar() == '*' {
			l.readRune()
			tok = l.readMultiLineComment()
		} else {
			tok = newToken(token.SLASH, l.curCh)
		}
	case '!':
		if l.peekChar() == '=' {
			l.readRune()
			tok = token.Token{
				Type:    token.NOT_EQ,
				Literal: "!=",
			}
		} else {
			tok = newToken(token.BANG, l.curCh)
		}

	// Equality
	case '=':
		if l.peekChar() == '=' {
			l.readRune()
			tok = token.Token{
				Type:    token.EQ,
				Literal: "==",
			}
		} else {
			tok = newToken(token.ASSIGN, l.curCh)
		}
	case '<':
		tok = newToken(token.LT, l.curCh)
	case '>':
		tok = newToken(token.GT, l.curCh)

	// Control characters
	case ',':
		tok = newToken(token.COMMA, l.curCh)
	case ';':
		tok = newToken(token.SEMICOLON, l.curCh)
	case ':':
		tok = newToken(token.COLON, l.curCh)

	// Groupings
	case '(':
		tok = newToken(token.LPAREN, l.curCh)
	case ')':
		tok = newToken(token.RPAREN, l.curCh)
	case '{':
		tok = newToken(token.LBRACE, l.curCh)
	case '}':
		tok = newToken(token.RBRACE, l.curCh)
	case '[':
		tok = newToken(token.LSQUARE, l.curCh)
	case ']':
		tok = newToken(token.RSQUARE, l.curCh)

	case '"':
		tok = l.readString()
	case '\'':
		tok = l.readRawString()
	case '#':
		tok = l.readSingleLineComment()
	case '\\':
		if l.peekCh == 'x' {
			l.readRune()
			tok = l.readNumber()
		} else {
			tok = newToken(token.ILLEGAL, l.curCh)
		}
	case 0:
		if l.needSemicolon() {
			tok = newToken(token.SEMICOLON, ';')
		} else {
			tok.Literal = ""
			tok.Type = token.EOF
		}

	default:
		if isLetter(l.curCh) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			l.lastToken = tok
			return tok
		} else if isDigit(l.curCh) {
			tok = l.readNumber()
			l.lastToken = tok
			return tok
		}

		tok = newToken(token.ILLEGAL, l.curCh)
	}

	l.readRune()
	l.lastToken = tok
	return tok
}

func (l *Lexer) peekChar() rune {
	return l.peekCh
}

func (l *Lexer) needSemicolon() bool {
	return l.lastTokenWas(
		token.IDENT,
		token.INT,
		token.FLOAT,
		token.STRING,
		token.NULL,
		token.RETURN,
		token.RPAREN,
		token.RSQUARE,
		token.RBRACE)
}

func (l *Lexer) readIdentifier() string {
	var ident bytes.Buffer
	for isIdent(l.curCh) || isDigit(l.curCh) {
		ident.WriteRune(l.curCh)
		l.readRune()
	}
	return ident.String()
}

func (l *Lexer) readString() token.Token {
	var ident bytes.Buffer
	l.readRune() // Go past the starting double quote

	for l.curCh != '"' {
		if l.curCh == '\n' {
			return token.Token{
				Literal: "Newline not allowed in string",
				Type:    token.ILLEGAL,
			}
		}

		if l.curCh == '\\' {
			l.readRune()
			switch l.curCh {
			case 'b': // backspace
				ident.WriteRune('\b')
			case 'n': // newline
				ident.WriteRune('\n')
			case 'r': // carriage return
				ident.WriteRune('\r')
			case 't': // horizontal tab
				ident.WriteRune('\t')
			case 'v': // vertical tab
				ident.WriteRune('\v')
			case 'f': // form feed
				ident.WriteRune('\f')
			case '\\': // back slash
				ident.WriteRune('\\')
			case '"': // double quote
				ident.WriteRune('"')
			default:
				ident.WriteByte('\\')
				ident.WriteRune(l.curCh)
			}
			l.readRune()
			continue
		}
		ident.WriteRune(l.curCh)
		l.readRune()
	}

	return token.Token{
		Literal: ident.String(),
		Type:    token.STRING,
	}
}

func (l *Lexer) readRawString() token.Token {
	var ident bytes.Buffer
	l.readRune() // Go past the starting double quote

	for l.curCh != '\'' {
		if l.curCh == '\\' && l.peekCh == '\'' {
			l.readRune() // Go past backslash so the next line will write a single quote
		}
		ident.WriteRune(l.curCh)
		l.readRune()
	}

	return token.Token{
		Literal: ident.String(),
		Type:    token.STRING,
	}
}

func (l *Lexer) readNumber() token.Token {
	var number bytes.Buffer
	base := ""
	tokenType := token.INT

	if l.curCh == 'x' {
		base = "0x"
		l.readRune()
	}

	if l.curCh == '.' {
		l.readRune()
		return token.Token{
			Type:    token.ILLEGAL,
			Literal: "Invalid float literal",
		}
	}

	for isDigit(l.curCh) || isHexDigit(l.curCh) {
		if l.curCh == '.' {
			if tokenType != token.INT {
				return token.Token{
					Type:    token.ILLEGAL,
					Literal: "Invalid float literal",
				}
			}
			tokenType = token.FLOAT
		}

		number.WriteRune(l.curCh)
		l.readRune()
	}

	return token.Token{
		Type:    token.TokenType(tokenType),
		Literal: base + number.String(),
	}
}

func (l *Lexer) readSingleLineComment() token.Token {
	var com bytes.Buffer
	l.readRune() // Go over # or / characters

	for l.curCh != '\n' {
		com.WriteRune(l.curCh)
		l.readRune()
	}

	return token.Token{
		Literal: strings.TrimSpace(com.String()),
		Type:    token.COMMENT,
	}
}

func (l *Lexer) readMultiLineComment() token.Token {
	var com bytes.Buffer
	l.readRune() // Go over * character

	for l.curCh != 0 {
		if l.curCh == '*' && l.peekChar() == '/' {
			l.readRune() // Skip *
			break
		}

		com.WriteRune(l.curCh)
		l.readRune()
	}

	return token.Token{
		Literal: com.String(),
		Type:    token.COMMENT,
	}
}

func (l *Lexer) devourWhitespace() {
	for isWhitespace(l.curCh) {
		l.readRune()
	}
}

func (l *Lexer) devourWhitespaceNotNewLine() {
	for l.curCh != '\n' && isWhitespace(l.curCh) {
		l.readRune()
	}
}

func newToken(tokenType token.TokenType, ch rune) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch)}
}

// Identifiers must start with a letter
func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || unicode.IsLetter(ch)
}

// After the first letter, an ident can be a letter or number
func isIdent(ch rune) bool {
	return isLetter(ch) || (ch != '.' && isDigit(ch)) // A period is not a valid identifier name
}

// Only Latin numbers
func isDigit(ch rune) bool {
	return ('0' <= ch && ch <= '9') || ch == '.'
}

func isHexDigit(ch rune) bool {
	return ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func (l *Lexer) lastTokenWas(types ...token.TokenType) bool {
	for _, t := range types {
		if l.lastToken.Type == t {
			return true
		}
	}
	return false
}
