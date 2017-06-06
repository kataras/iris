// Package lexer provides a handlebars tokenizer.
package lexer

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// References:
//   - https://github.com/wycats/handlebars.js/blob/master/src/handlebars.l
//   - https://github.com/golang/go/blob/master/src/text/template/parse/lex.go

const (
	// Mustaches detection
	escapedEscapedOpenMustache  = "\\\\{{"
	escapedOpenMustache         = "\\{{"
	openMustache                = "{{"
	closeMustache               = "}}"
	closeStripMustache          = "~}}"
	closeUnescapedStripMustache = "}~}}"
)

const eof = -1

// lexFunc represents a function that returns the next lexer function.
type lexFunc func(*Lexer) lexFunc

// Lexer is a lexical analyzer.
type Lexer struct {
	input    string     // input to scan
	name     string     // lexer name, used for testing purpose
	tokens   chan Token // channel of scanned tokens
	nextFunc lexFunc    // the next function to execute

	pos   int // current byte position in input string
	line  int // current line position in input string
	width int // size of last rune scanned from input string
	start int // start position of the token we are scanning

	// the shameful contextual properties needed because `nextFunc` is not enough
	closeComment *regexp.Regexp // regexp to scan close of current comment
	rawBlock     bool           // are we parsing a raw block content ?
}

var (
	lookheadChars        = `[\s` + regexp.QuoteMeta("=~}/)|") + `]`
	literalLookheadChars = `[\s` + regexp.QuoteMeta("~})") + `]`

	// characters not allowed in an identifier
	unallowedIDChars = " \n\t!\"#%&'()*+,./;<=>@[\\]^`{|}~"

	// regular expressions
	rID                  = regexp.MustCompile(`^[^` + regexp.QuoteMeta(unallowedIDChars) + `]+`)
	rDotID               = regexp.MustCompile(`^\.` + lookheadChars)
	rTrue                = regexp.MustCompile(`^true` + literalLookheadChars)
	rFalse               = regexp.MustCompile(`^false` + literalLookheadChars)
	rOpenRaw             = regexp.MustCompile(`^\{\{\{\{`)
	rCloseRaw            = regexp.MustCompile(`^\}\}\}\}`)
	rOpenEndRaw          = regexp.MustCompile(`^\{\{\{\{/`)
	rOpenEndRawLookAhead = regexp.MustCompile(`\{\{\{\{/`)
	rOpenUnescaped       = regexp.MustCompile(`^\{\{~?\{`)
	rCloseUnescaped      = regexp.MustCompile(`^\}~?\}\}`)
	rOpenBlock           = regexp.MustCompile(`^\{\{~?#`)
	rOpenEndBlock        = regexp.MustCompile(`^\{\{~?/`)
	rOpenPartial         = regexp.MustCompile(`^\{\{~?>`)
	// {{^}} or {{else}}
	rInverse          = regexp.MustCompile(`^(\{\{~?\^\s*~?\}\}|\{\{~?\s*else\s*~?\}\})`)
	rOpenInverse      = regexp.MustCompile(`^\{\{~?\^`)
	rOpenInverseChain = regexp.MustCompile(`^\{\{~?\s*else`)
	// {{ or {{&
	rOpen            = regexp.MustCompile(`^\{\{~?&?`)
	rClose           = regexp.MustCompile(`^~?\}\}`)
	rOpenBlockParams = regexp.MustCompile(`^as\s+\|`)
	// {{!--  ... --}}
	rOpenCommentDash  = regexp.MustCompile(`^\{\{~?!--\s*`)
	rCloseCommentDash = regexp.MustCompile(`^\s*--~?\}\}`)
	// {{! ... }}
	rOpenComment  = regexp.MustCompile(`^\{\{~?!\s*`)
	rCloseComment = regexp.MustCompile(`^\s*~?\}\}`)
)

// Scan scans given input.
//
// Tokens can then be fetched sequentially thanks to NextToken() function on returned lexer.
func Scan(input string) *Lexer {
	return scanWithName(input, "")
}

// scanWithName scans given input, with a name used for testing
//
// Tokens can then be fetched sequentially thanks to NextToken() function on returned lexer.
func scanWithName(input string, name string) *Lexer {
	result := &Lexer{
		input:  input,
		name:   name,
		tokens: make(chan Token),
		line:   1,
	}

	go result.run()

	return result
}

