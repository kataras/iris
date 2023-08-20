package view

import (
	"bytes"
	"os"

	"github.com/Joker/jade"
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
// Pug(embed.FS, ".pug") or Pug(AssetFile(), ".pug") for embedded data.
//
// Examples:
// https://github.com/kataras/iris/tree/main/_examples/view/template_pug_0
// https://github.com/kataras/iris/tree/main/_examples/view/template_pug_1
// https://github.com/kataras/iris/tree/main/_examples/view/template_pug_2
// https://github.com/kataras/iris/tree/main/_examples/view/template_pug_3
func Pug(fs interface{}, extension string) *HTMLEngine {
	s := HTML(fs, extension)
	s.name = "Pug"
	s.middleware = func(name string, text []byte) (contents string, err error) {
		jade.ReadFunc = func(filename string) ([]byte, error) {
			return asset(s.fs, filename)
		}

		tmpl := jade.New(name)
		exec, err := tmpl.Parse(text)
		if err != nil {
			return
		}

		b := new(bytes.Buffer)
		exec.WriteIn(b)
		jade.ReadFunc = os.ReadFile // reset to original.
		return b.String(), nil
	}
	return s
}
