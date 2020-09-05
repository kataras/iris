package view

import (
	"sync"

	"github.com/yosssi/ace"
)

// Ace returns a new ace view engine.
// It shares the same exactly logic with the
// html view engine, it uses the same exactly configuration.
// The given "extension" MUST begin with a dot.
//
// Read more about the Ace Go Parser: https://github.com/yosssi/ace
//
// Usage:
// Ace("./views", ".ace") or
// Ace(iris.Dir("./views"), ".ace") or
// Ace(AssetFile(), ".ace") for embedded data.
func Ace(fs interface{}, extension string) *HTMLEngine {
	s := HTML(fs, extension)

	funcs := make(map[string]interface{}, 0)

	once := new(sync.Once)
	s.middleware = func(name string, text []byte) (contents string, err error) {
		once.Do(func() { // on first template parse, all funcs are given.
			for k, v := range emptyFuncs {
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

		rslt, err := ace.ParseSource(src, nil)
		if err != nil {
			return "", err
		}

		t, err := ace.CompileResult(name, rslt, &ace.Options{
			Extension:  extension[1:],
			FuncMap:    funcs,
			DelimLeft:  s.left,
			DelimRight: s.right,
		})
		if err != nil {
			return "", err
		}

		return t.Lookup(name).Tree.Root.String(), nil
	}
	return s
}
