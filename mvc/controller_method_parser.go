package mvc

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/macro"
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

/*
var allowedCapitalWords = map[string]struct{}{
	"ID":   {},
	"JSON": {},
}
*/

func (l *methodLexer) reset(s string) {
	l.cur = -1
	var words []string
	if s != "" {
		end := len(s)
		start := -1

		// outter:
		for i, n := 0, end; i < n; i++ {
			c := rune(s[i])
			if unicode.IsUpper(c) {
				// it doesn't count the last uppercase
				if start != -1 {
					/*
						for allowedCapitalWord := range allowedCapitalWords {
							capitalWordEnd := i + len(allowedCapitalWord) // takes last char too, e.g. ReadJSON, we need the JSON.
							if len(s) >= capitalWordEnd {
								word := s[i:capitalWordEnd]
								if word == allowedCapitalWord {
									words = append(words, word)
									i = capitalWordEnd
									start = i
									continue outter
								}
							}
						}
					*/

					end = i
					words = append(words, s[start:end])
				}

				start = i
				continue
			}
			end = i + 1
		}

		if end > 0 && len(s) >= end {
			words = append(words, s[start:])
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

func genParamKey(argIdx int) string {
	return "param" + strconv.Itoa(argIdx) // param0, param1, param2...
}

type methodParser struct {
	lexer              *methodLexer
	fn                 reflect.Method
	macros             *macro.Macros
	customPathWordFunc CustomPathWordFunc
}

func parseMethod(macros *macro.Macros, fn reflect.Method, skipper func(string) bool, wordFunc CustomPathWordFunc) (method, path string, err error) {
	if skipper(fn.Name) {
		return "", "", errSkip
	}

	p := &methodParser{
		fn:                 fn,
		lexer:              newMethodLexer(fn.Name),
		macros:             macros,
		customPathWordFunc: wordFunc,
	}
	return p.parse()
}

func methodTitle(httpMethod string) string {
	httpMethodFuncName := strings.Title(strings.ToLower(httpMethod))
	return httpMethodFuncName
}

var errSkip = errors.New("skip")

var allMethods = append(router.AllMethods[0:], []string{"ALL", "ANY"}...)

// CustomPathWordFunc describes the function which can be passed
// through `Application.SetCustomPathWordFunc` to customize
// the controllers method parsing.
type CustomPathWordFunc func(path, w string, wordIndex int) string

func addPathWord(path, w string) string {
	if path[len(path)-1] != '/' {
		path += "/"
	}
	path += strings.ToLower(w)
	return path
}

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

	wordIndex := 0
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

		// custom static path.
		if p.customPathWordFunc != nil {
			path = p.customPathWordFunc(path, w, wordIndex)
		} else {
			// default static path.
			path = addPathWord(path, w)
		}
		wordIndex++
	}
	return
}

func (p *methodParser) parsePathParam(path string, w string, funcArgPos int) (string, int, error) {
	typ := p.fn.Type

	if typ.NumIn() <= funcArgPos {

		// By found but input arguments are not there, so act like /by path without restricts.
		path = addPathWord(path, w)
		return path, funcArgPos, nil
	}

	var (
		paramKey  = genParamKey(funcArgPos) // argfirst, argsecond...
		m         = p.macros.GetMaster()    // default (String by-default)
		trailings = p.macros.GetTrailings()
	)

	// string, int...
	goType := typ.In(funcArgPos).Kind()
	nextWord := p.lexer.peekNext()

	if nextWord == tokenWildcard {
		p.lexer.skip() // skip the Wildcard word.
		if len(trailings) == 0 {
			return "", 0, errors.New("no trailing path parameter found")
		}
		m = trailings[0]
	} else {
		// validMacros := p.macros.LookupForGoType(goType)

		// instead of mapping with a reflect.Kind which has its limitation,
		// we map the param types with a go type as a string,
		// so custom structs such as "user" can be mapped to a macro with indent || alias == "user".
		m = p.macros.Get(strings.ToLower(goType.String()))

		if m == nil {
			if typ.NumIn() > funcArgPos {
				// has more input arguments but we are not in the correct
				// index now, maybe the first argument was an `iris/context.Context`
				// so retry with the "funcArgPos" incremented.
				//
				// the "funcArgPos" will be updated to the caller as well
				// because we return it among the path and the error.
				return p.parsePathParam(path, w, funcArgPos+1)
			}

			return "", 0, fmt.Errorf("invalid syntax: the standard go type: %s found in controller's function: %s at position: %d does not match any valid macro", goType, p.fn.Name, funcArgPos)
		}
	}

	// /{argfirst:path}, /{argfirst:int64}...
	if path[len(path)-1] != '/' {
		path += "/"
	}
	path += fmt.Sprintf("{%s:%s}", paramKey, m.Indent())

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
