package view

import (
	"bytes"
	"io/ioutil"
	"path"
	"strings"

	"github.com/iris-contrib/jade"
)

// Pug (or Jade) returns a new pug view engine.
// It shares the same exactly logic with the
// html view engine, it uses the same exactly configuration.
// It has got some features and a lot of functions
// which will make your life easier.
// Read more about the Jade Go Parser: https://github.com/Joker/jade
//
// Examples:
// https://github.com/kataras/iris/tree/master/_examples/view/template_pug_0
// https://github.com/kataras/iris/tree/master/_examples/view/template_pug_1
// https://github.com/kataras/iris/tree/master/_examples/view/template_pug_2
// https://github.com/kataras/iris/tree/master/_examples/view/template_pug_3
func Pug(directory, extension string) *HTMLEngine {
	s := HTML(directory, extension)

	s.middleware = func(name string, text []byte) (contents string, err error) {
		name = path.Join(path.Clean(directory), name)
		tmpl := jade.New(name)
		tmpl.ReadFunc = func(name string) ([]byte, error) {
			if !strings.HasPrefix(path.Clean(name), path.Clean(directory)) {
				name = path.Join(directory, name)
			}

			if s.assetFn != nil {
				return s.assetFn(name)
			}
			return ioutil.ReadFile(name)
		}

		// Fixes: https://github.com/kataras/iris/issues/1450
		// by adding a custom ReadFunc inside the jade parser.
		// And Also able to use relative paths on "extends" and "include" directives:
		// e.g. instead of extends "templates/layout.pug" we use "layout.pug"
		// so contents of templates are independent of their root location.

		exec, err := tmpl.Parse(text)
		if err != nil {
			return
		}

		b := new(bytes.Buffer)
		exec.WriteIn(b)
		return b.String(), nil
	}
	return s
}
