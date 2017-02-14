package jade

import (
	"strings"
)

type mode int

const (
	mInterpolation mode = iota
	mInText
	mBrText
	mExtends
)
const (
	stText int = iota
	stInlineText
)

// itemType identifies the type of lex items.
type itemType int

const (
	itemError itemType = iota // error occurred; value is text of error
	itemEOF
	itemEndL
	itemEndTag
	itemEndAttr

	itemIdentSpace
	itemIdentTab

	itemTag           // html tag
	itemDiv           // html div for . or #
	itemInlineTag     // inline tags
	itemVoidTag       // self-closing tags
	itemInlineVoidTag // inline + self-closing tags
	itemComment

	itemID       // id    attribute
	itemClass    // class attribute
	itemAttr     // html  attribute value
	itemAttrN    // html  attribute value without quotes
	itemAttrName // html  attribute name
	itemAttrVoid // html  attribute without value

	itemParentIdent // Ident for 'tag:'
	itemChildIdent  // Ident for ']'
	itemText        // plain text
	itemEmptyLine   // empty line
	itemInlineText
	itemHTMLTag // html <tag>

	itemDoctype // Doctype tag
	itemBlank
	itemFilter
	itemAction       // from go template {{...}}
	itemActionEnd    // from go template {{...}} {{end}}
	itemInlineAction // title= .titleName

	itemExtends
	itemDefine
	itemBlock

	itemElse
	itemEnd
	itemIf
	itemRange
	itemNil
	itemTemplate
	itemWith
)

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for l.state = lexTags; l.state != nil; {
		l.state = l.state(l)
	}
}

func lexDoc(l *lexer) stateFn {
	switch l.next() {
	case eof:
		l.backup()
		l.emit(itemDoctype)
		l.next()
		l.emit(itemEOF)
		return nil
	case '\r', '\n':
		l.backup()
		l.emit(itemDoctype)
		return lexTags
	default:
		l.toFirstCh()
		if !isAlphaNumeric(l.peek()) {
			return l.errorf("lexDoc: expected Letter or Digit")
		}
		l.toWordEmit(itemDoctype)
		return lexAfterTag
	}
}

func (l *lexer) toFirstCh() {
	for {
		switch l.next() {
		case ' ', '\t':

		default:
			l.backup()
			l.ignore()
			return
		}
	}
}

func lexComment(l *lexer) stateFn {
	l.next()
	l.next()
	l.emit(itemComment)
	return lexAfterTag
}

func lexCommentSkip(l *lexer) stateFn {
	l.next()
	l.next()
	l.next()
	l.emit(itemBlank)
	return lexLongText
}

func lexIndents(l *lexer) stateFn {
	l.previous = l.parenDepth
	l.parenDepth = 0
Loop:
	for {
		switch l.next() {
		case ' ':
			l.emit(itemIdentSpace)
			l.parenDepth++
		case '\t':
			l.emit(itemIdentTab)
			l.parenDepth += TabSize
		case '\r', '\n':
			if l.parenDepth < l.previous {
				l.parenDepth = l.previous
			}
			l.backup()
			l.emit(itemEmptyLine)
			break Loop
		default:
			l.backup()
			break Loop
		}
	}
	if l.env[mInText] > 0 {
		l.env[mBrText] = stText
		return lexLongText
	}
	return lexTags
}

// lexTags scans tags.
func lexTags(l *lexer) stateFn {
	if strings.HasPrefix(l.input[l.pos:], l.leftDelim) {
		return lexAction
	}
	if strings.HasPrefix(l.input[l.pos:], tabComment) {
		return lexCommentSkip
	}
	if strings.HasPrefix(l.input[l.pos:], htmlComment) {
		return lexComment
	}
	if strings.HasPrefix(l.input[l.pos:], "doctype") {
		l.start += 7
		l.pos = l.start
		return lexDoc
	}
	if strings.HasPrefix(l.input[l.pos:], "!!!") {
		l.start += 3
		l.pos = l.start
		return lexDoc
	}

	switch r := l.next(); {
	case r == eof:
		l.emit(itemEOF)
		return nil
	case r == '\r':
		return lexTags
	case r == '\n':
		l.emit(itemEndL)
		return lexIndents
	case r == ' ' || r == '\t':
		l.backup()
		return lexIndents

	case r == '.':
		l.emit(itemDiv)
		return lexClass
	case r == '#':
		l.emit(itemDiv)
		return lexID
	case r == '|':
		l.ignore()
		l.env[mBrText] = stText
		return lexText
	case r == ':':
		l.ignore()
		return lexFilter
	case r == '<':
		return lexHTMLTag
	case r == '+':
		l.ignore()
		l.toFirstCh()
		l.toWordEmit(itemTemplate)
		l.toEndL(itemEmptyLine)
		return lexIndents
	case r == '-':
		// l.toFirstCh()
		sp := l.peek()
		if sp == '\r' || sp == '\n' {
			l.emit(itemComment)
			return lexLongText
		}
		l.ignore()
		return lexActionEndL
	case r == '=' || r == '$':
		l.ignore()
		return lexActionEndL
	case r == '!':
		if l.next() == '=' {
			l.ignore()
			return lexActionEndL
		}
		return l.errorf("expect '=' after '!'")
	case isAlphaNumeric(r):
		l.backup()
		return lexTagName

	default:
		return l.errorf("lexTags: %#U", r)
	}
}

