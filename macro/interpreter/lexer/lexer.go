package lexer

import (
	"github.com/kataras/iris/macro/interpreter/token"
)

// Lexer helps us to read/scan characters of a source and resolve their token types.
type Lexer struct {
	input   string
	pos     int  // current pos in input, current char
	readPos int  // current reading pos in input, after current char
	ch      byte // current char under examination
}

// New takes a source, series of chars, and returns
// a new, ready to read from the first letter, lexer.
func New(src string) *Lexer {
	l := &Lexer{
		input: src,
	}
	// step to the first character in order to be ready
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
}

const (
	// Begin is the symbol which lexer should scan forward to.
	Begin = '{' // token.LBRACE
	// End is the symbol which lexer should stop scanning.
	End = '}' // token.RBRACE
)

func resolveTokenType(ch byte) token.Type {
	switch ch {
	case Begin:
		return token.LBRACE
	case End:
		return token.RBRACE
	// Let's keep it simple, no evaluation for logical operators, we are not making a new programming language, keep it simple makis.
	// ||
	// case '|':
	// 	if l.peekChar() == '|' {
	// 		ch := l.ch
	// 		l.readChar()
	// 		t = token.Token{Type: token.OR, Literal: string(ch) + string(l.ch)}
	// 	}
	// ==
	case ':':
		return token.COLON
	case '(':
		return token.LPAREN
	case ')':
		return token.RPAREN
	case ',':
		return token.COMMA
		// literals
	case 0:
		return token.EOF
	default:
		return token.IDENT //
	}

}

// NextToken returns the next token in the series of characters.
// It can be a single symbol, a token type or a literal.
// It's able to return an EOF token too.
//
// It moves the cursor forward.
func (l *Lexer) NextToken() (t token.Token) {
	l.skipWhitespace()
	typ := resolveTokenType(l.ch)
	t.Type = typ
	switch typ {
	case token.EOF:
		t.Literal = ""
	case token.IDENT:
		if isLetter(l.ch) {
			// letters
			lit := l.readIdentifier()
			typ := token.LookupIdent(lit)
			t = l.newToken(typ, lit)
			return
		}
		if isDigit(l.ch) {
			// numbers
			lit := l.readNumber()
			t = l.newToken(token.INT, lit)
			return
		}

		t = l.newTokenRune(token.ILLEGAL, l.ch)
	default:
		t = l.newTokenRune(typ, l.ch)
	}
	l.readChar() // set the pos to the next
	return
}

// NextDynamicToken doesn't cares about the grammar.
// It reads numbers or any unknown symbol,
// it's being used by parser to skip all characters
// between parameter function's arguments inside parenthesis,
// in order to allow custom regexp on the end-language too.
//
// It moves the cursor forward.
func (l *Lexer) NextDynamicToken() (t token.Token) {
	// calculate anything, even spaces.

	// numbers
	lit := l.readNumber()
	if lit != "" {
		return l.newToken(token.INT, lit)
	}

	lit = l.readIdentifierFuncArgument()
	return l.newToken(token.IDENT, lit)
}

// used to skip any illegal token if inside parenthesis, used to be able to set custom regexp inside a func.
func (l *Lexer) readIdentifierFuncArgument() string {
	pos := l.pos
	for resolveTokenType(l.ch) != token.RPAREN {
		l.readChar()
	}

	return l.input[pos:l.pos]
}

// PeekNextTokenType returns only the token type
// of the next character and it does not move forward the cursor.
// It's being used by parser to recognise empty functions, i.e `even()`
// as valid functions with zero input arguments.
func (l *Lexer) PeekNextTokenType() token.Type {
	if len(l.input)-1 > l.pos {
		ch := l.input[l.pos]
		return resolveTokenType(ch)
	}
	return resolveTokenType(0) // EOF
}

func (l *Lexer) newToken(tokenType token.Type, lit string) token.Token {
	t := token.Token{
		Type:    tokenType,
		Literal: lit,
		Start:   l.pos,
		End:     l.pos,
	}
	// remember, l.pos is the last char
	// and we want to include both start and end
	// in order to be easy to the user to see by just marking the expression
	if l.pos > 1 && len(lit) > 1 {
		t.End = l.pos - 1
		t.Start = t.End - len(lit) + 1
	}

	return t
}

func (l *Lexer) newTokenRune(tokenType token.Type, ch byte) token.Token {
	return l.newToken(tokenType, string(ch))
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	pos := l.pos
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[pos:l.pos]
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func (l *Lexer) readNumber() string {
	pos := l.pos
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[pos:l.pos]
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
