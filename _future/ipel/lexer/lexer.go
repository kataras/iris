package lexer

import (
	"gopkg.in/kataras/iris.v6/_future/ipel/token"
)

type Lexer struct {
	input   string
	pos     int  // current pos in input, current char
	readPos int  // current reading pos in input, after current char
	ch      byte // current char under examination
}

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
	l.readPos += 1
}

func (l *Lexer) NextToken() (t token.Token) {
	l.skipWhitespace()

	switch l.ch {
	case '{':
		t = l.newTokenRune(token.LBRACE, l.ch)
	case '}':
		t = l.newTokenRune(token.RBRACE, l.ch)
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
		t = l.newTokenRune(token.COLON, l.ch)
	case '(':
		t = l.newTokenRune(token.LPAREN, l.ch)
	case ')':
		t = l.newTokenRune(token.RPAREN, l.ch)
	case ',':
		t = l.newTokenRune(token.COMMA, l.ch)
		// literals
	case 0:
		t.Literal = ""
		t.Type = token.EOF
	default:
		// letters
		if isLetter(l.ch) {
			lit := l.readIdentifier()
			typ := token.LookupIdent(lit)
			t = l.newToken(typ, lit)
			return
			// numbers
		} else if isDigit(l.ch) {
			lit := l.readNumber()
			t = l.newToken(token.INT, lit)
			return
		} else {
			t = l.newTokenRune(token.ILLEGAL, l.ch)
		}
	}
	l.readChar() // set the pos to the next
	return
}

func (l *Lexer) newToken(tokenType token.TokenType, lit string) token.Token {
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

func (l *Lexer) newTokenRune(tokenType token.TokenType, ch byte) token.Token {
	return l.newToken(tokenType, string(ch))
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	pos := l.pos
	for isLetter(l.ch) {
		l.readChar()
	}
	return l.input[pos:l.pos]
}

func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
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