func lexAfterTag(l *lexer) stateFn {
	switch r := l.next(); {
	case r == eof:
		l.emit(itemEOF)
		return nil
	case r == '(':
		l.ignore()
		return lexAttr
	case r == '/':
		l.emit(itemEndTag)
		return lexAfterTag
	case r == ':':
		l.ignore()
		for isSpace(l.peek()) {
			l.next()
		}
		l.emit(itemParentIdent)
		return lexTags
	case r == ' ' || r == '\t':
		l.ignore()
		l.env[mBrText] = stInlineText
		return lexText
	case r == ']':
		if l.env[mInterpolation] > 0 {
			l.ignore()
			l.emit(itemInlineText)
			l.emit(itemChildIdent)
			l.env[mInterpolation]--
			l.env[mBrText] = stInlineText
			return lexLongText
		}
		return l.errorf("lexAfterTag: %#U", r)
	case r == '=':
		l.ignore()
		return lexInlineAction
	case r == '!':
		if l.next() == '=' {
			l.ignore()
			return lexInlineAction
		}
		return l.errorf("expect '=' after '!'")
	case r == '#':
		l.ignore()
		return lexID
	case r == '.':
		sp := l.peek()
		l.ignore()
		if sp == ' ' {
			l.next()
			l.ignore()
			return lexLongText
		}
		if sp == '\r' || sp == '\n' {
			return lexLongText
		}
		return lexClass
	case r == '\r':
		l.next()
		l.emit(itemEndL)
		return lexIndents
	case r == '\n':
		l.emit(itemEndL)
		return lexIndents
	default:
		return l.errorf("lexAfterTag: %#U", r)
	}
}

func lexID(l *lexer) stateFn {
	if !isAlphaNumeric(l.peek()) {
		return l.errorf("lexID: expect id name")
	}
	l.toWordEmit(itemID)
	return lexAfterTag
}

func lexClass(l *lexer) stateFn {
	if !isAlphaNumeric(l.peek()) {
		return l.errorf("lexClass: expect class name")
	}
	l.toWordEmit(itemClass)
	return lexAfterTag
}

func lexFilter(l *lexer) stateFn {
	if !isAlphaNumeric(l.peek()) {
		return l.errorf("lexFilter: expect filter name")
	}
	l.toWordEmit(itemFilter)
	return lexLongText
}

func lexTagName(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			// absorb.
		default:
			l.backup()
			word := l.input[l.start:l.pos]
			switch key[word] {
			case itemExtends:
				if l.env[mInterpolation] > 0 {
					l.errorf("lexTagName: Tag Interpolation error (no itemExtends)")
				}
				l.env[mExtends] = 1
				l.toEndL(itemExtends)
				return lexAfterTag
			case itemBlock:
				if l.env[mInterpolation] > 0 {
					l.errorf("lexTagName: Tag Interpolation error (no itemBlock)")
				}
				l.toFirstCh()
				if l.env[mExtends] == 1 {
					l.toWordEmit(itemDefine)
				} else {
					l.toWordEmit(itemBlock)
				}
				return lexAfterTag
			case itemDefine:
				if l.env[mInterpolation] > 0 {
					l.errorf("lexTagName: Tag Interpolation error (no itemDefine)")
				}
				l.toFirstCh()
				l.toWordEmit(itemDefine)
				return lexAfterTag
			case itemAction:
				if l.env[mInterpolation] > 0 {
					l.errorf("lexTagName: Tag Interpolation error (no itemAction)")
				}
				if l.toEndL(itemAction) {
					return lexIndents
				}
				return nil
			case itemActionEnd:
				if l.env[mInterpolation] > 0 {
					l.errorf("lexTagName: Tag Interpolation error (no itemActionEnd)")
				}
				if l.toEndL(itemActionEnd) {
					return lexIndents
				}
				return nil
			case itemVoidTag,
				itemInlineVoidTag,
				itemInlineTag:
				l.emit(key[word])
			default:
				l.emit(itemTag)
			}
			return lexAfterTag
		}
	}
}