// Collect scans and collect all tokens.
//
// This should be used for debugging purpose only. You should use Scan() and lexer.NextToken() functions instead.
func Collect(input string) []Token {
	var result []Token

	l := Scan(input)
	for {
		token := l.NextToken()
		result = append(result, token)

		if token.Kind == TokenEOF || token.Kind == TokenError {
			break
		}
	}

	return result
}

// NextToken returns the next scanned token.
func (l *Lexer) NextToken() Token {
	result := <-l.tokens

	return result
}

// run starts lexical analysis
func (l *Lexer) run() {
	for l.nextFunc = lexContent; l.nextFunc != nil; {
		l.nextFunc = l.nextFunc(l)
	}
}

// next returns next character from input, or eof of there is nothing left to scan
func (l *Lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width

	return r
}

func (l *Lexer) produce(kind TokenKind, val string) {
	l.tokens <- Token{kind, val, l.start, l.line}

	// scanning a new token
	l.start = l.pos

	// update line number
	l.line += strings.Count(val, "\n")
}

// emit emits a new scanned token
func (l *Lexer) emit(kind TokenKind) {
	l.produce(kind, l.input[l.start:l.pos])
}

// emitContent emits scanned content
func (l *Lexer) emitContent() {
	if l.pos > l.start {
		l.emit(TokenContent)
	}
}

// emitString emits a scanned string
func (l *Lexer) emitString(delimiter rune) {
	str := l.input[l.start:l.pos]

	// replace escaped delimiters
	str = strings.Replace(str, "\\"+string(delimiter), string(delimiter), -1)

	l.produce(TokenString, str)
}

// peek returns but does not consume the next character in the input
func (l *Lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one character
//
// WARNING: Can only be called once per call of next
func (l *Lexer) backup() {
	l.pos -= l.width
}

// ignoreskips all characters that have been scanned up to current position
func (l *Lexer) ignore() {
	l.start = l.pos
}

// accept scans the next character if it is included in given string
func (l *Lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}

	l.backup()

	return false
}

// acceptRun scans all following characters that are part of given string
func (l *Lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}

	l.backup()
}

// errorf emits an error token
func (l *Lexer) errorf(format string, args ...interface{}) lexFunc {
	l.tokens <- Token{TokenError, fmt.Sprintf(format, args...), l.start, l.line}
	return nil
}

// isString returns true if content at current scanning position starts with given string
func (l *Lexer) isString(str string) bool {
	return strings.HasPrefix(l.input[l.pos:], str)
}

// findRegexp returns the first string from current scanning position that matches given regular expression
func (l *Lexer) findRegexp(r *regexp.Regexp) string {
	return r.FindString(l.input[l.pos:])
}

// indexRegexp returns the index of the first string from current scanning position that matches given regular expression
//
// It returns -1 if not found
func (l *Lexer) indexRegexp(r *regexp.Regexp) int {
	loc := r.FindStringIndex(l.input[l.pos:])
	if loc == nil {
		return -1
	}
	return loc[0]
}

// lexContent scans content (ie: not between mustaches)
func lexContent(l *Lexer) lexFunc {
	var next lexFunc

	if l.rawBlock {
		if i := l.indexRegexp(rOpenEndRawLookAhead); i != -1 {
			// {{{{/
			l.rawBlock = false
			l.pos += i

			next = lexOpenMustache
		} else {
			return l.errorf("Unclosed raw block")
		}
	} else if l.isString(escapedEscapedOpenMustache) {
		// \\{{

		// emit content with only one escaped escape
		l.next()
		l.emitContent()

		// ignore second escaped escape
		l.next()
		l.ignore()

		next = lexContent
	} else if l.isString(escapedOpenMustache) {
		// \{{
		next = lexEscapedOpenMustache
	} else if str := l.findRegexp(rOpenCommentDash); str != "" {
		// {{!--
		l.closeComment = rCloseCommentDash

		next = lexComment
	} else if str := l.findRegexp(rOpenComment); str != "" {
		// {{!
		l.closeComment = rCloseComment

		next = lexComment
	} else if l.isString(openMustache) {
		// {{
		next = lexOpenMustache
	}

	if next != nil {
		// emit scanned content
		l.emitContent()

		// scan next token
		return next
	}

	// scan next rune
	if l.next() == eof {
		// emit scanned content
		l.emitContent()

		// this is over
		l.emit(TokenEOF)
		return nil
	}

	// continue content scanning
	return lexContent
}

