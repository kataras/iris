package jade

import (
	"strings"
)

func lexIndents(l *lexer) stateFn {
	d := l.indents()
	if d == -1 {
		l.depth = 0
		l.emit(itemEmptyLine)
	} else {
		l.depth = d
		l.emit(itemIdent)
	}
	return lexTags
}
func (l *lexer) indents() (depth int) {
	for {
		switch l.next() {
		case ' ':
			depth += 1
		case '\t':
			depth += TabSize
		case '\r':
			// skip
		case '\n':
			return -1
		default:
			l.backup()
			return
		}
	}
}

func lexEndLine(l *lexer) stateFn {
	switch r := l.next(); {
	case r == '\r':
		if l.next() == '\n' {
			l.emit(itemEndL)
			return lexIndents
		}
		return l.errorf("lexTags: standalone '\\r' ")
	case r == '\n':
		l.emit(itemEndL)
		return lexIndents
	case r == eof:
		l.depth = 0
		l.emit(itemEOF)
		return nil
	default:
		return l.errorf("lexEndLine: unexpected token %#U `%s`", r, string(r))
	}
}

// lexTags scans tags.
func lexTags(l *lexer) stateFn {
	switch r := l.next(); {

	case isEndOfLine(r), r == eof:
		l.backup()
		return lexEndLine
	case r == ' ' || r == '\t':
		l.backup()
		return lexIndents
	//
	//
	case r == '.':
		n := l.skipSpaces()
		if n == 0 {
			l.emit(itemDiv)
			return lexClass
		}
		if n == -1 {
			l.ignore()
			return lexLongText
		}
		return l.errorf("lexTags: class name cannot start with a space.")
	case r == '#':
		l.emit(itemDiv)
		return lexID
	case r == ':':
		l.ignore()
		if l.emitWordByType(itemFilter) {
			return lexFilter
		}
		return l.errorf("lexTags: expect filter name")
	case r == '|':
		r = l.next()
		if r != ' ' {
			l.backup()
		}
		l.ignore()
		return lexText
	case r == '<':
		l.emitLineByType(itemHTMLTag)
		return lexEndLine
	case r == '+':
		l.skipSpaces()
		l.ignore()
		if l.emitWordByType(itemMixinCall) {
			return lexAfterTag
		}
		return l.errorf("lexTags: expect mixin name")
	case r == '/':
		return lexComment
	case r == '-':
		l.ignore()
		return lexCode
	case r == '=':
		l.skipSpaces()
		l.ignore()
		l.emitLineByType(itemCodeBuffered)
		return lexEndLine
	case r == '!':
		np := l.next()
		if np == '=' {
			l.skipSpaces()
			l.ignore()
			l.emitLineByType(itemCodeUnescaped)
			return lexEndLine
		}
		if np == '!' && l.next() == '!' && l.depth == 0 {
			if l.skipSpaces() != -1 {
				l.ignore()
				l.emitLineByType(itemDoctype)
				return lexEndLine
			}
		}
		return l.errorf("expect '=' after '!'")
	case isAlphaNumeric(r):
		l.backup()
		return lexTagName
	default:
		return l.errorf("lexTags: unexpected token %#U `%s`", r, string(r))
	}
}

//
//

func lexID(l *lexer) stateFn {
	if l.emitWordByType(itemID) {
		return lexAfterTag
	}
	return l.errorf("lexID: expect id name")
}
func lexClass(l *lexer) stateFn {
	if l.emitWordByType(itemClass) {
		return lexAfterTag
	}
	return l.errorf("lexClass: expect class name")
}

func lexFilter(l *lexer) stateFn {
	l.multiline()
	l.emit(itemFilterText)
	return lexIndents
}

func lexCode(l *lexer) stateFn {
	if l.skipSpaces() == -1 {
		l.multiline()
		l.emit(itemCode)
		return lexIndents
	} else {
		l.ignore()
		l.emitLineByType(itemCode)
		return lexEndLine
	}
}
func lexComment(l *lexer) stateFn {
	sp := l.next()
	tp := l.peek()
	if sp == '/' {
		if tp == '-' {
			l.multiline()
			l.ignore()
			return lexIndents
		} else {
			l.ignore()
			l.multiline()
			l.emit(itemComment)
			return lexIndents
		}
	}
	return l.errorf("lexComment: unexpected token '%#U' expect '/'", sp)
}

//
//

