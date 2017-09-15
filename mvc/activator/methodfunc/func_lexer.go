package methodfunc

import (
	"unicode"
)

const (
	tokenBy       = "By"
	tokenWildcard = "Wildcard" // should be followed by "By",
)

// word lexer, not characters.
type lexer struct {
	words []string
	cur   int
}

func newLexer(s string) *lexer {
	l := new(lexer)
	l.reset(s)
	return l
}

func (l *lexer) reset(trailing string) {
	l.cur = -1
	var words []string
	if trailing != "" {
		end := len(trailing)
		start := -1

		for i, n := 0, end; i < n; i++ {
			c := rune(trailing[i])
			if unicode.IsUpper(c) {
				// it doesn't count the last uppercase
				if start != -1 {
					end = i
					words = append(words, trailing[start:end])
				}
				start = i
				continue
			}
			end = i + 1
		}

		if end > 0 && len(trailing) >= end {
			words = append(words, trailing[start:end])
		}
	}

	l.words = words
}

func (l *lexer) next() (w string) {
	cur := l.cur + 1

	if w = l.peek(cur); w != "" {
		l.cur++
	}

	return
}

func (l *lexer) skip() {
	if cur := l.cur + 1; cur < len(l.words) {
		l.cur = cur
	} else {
		l.cur = len(l.words) - 1
	}
}

func (l *lexer) peek(idx int) string {
	if idx < len(l.words) {
		return l.words[idx]
	}
	return ""
}

func (l *lexer) peekNext() (w string) {
	return l.peek(l.cur + 1)
}

func (l *lexer) peekPrev() (w string) {
	if l.cur > 0 {
		cur := l.cur - 1
		w = l.words[cur]
	}

	return w
}