// lexEscapedOpenMustache scans \{{
func lexEscapedOpenMustache(l *Lexer) lexFunc {
	// ignore escape character
	l.next()
	l.ignore()

	// scan mustaches
	for l.peek() == '{' {
		l.next()
	}

	return lexContent
}

// lexOpenMustache scans {{
func lexOpenMustache(l *Lexer) lexFunc {
	var str string
	var tok TokenKind

	nextFunc := lexExpression

	if str = l.findRegexp(rOpenEndRaw); str != "" {
		tok = TokenOpenEndRawBlock
	} else if str = l.findRegexp(rOpenRaw); str != "" {
		tok = TokenOpenRawBlock
		l.rawBlock = true
	} else if str = l.findRegexp(rOpenUnescaped); str != "" {
		tok = TokenOpenUnescaped
	} else if str = l.findRegexp(rOpenBlock); str != "" {
		tok = TokenOpenBlock
	} else if str = l.findRegexp(rOpenEndBlock); str != "" {
		tok = TokenOpenEndBlock
	} else if str = l.findRegexp(rOpenPartial); str != "" {
		tok = TokenOpenPartial
	} else if str = l.findRegexp(rInverse); str != "" {
		tok = TokenInverse
		nextFunc = lexContent
	} else if str = l.findRegexp(rOpenInverse); str != "" {
		tok = TokenOpenInverse
	} else if str = l.findRegexp(rOpenInverseChain); str != "" {
		tok = TokenOpenInverseChain
	} else if str = l.findRegexp(rOpen); str != "" {
		tok = TokenOpen
	} else {
		// this is rotten
		panic("Current pos MUST be an opening mustache")
	}

	l.pos += len(str)
	l.emit(tok)

	return nextFunc
}

// lexCloseMustache scans }} or ~}}
func lexCloseMustache(l *Lexer) lexFunc {
	var str string
	var tok TokenKind

	if str = l.findRegexp(rCloseRaw); str != "" {
		// }}}}
		tok = TokenCloseRawBlock
	} else if str = l.findRegexp(rCloseUnescaped); str != "" {
		// }}}
		tok = TokenCloseUnescaped
	} else if str = l.findRegexp(rClose); str != "" {
		// }}
		tok = TokenClose
	} else {
		// this is rotten
		panic("Current pos MUST be a closing mustache")
	}

	l.pos += len(str)
	l.emit(tok)

	return lexContent
}

// lexExpression scans inside mustaches
func lexExpression(l *Lexer) lexFunc {
	// search close mustache delimiter
	if l.isString(closeMustache) || l.isString(closeStripMustache) || l.isString(closeUnescapedStripMustache) {
		return lexCloseMustache
	}

	// search some patterns before advancing scanning position

	// "as |"
	if str := l.findRegexp(rOpenBlockParams); str != "" {
		l.pos += len(str)
		l.emit(TokenOpenBlockParams)
		return lexExpression
	}

	// ..
	if l.isString("..") {
		l.pos += len("..")
		l.emit(TokenID)
		return lexExpression
	}

	// .
	if str := l.findRegexp(rDotID); str != "" {
		l.pos += len(".")
		l.emit(TokenID)
		return lexExpression
	}

	// true
	if str := l.findRegexp(rTrue); str != "" {
		l.pos += len("true")
		l.emit(TokenBoolean)
		return lexExpression
	}

	// false
	if str := l.findRegexp(rFalse); str != "" {
		l.pos += len("false")
		l.emit(TokenBoolean)
		return lexExpression
	}

	// let's scan next character
	switch r := l.next(); {
	case r == eof:
		return l.errorf("Unclosed expression")
	case isIgnorable(r):
		return lexIgnorable
	case r == '(':
		l.emit(TokenOpenSexpr)
	case r == ')':
		l.emit(TokenCloseSexpr)
	case r == '=':
		l.emit(TokenEquals)
	case r == '@':
		l.emit(TokenData)
	case r == '"' || r == '\'':
		l.backup()
		return lexString
	case r == '/' || r == '.':
		l.emit(TokenSep)
	case r == '|':
		l.emit(TokenCloseBlockParams)
	case r == '+' || r == '-' || (r >= '0' && r <= '9'):
		l.backup()
		return lexNumber
	case r == '[':
		return lexPathLiteral
	case strings.IndexRune(unallowedIDChars, r) < 0:
		l.backup()
		return lexIdentifier
	default:
		return l.errorf("Unexpected character in expression: '%c'", r)
	}

	return lexExpression
}