func lexText(l *lexer) stateFn {
	if l.skipSpaces() == -1 {
		l.ignore()
		return lexEndLine
	}
	return text(l)
}
func lexLongText(l *lexer) stateFn {
	l.longtext = true
	return text(l)
}
func text(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == '\\':
			l.next()
			continue
		case r == '#':
			sp := l.peek()
			if sp == '[' {
				l.backup()
				if l.pos > l.start {
					l.emit(itemText)
				}
				l.next()
				l.next()
				l.skipSpaces()
				l.interpolation += 1
				l.depth += 1
				// l.emit(itemInterpolation)
				l.ignore()
				return lexTags
			}
			if sp == '{' {
				l.interpol(itemCodeBuffered)
			}
		case r == '$':
			sp := l.peek()
			if sp == '{' {
				l.interpol(itemCodeBuffered)
			}
		case r == '!':
			sp := l.peek()
			if sp == '{' {
				l.interpol(itemCodeUnescaped)
			}
		case r == ']':
			if l.interpolation > 0 {
				l.backup()
				if l.pos > l.start {
					l.emit(itemText)
				}
				l.next()
				// l.emit(itemInterpolationEnd)
				l.ignore()
				l.interpolation -= 1
				l.depth -= 1
			}
		case r == eof:
			l.backup()
			l.emit(itemText)
			return lexEndLine
		case r == '\n':
			if l.longtext {
				var (
					indent int
					pos    Pos
				)
				l.backup()
				pos = l.pos
				l.next()
				indent = l.indents()
				if indent != -1 {
					if indent < l.depth {
						l.pos = pos
						if l.pos > l.start {
							l.emit(itemText)
						}
						l.longtext = false
						return lexIndents
					}
				} else {
					l.backup()
				}
			} else {
				l.backup()
				if l.pos > l.start {
					l.emit(itemText)
				}
				return lexIndents
			}
		}
	}
}
func (l *lexer) interpol(item itemType) {
	l.backup()
	if l.pos > l.start {
		l.emit(itemText)
	}
	l.next()
	l.next()
	l.skipSpaces()
	l.ignore()
Loop:
	for {
		switch r := l.next(); {
		case r == '`':
			l.toStopRune('`', false)
		case r == '"':
			l.toStopRune('"', false)
		case r == '\'':
			l.toStopRune('\'', false)
		case r == '\n', r == eof:
			l.backup()
			l.errorf("interpolation error: expect '}'")
			return
		case r == '}':
			break Loop
		}
	}
	l.backup()
	l.emit(item)
	l.next()
	l.ignore()
}

func lexTagName(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			// absorb.
		default:
			l.backup()
			word := l.input[l.start:l.pos]
			if w, ok := key[word]; ok {
				switch w {
				case itemElse:
					l.emit(w)
					l.skipSpaces()
					l.ignore()
					return lexTags
				case itemDoctype, itemExtends:
					if l.depth == 0 {
						ss := l.skipSpaces()
						l.ignore()
						if ss != -1 {
							l.emitLineByType(w)
						} else if w == itemDoctype {
							l.emit(w)
						} else {
							return l.errorf("lexTagName: itemExtends need path ")
						}
						return lexEndLine
					} else {
						l.emit(itemTag)
					}
				case itemBlock:
					sp := l.skipSpaces()
					l.ignore()
					if sp == -1 {
						l.emit(itemMixinBlock)
					} else if strings.HasPrefix(l.input[l.pos:], "prepend ") {
						l.toStopRune(' ', true)
						l.skipSpaces()
						l.ignore()
						l.emitLineByType(itemBlockPrepend)
					} else if strings.HasPrefix(l.input[l.pos:], "append ") {
						l.toStopRune(' ', true)
						l.skipSpaces()
						l.ignore()
						l.emitLineByType(itemBlockAppend)
					} else {
						l.emitLineByType(itemBlock)
					}
					return lexEndLine
				case itemBlockAppend, itemBlockPrepend,
					itemIf, itemUnless, itemCase,
					itemEach, itemWhile, itemFor,
					itemInclude:

					l.skipSpaces()
					l.ignore()
					l.emitLineByType(w)
					return lexEndLine
				case itemMixin:
					l.skipSpaces()
					l.ignore()
					l.emitWordByType(w)
					return lexAfterTag
				case itemCaseWhen:
					l.skipSpaces()
					l.ignore()
					l.toStopRune(':', true)
					l.emit(w)
					return lexAfterTag
				default:
					l.emit(w)
				}
			} else {
				l.emit(itemTag)
			}
			return lexAfterTag
		}
	}
}