func lexAttr(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			l.backup()
			l.ignore()
			return lexAttrName
		case r == ')':
			l.ignore()
			return lexAfterTag
		case r == ' ' || r == ',' || r == '\t':
		case r == eof:
			return l.errorf("lexAttr: expected ')'")
		}
	}
}
func lexAttrName(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			// absorb.
		case r == '=':
			word := l.input[l.start:l.pos]
			switch {
			case word == "id=":
				l.ignore()
				return lexAttrID
			case word == "class=":
				l.ignore()
				return lexAttrClass
			default:
				l.backup()
				l.emit(itemAttrName)
				l.next()
				l.ignore()
				return lexAttrVal
			}
		case r == ' ' || r == ',' || r == ')' || r == '\r' || r == '\n':
			l.backup()
			l.emit(itemAttrVoid)
			return lexAttr
		default:
			return l.errorf("lexAttrName: expected '=' or ' ' %#U", r)
		}
	}
}
func lexAttrID(l *lexer) stateFn {
	stopCh := l.next()
	if stopCh == '"' || stopCh == '\'' {
		l.ignore()
		l.toStopCh(stopCh, itemID, true)
	} else {
		l.toStopSpace(itemID)
	}
	return lexAttr
}
func lexAttrClass(l *lexer) stateFn {
	stopCh := l.next()
	if stopCh == '"' || stopCh == '\'' {
		l.ignore()
		l.toStopCh(stopCh, itemClass, true)
	} else {
		l.toStopSpace(itemClass)
	}
	return lexAttr
}
func lexAttrVal(l *lexer) stateFn {
	stopCh := l.next()
	if stopCh == '"' || stopCh == '\'' {
		l.toStopCh(stopCh, itemAttr, false)
	} else {
		l.toStopSpace(itemAttrN)
	}
	return lexAttr
}
func (l *lexer) toStopCh(stopCh rune, item itemType, backup bool) {
	for {
		switch r := l.next(); {
		case r == stopCh:
			if backup {
				l.backup()
			}
			l.emit(item)
			return
		case r == eof || r == '\r' || r == '\n':
			l.errorf("toStopCh: expected '%#U' %#U", stopCh, r)
			return
		}
	}
}
func (l *lexer) toStopSpace(item itemType) {
	var bracket int
	for {
		switch r := l.next(); {
		case r == '(':
			bracket++
		case r == ')':
			if bracket == 0 {
				l.backup()
				l.emit(item)
				return
			}
			if bracket > 0 {
				bracket--
			}
		case r == ' ' || r == ',' || r == '\r' || r == '\n':
			l.backup()
			l.emit(item)
			return
		case r == eof:
			l.errorf("toStopCh: expected ')' %#U", r)
			return
		}
	}
}
func (l *lexer) toWordEmit(item itemType) {
	for {
		if !isAlphaNumeric(l.next()) {
			l.backup()
			break
		}
	}
	l.emit(item)
}

func lexLongText(l *lexer) stateFn {
	if l.env[mInText] == 0 {
		if l.parenDepth == 0 { // for tags indent = 0
			l.env[mInText] = 1
			return lexText
		}
		l.env[mInText] = l.parenDepth
		l.env[mBrText] = stInlineText
		return lexText
	}

	if l.env[mInText] < l.parenDepth {
		return lexText
	}

	l.env[mInText] = 0
	return lexIndents
}
func lexText(l *lexer) stateFn {
	var item itemType
	switch l.env[mBrText] {
	case stText:
		item = itemText
	case stInlineText:
		item = itemInlineText
	default:
		l.errorf("lexText: expected 'l.env[mBrText]'")
	}
	for {
		switch r := l.next(); {
		case r == '#':
			sp := l.peek()
			if sp == '[' {
				l.env[mInterpolation]++
				l.backup()
				l.emit(item)
				l.next()
				l.next()
				l.emit(itemParentIdent)
				return lexTags
			}
		case r == ']':
			if l.env[mInterpolation] > 0 {
				l.backup()
				if l.pos > l.start {
					l.emit(item)
				}
				l.next()
				l.emit(itemChildIdent)
				l.env[mInterpolation]--
			}
		case r == '\n', r == '\r':
			l.backup()
			if l.pos > l.start {
				l.emit(item)
			}
			l.next()
			if r == '\r' {
				l.next()
			}
			l.emit(itemEndL)
			if l.env[mInterpolation] > 0 {
				l.errorf("toEndText: expected ']' (no closing bracket found)")
			}
			return lexIndents
		case r == eof:
			if l.pos > l.start {
				l.emit(item)
			}
			l.emit(itemEOF)
			return nil
		}
	}
}

func lexHTMLTag(l *lexer) stateFn {
	if l.toEndL(itemHTMLTag) {
		return lexIndents
	}
	return nil
}

func lexActionEndL(l *lexer) stateFn {
	if l.toEndL(itemAction) {
		return lexIndents
	}
	return nil
}

func lexInlineAction(l *lexer) stateFn {
	if l.toEndL(itemInlineAction) {
		return lexIndents
	}
	return nil
}

func (l *lexer) toEndL(item itemType) bool {
Loop:
	for {
		switch r := l.next(); {
		case r == eof:
			if l.pos > l.start {
				l.emit(item)
			}
			l.emit(itemEOF)
			return false
		case r == '\r':
			l.backup()
			if l.pos > l.start {
				l.emit(item)
			}
			l.next()
			break Loop
		case r == '\n':
			l.backup()
			if l.pos > l.start {
				l.emit(item)
			}
			break Loop
		}
	}
	l.next()
	l.emit(itemEndL)
	return true
}

func lexAction(l *lexer) stateFn {
	l.next()
	l.next()
	l.ignore()
	for {
		l.next()
		if strings.HasPrefix(l.input[l.pos:], l.rightDelim) {
			break
		}
	}
	l.emit(itemAction)
	l.next()
	l.next()
	l.ignore()
	return lexAfterTag
}
