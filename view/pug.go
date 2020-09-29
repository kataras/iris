package view

import (
	"bytes"

	"github.com/iris-contrib/jade"
)

// Pug (or Jade) returns a new pug view engine.
// It shares the same exactly logic with the
// html view engine, it uses the same exactly configuration.
// The given "extension" MUST begin with a dot.
//
// Read more about the Jade Go Parser: https://github.com/Joker/jade
//
// Usage:
// Pug("./views", ".pug") or
// Pug(iris.Dir("./views"), ".pug") or
// Pug(AssetFile(), ".pug") for embedded data.
//
// Examples:
// https://github.com/kataras/iris/tree/master/_examples/view/template_pug_0
// https://github.com/kataras/iris/tree/master/_examples/view/template_pug_1
// https://github.com/kataras/iris/tree/master/_examples/view/template_pug_2
// https://github.com/kataras/iris/tree/master/_examples/view/template_pug_3
func Pug(fs interface{}, extension string) *HTMLEngine {
	s := HTML(fs, extension)
	s.name = "Pug"

	s.middleware = func(name string, text []byte) (contents string, err error) {
		tmpl := jade.New(name)
		tmpl.ReadFunc = func(name string) ([]byte, error) {
			return asset(s.fs, name)
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