func lexAfterTag(l *lexer) stateFn {
	switch r := l.next(); {
	case r == '(':
		l.emit(itemAttrStart)
		return lexAttr
	case r == '/':
		l.emit(itemTagEnd)
		return lexAfterTag
	case r == ':':
		l.skipSpaces()
		l.ignore()
		l.depth += 1
		return lexTags
	case r == ' ' || r == '\t':
		l.ignore()
		l.depth += 1
		return lexText
	case r == ']':
		if l.interpolation > 0 {
			l.ignore()
			if l.pos > l.start {
				l.emit(itemText)
			}
			l.interpolation -= 1
			l.depth -= 1
			if l.longtext {
				return lexLongText
			} else {
				return lexText
			}
		}
		return l.errorf("lexAfterTag: %#U", r)
	case r == '=':
		l.skipSpaces()
		l.ignore()
		l.depth += 1
		l.emitLineByType(itemCodeBuffered)
		return lexEndLine
	case r == '!':
		if l.next() == '=' {
			l.skipSpaces()
			l.ignore()
			l.depth += 1
			l.emitLineByType(itemCodeUnescaped)
			return lexEndLine
		}
		return l.errorf("expect '=' after '!'")
	case r == '#':
		l.ignore()
		return lexID
	case r == '&':
		l.toStopRune(')', false)
		l.ignore() // TODO: now ignore div(data-bar="foo")&attributes({'data-foo': 'baz'})
		return lexAfterTag
	case r == '.':
		switch l.skipSpaces() {
		case 0:
			l.ignore()
			return lexClass
		case -1:
			if sp := l.next(); sp != eof {
				l.ignore()
				l.depth += 1
				return lexLongText
			}
			return lexEndLine
		default:
			l.ignore()
			l.depth += 1
			return lexText
		}
	case isEndOfLine(r), r == eof:
		l.backup()
		return lexEndLine
	default:
		return l.errorf("lexAfterTag: %#U", r)
	}
}

//
//

func lexAttr(l *lexer) stateFn {
	b1, b2, b3 := 0, 0, 0
	for {
		switch r := l.next(); {
		case r == '"' || r == '\'':
			l.toStopRune(r, false)
		case r == '`':
			for {
				r = l.next()
				if r == '`' {
					break
				}
			}
		case r == '(':
			b1 += 1
		case r == ')':
			b1 -= 1
			if b1 == -1 {
				if b2 != 0 || b3 != 0 {
					return l.errorf("lexAttrName: mismatched bracket")
				}
				l.backup()
				if l.pos > l.start {
					l.emit(itemAttr)
				}
				l.next()
				l.emit(itemAttrEnd)
				return lexAfterTag
			}
		case r == '[':
			b2 += 1
		case r == ']':
			b2 -= 1
			if b2 == -1 {
				return l.errorf("lexAttrName: mismatched bracket '['")
			}
		case r == '{':
			b3 += 1
		case r == '}':
			b3 -= 1
			if b3 == -1 {
				return l.errorf("lexAttrName: mismatched bracket '{'")
			}
		case r == ' ' || r == '\t':
			l.backup()
			if l.pos > l.start {
				l.emit(itemAttr)
			}
			l.skipSpaces()
			l.emit(itemAttrSpace)
		case r == '=':
			if l.peek() == '=' {
				l.toStopRune(' ', true)
				l.emit(itemAttr)
				continue
			}
			l.backup()
			l.emit(itemAttr)
			l.next()
			l.emit(itemAttrEqual)
		case r == '!':
			if l.peek() == '=' {
				l.backup()
				l.emit(itemAttr)
				l.next()
				l.next()
				l.emit(itemAttrEqualUn)
			}
		case r == ',' || r == '\n':
			if b1 == 0 && b2 == 0 && b3 == 0 {
				l.backup()
				if l.pos > l.start {
					l.emit(itemAttr)
				}
				l.next()
				l.emit(itemAttrComma)
			}
		case r == eof:
			return l.errorf("lexAttr: expected ')'")
		}
	}
}

//
//
//
//
//
//
//
//
//
//

func (l *lexer) emitWordByType(item itemType) bool {
	for {
		if !isAlphaNumeric(l.next()) {
			l.backup()
			break
		}
	}
	if l.pos > l.start {
		l.emit(item)
		return true
	}
	return false
}

func (l *lexer) emitLineByType(item itemType) bool {
	var r rune
	for {
		r = l.next()
		if r == '\n' || r == '\r' || r == eof {
			l.backup()
			if l.pos > l.start {
				l.emit(item)
				return true
			}
			return false
		}
	}
}

//

func (l *lexer) skipSpaces() (out int) {
	for {
		switch l.next() {
		case ' ', '\t':
			out += 1
		case '\n', eof:
			l.backup()
			return -1
		default:
			l.backup()
			return
		}
	}
}

func (l *lexer) toStopRune(stopRune rune, backup bool) {
	for {
		switch r := l.next(); {
		case r == stopRune:
			if backup {
				l.backup()
			}
			return
		case r == eof || r == '\r' || r == '\n':
			l.backup()
			return
		}
	}
}

func (l *lexer) multiline() {
	var (
		indent int
		pos    Pos
	)
	for {
		switch r := l.next(); {
		case r == '\n':
			l.backup()
			pos = l.pos
			l.next()
			indent = l.indents()
			if indent != -1 {
				if indent <= l.depth {
					l.pos = pos
					return
				}
			} else {
				l.backup()
			}
		case r == eof:
			l.backup()
			return
		}
	}
}