// lexComment scans {{!-- or {{!
func lexComment(l *Lexer) lexFunc {
	if str := l.findRegexp(l.closeComment); str != "" {
		l.pos += len(str)
		l.emit(TokenComment)

		return lexContent
	}

	if r := l.next(); r == eof {
		return l.errorf("Unclosed comment")
	}

	return lexComment
}

// lexIgnorable scans all following ignorable characters
func lexIgnorable(l *Lexer) lexFunc {
	for isIgnorable(l.peek()) {
		l.next()
	}
	l.ignore()

	return lexExpression
}

// lexString scans a string
func lexString(l *Lexer) lexFunc {
	// get string delimiter
	delim := l.next()
	var prev rune

	// ignore delimiter
	l.ignore()

	for {
		r := l.next()
		if r == eof || r == '\n' {
			return l.errorf("Unterminated string")
		}

		if (r == delim) && (prev != '\\') {
			break
		}

		prev = r
	}

	// remove end delimiter
	l.backup()

	// emit string
	l.emitString(delim)

	// skip end delimiter
	l.next()
	l.ignore()

	return lexExpression
}

// lexNumber scans a number: decimal, octal, hex, float, or imaginary. This
// isn't a perfect number scanner - for instance it accepts "." and "0x0.2"
// and "089" - but when it's wrong the input is invalid and the parser (via
// strconv) will notice.
//
// NOTE: borrowed from https://github.com/golang/go/tree/master/src/text/template/parse/lex.go
func lexNumber(l *Lexer) lexFunc {
	if !l.scanNumber() {
		return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
	}
	if sign := l.peek(); sign == '+' || sign == '-' {
		// Complex: 1+2i. No spaces, must end in 'i'.
		if !l.scanNumber() || l.input[l.pos-1] != 'i' {
			return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
		}
		l.emit(TokenNumber)
	} else {
		l.emit(TokenNumber)
	}
	return lexExpression
}

// scanNumber scans a number
//
// NOTE: borrowed from https://github.com/golang/go/tree/master/src/text/template/parse/lex.go
func (l *Lexer) scanNumber() bool {
	// Optional leading sign.
	l.accept("+-")

	// Is it hex?
	digits := "0123456789"

	if l.accept("0") && l.accept("xX") {
		digits = "0123456789abcdefABCDEF"
	}

	l.acceptRun(digits)

	if l.accept(".") {
		l.acceptRun(digits)
	}

	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789")
	}

	// Is it imaginary?
	l.accept("i")

	// Next thing mustn't be alphanumeric.
	if isAlphaNumeric(l.peek()) {
		l.next()
		return false
	}

	return true
}

// lexIdentifier scans an ID
func lexIdentifier(l *Lexer) lexFunc {
	str := l.findRegexp(rID)
	if len(str) == 0 {
		// this is rotten
		panic("Identifier expected")
	}

	l.pos += len(str)
	l.emit(TokenID)

	return lexExpression
}

// lexPathLiteral scans an [ID]
func lexPathLiteral(l *Lexer) lexFunc {
	for {
		r := l.next()
		if r == eof || r == '\n' {
			return l.errorf("Unterminated path literal")
		}

		if r == ']' {
			break
		}
	}

	l.emit(TokenID)

	return lexExpression
}

// isIgnorable returns true if given character is ignorable (ie. whitespace of line feed)
func isIgnorable(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
//
// NOTE borrowed from https://github.com/golang/go/tree/master/src/text/template/parse/lex.go
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
