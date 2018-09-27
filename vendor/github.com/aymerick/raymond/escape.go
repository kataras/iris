package raymond

import (
	"bytes"
	"strings"
)

//
// That whole file is borrowed from https://github.com/golang/go/tree/master/src/html/escape.go
//
// With changes:
//    &#39 => &apos;
//    &#34 => &quot;
//
// To stay in sync with JS implementation, and make mustache tests pass.
//

type writer interface {
	WriteString(string) (int, error)
}

const escapedChars = `&'<>"`

func escape(w writer, s string) error {
	i := strings.IndexAny(s, escapedChars)
	for i != -1 {
		if _, err := w.WriteString(s[:i]); err != nil {
			return err
		}
		var esc string
		switch s[i] {
		case '&':
			esc = "&amp;"
		case '\'':
			esc = "&apos;"
		case '<':
			esc = "&lt;"
		case '>':
			esc = "&gt;"
		case '"':
			esc = "&quot;"
		default:
			panic("unrecognized escape character")
		}
		s = s[i+1:]
		if _, err := w.WriteString(esc); err != nil {
			return err
		}
		i = strings.IndexAny(s, escapedChars)
	}
	_, err := w.WriteString(s)
	return err
}

// Escape escapes special HTML characters.
//
// It can be used by helpers that return a SafeString and that need to escape some content by themselves.
func Escape(s string) string {
	if strings.IndexAny(s, escapedChars) == -1 {
		return s
	}
	var buf bytes.Buffer
	escape(&buf, s)
	return buf.String()
}
