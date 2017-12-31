package mvc

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/kataras/iris/core/router"
	"github.com/kataras/iris/core/router/macro/interpreter/ast"
)

const (
	tokenBy       = "By"
	tokenWildcard = "Wildcard" // "ByWildcard".
)

// word lexer, not characters.
type methodLexer struct {
	words []string
	cur   int
}

func newMethodLexer(s string) *methodLexer {
	l := new(methodLexer)
	l.reset(s)
	return l
}

func (l *methodLexer) reset(s string) {
	l.cur = -1
	var words []string
	if s != "" {
		end := len(s)
		start := -1

		for i, n := 0, end; i < n; i++ {
			c := rune(s[i])
			if unicode.IsUpper(c) {
				// it doesn't count the last uppercase
				if start != -1 {
					end = i
					words = append(words, s[start:end])
				}
				start = i
				continue
			}
			end = i + 1
		}

		if end > 0 && len(s) >= end {
			words = append(words, s[start:end])
		}
	}

	l.words = words
}

func (l *methodLexer) next() (w string) {
	cur := l.cur + 1

	if w = l.peek(cur); w != "" {
		l.cur++
	}

	return
}

func (l *methodLexer) skip() {
	if cur := l.cur + 1; cur < len(l.words) {
		l.cur = cur
	} else {
		l.cur = len(l.words) - 1
	}
}

func (l *methodLexer) peek(idx int) string {
	if idx < len(l.words) {
		return l.words[idx]
	}
	return ""
}

func (l *methodLexer) peekNext() (w string) {
	return l.peek(l.cur + 1)
}

func (l *methodLexer) peekPrev() (w string) {
	if l.cur > 0 {
		cur := l.cur - 1
		w = l.words[cur]
	}

	return w
}

var posWords = map[int]string{
	0:  "",
	1:  "first",
	2:  "second",
	3:  "third",
	4:  "forth",
	5:  "five",
	6:  "sixth",
	7:  "seventh",
	8:  "eighth",
	9:  "ninth",
	10: "tenth",
	11: "eleventh",
	12: "twelfth",
	13: "thirteenth",
	14: "fourteenth",
	15: "fifteenth",
	16: "sixteenth",
	17: "seventeenth",
	18: "eighteenth",
	19: "nineteenth",
	20: "twentieth",
}

func genParamKey(argIdx int) string {
	return "arg" + posWords[argIdx] // argfirst, argsecond...
}

type methodParser struct {
	lexer *methodLexer
	fn    reflect.Method
}

func parseMethod(fn reflect.Method, skipper func(string) bool) (method, path string, err error) {
	if skipper(fn.Name) {
		return "", "", errSkip
	}

	p := &methodParser{
		fn:    fn,
		lexer: newMethodLexer(fn.Name),
	}
	return p.parse()
}

func methodTitle(httpMethod string) string {
	httpMethodFuncName := strings.Title(strings.ToLower(httpMethod))
	return httpMethodFuncName
}

var errSkip = errors.New("skip")

var allMethods = append(router.AllMethods[0:], []string{"ALL", "ANY"}...)

func (p *methodParser) parse() (method, path string, err error) {
	funcArgPos := 0
	path = "/"
	// take the first word and check for the method.
	w := p.lexer.next()

	for _, httpMethod := range allMethods {
		possibleMethodFuncName := methodTitle(httpMethod)
		if strings.Index(w, possibleMethodFuncName) == 0 {
			method = httpMethod
			break
		}
	}

	if method == "" {
		// this is not a valid method to parse, we just skip it,
		//  it may be used for end-dev's use cases.
		return "", "", errSkip
	}

	for {
		w := p.lexer.next()
		if w == "" {
			break
		}

		if w == tokenBy {
			funcArgPos++ // starting with 1 because in typ.NumIn() the first is the struct receiver.

			// No need for these:
			// ByBy will act like /{param:type}/{param:type} as users expected
			// if func input arguments are there, else act By like normal path /by.
			//
			// if p.lexer.peekPrev() == tokenBy || typ.NumIn() == 1 { // ByBy, then act this second By like a path
			// 	a.relPath += "/" + strings.ToLower(w)
			// 	continue
			// }

			if path, funcArgPos, err = p.parsePathParam(path, w, funcArgPos); err != nil {
				return "", "", err
			}

			continue
		}
		// static path.
		path += "/" + strings.ToLower(w)
	}
	return
}

func (p *methodParser) parsePathParam(path string, w string, funcArgPos int) (string, int, error) {
	typ := p.fn.Type

	if typ.NumIn() <= funcArgPos {

		// By found but input arguments are not there, so act like /by path without restricts.
		path += "/" + strings.ToLower(w)
		return path, funcArgPos, nil
	}

	var (
		paramKey  = genParamKey(funcArgPos) // argfirst, argsecond...
		paramType = ast.ParamTypeString     // default string
	)

	// string, int...
	goType := typ.In(funcArgPos).Name()
	nextWord := p.lexer.peekNext()

	if nextWord == tokenWildcard {
		p.lexer.skip() // skip the Wildcard word.
		paramType = ast.ParamTypePath
	} else if pType := ast.LookupParamTypeFromStd(goType); pType != ast.ParamTypeUnExpected {
		// it's not wildcard, so check base on our available macro types.
		paramType = pType
	} else {
		if typ.NumIn() > funcArgPos {
			// has more input arguments but we are not in the correct
			// index now, maybe the first argument was an `iris/context.Context`
			// so retry with the "funcArgPos" incremented.
			//
			// the "funcArgPos" will be updated to the caller as well
			// because we return it among the path and the error.
			return p.parsePathParam(path, w, funcArgPos+1)
		}
		return "", 0, errors.New("invalid syntax for " + p.fn.Name)
	}

	// /{argfirst:path}, /{argfirst:long}...
	path += fmt.Sprintf("/{%s:%s}", paramKey, paramType.String())

	if nextWord == "" && typ.NumIn() > funcArgPos+1 {
		// By is the latest word but func is expected
		// more path parameters values, i.e:
		// GetBy(name string, age int)
		// The caller (parse) doesn't need to know
		// about the incremental funcArgPos because
		// it will not need it.
		return p.parsePathParam(path, nextWord, funcArgPos+1)
	}

	return path, funcArgPos, nil
}
