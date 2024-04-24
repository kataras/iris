package view

import (
	"strings"
	"sync"

	"github.com/yosssi/ace"
)

// AceEngine represents the Ace view engine.
// See the `Ace` package-level function for more.
type AceEngine struct {
	*HTMLEngine

	indent string
}

// SetIndent string used for indentation.
// Do NOT use tabs, only spaces characters.
// Defaults to minified response, no indentation.
func (s *AceEngine) SetIndent(indent string) *AceEngine {
	s.indent = indent
	return s
}

// Ace returns a new Ace view engine.
// It shares the same exactly logic with the
// html view engine, it uses the same exactly configuration.
// The given "extension" MUST begin with a dot.
// Ace minifies the response automatically unless
// SetIndent() method is set.
//
// Read more about the Ace Go Parser: https://github.com/yosssi/ace
//
// Usage:
// Ace("./views", ".ace") or
// Ace(iris.Dir("./views"), ".ace") or
// Ace(embed.FS, ".ace") or Ace(AssetFile(), ".ace") for embedded data.
func Ace(fs interface{}, extension string) *AceEngine {
	s := &AceEngine{HTMLEngine: HTML(fs, extension), indent: ""}
	s.name = "Ace"

	funcs := make(map[string]interface{})

	once := new(sync.Once)

	s.middleware = func(name string, text []byte) (contents string, err error) {
		once.Do(func() { // on first template parse, all funcs are given.
			for k, v := range s.getBuiltinFuncs(name) {
				funcs[k] = v
			}

			for k, v := range s.funcs {
				funcs[k] = v
			}
		})

		//	name = path.Join(path.Clean(directory), name)

		src := ace.NewSource(
			ace.NewFile(name, text),
			ace.NewFile("", []byte{}),
			[]*ace.File{},
		)

		if strings.Contains(name, "layout") {
			for k, v := range s.layoutFuncs {
				funcs[k] = v
			}
		}

		opts := &ace.Options{
			Extension:  extension[1:],
			FuncMap:    funcs,
			DelimLeft:  s.left,
			DelimRight: s.right,
			Indent:     s.indent,
		}

		rslt, err := ace.ParseSource(src, opts)
		if err != nil {
			return "", err
		}

		t, err := ace.CompileResult(name, rslt, opts)
		if err != nil {
			return "", err
		}

		return t.Lookup(name).Tree.Root.String(), nil
	}

	return s
}
