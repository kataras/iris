package view

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/eknkc/amber"
)

// AmberEngine contains the amber view engine structure.
type AmberEngine struct {
	fs http.FileSystem
	// files configuration
	rootDir   string
	extension string
	reload    bool
	//
	rmu           sync.RWMutex // locks for `ExecuteWiter` when `reload` is true.
	templateCache map[string]*template.Template

	Options amber.Options
}

var (
	_ Engine       = (*AmberEngine)(nil)
	_ EngineFuncer = (*AmberEngine)(nil)
)

// Amber creates and returns a new amber view engine.
// The given "extension" MUST begin with a dot.
//
// Usage:
// Amber("./views", ".amber") or
// Amber(iris.Dir("./views"), ".amber") or
// Amber(AssetFile(), ".amber") for embedded data.
func Amber(fs interface{}, extension string) *AmberEngine {
	fileSystem := getFS(fs)
	s := &AmberEngine{
		fs:            fileSystem,
		rootDir:       "/",
		extension:     extension,
		templateCache: make(map[string]*template.Template),
		Options: amber.Options{
			PrettyPrint:       false,
			LineNumbers:       false,
			VirtualFilesystem: fileSystem,
		},
	}

	return s
}

// RootDir sets the directory to be used as a starting point
// to load templates from the provided file system.
func (s *AmberEngine) RootDir(root string) *AmberEngine {
	s.rootDir = filepath.ToSlash(root)
	return s
}

// Ext returns the file extension which this view engine is responsible to render.
func (s *AmberEngine) Ext() string {
	return s.extension
}

// Reload if set to true the templates are reloading on each render,
// use it when you're in development and you're boring of restarting
// the whole app when you edit a template file.
//
// Note that if `true` is passed then only one `View -> ExecuteWriter` will be render each time,
// no concurrent access across clients, use it only on development status.
// It's good to be used side by side with the https://github.com/kataras/rizla reloader for go source files.
func (s *AmberEngine) Reload(developmentMode bool) *AmberEngine {
	s.reload = developmentMode
	return s
}

// AddFunc adds the function to the template's function map.
// It is legal to overwrite elements of the default actions:
// - url func(routeName string, args ...string) string
// - urlpath func(routeName string, args ...string) string
// - render func(fullPartialName string) (template.HTML, error).
//
// Note that, Amber does not support functions per template,
// instead it's using the "call" directive so any template-specific
// functions should be passed using `Context.View/ViewData` binding data.
// This method will modify the global amber's FuncMap which considers
// as the "builtin" as this is the only way to actually add a function.
// Note that, if you use more than one amber engine, the functions are shared.
func (s *AmberEngine) AddFunc(funcName string, funcBody interface{}) {
	s.rmu.Lock()
	amber.FuncMap[funcName] = funcBody
	s.rmu.Unlock()
}

// Load parses the templates to the engine.
// It is responsible to add the necessary global functions.
//
// Returns an error if something bad happens, user is responsible to catch it.
func (s *AmberEngine) Load() error {
	return walk(s.fs, s.rootDir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		if s.extension != "" {
			if !strings.HasSuffix(path, s.extension) {
				return nil
			}
		}

		contents, err := asset(s.fs, path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}

		err = s.ParseTemplate(path, contents)
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}
		return nil
	})
}

// ParseTemplate adds a custom template from text.
// This template parser does not support funcs per template directly.
// Two ways to add a function:
// Globally: Use `AddFunc` or pass them on `View` instead.
// Per Template: Use `Context.ViewData/View`.
func (s *AmberEngine) ParseTemplate(name string, contents []byte) error {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	comp := amber.New()
	comp.Options = s.Options

	err := comp.ParseData(contents, name)
	if err != nil {
		return err
	}

	data, err := comp.CompileString()
	if err != nil {
		return err
	}

	name = strings.TrimPrefix(name, "/")

	/* Sadly, this does not work, only builtin amber.FuncMap
	can be executed as function, the rest are compiled as data (prepends a "call"),
	relative code:
	https://github.com/eknkc/amber/blob/cdade1c073850f4ffc70a829e31235ea6892853b/compiler.go#L771-L794

	tmpl := template.New(name).Funcs(amber.FuncMap).Funcs(s.funcs)
	if len(funcs) > 0 {
		tmpl.Funcs(funcs)
	}

	We can't add them as binding data of map type
	because those data can be a struct by the caller and we don't want to messup.
	*/

	tmpl := template.New(name).Funcs(amber.FuncMap)
	_, err = tmpl.Parse(data)
	if err == nil {
		s.templateCache[name] = tmpl
	}

	return err
}

func (s *AmberEngine) fromCache(relativeName string) *template.Template {
	if s.reload {
		s.rmu.RLock()
		defer s.rmu.RUnlock()
	}

	if tmpl, ok := s.templateCache[relativeName]; ok {
		return tmpl
	}

	return nil
}

// ExecuteWriter executes a template and writes its result to the w writer.
// layout here is useless.
func (s *AmberEngine) ExecuteWriter(w io.Writer, filename string, layout string, bindingData interface{}) error {
	// re-parse the templates if reload is enabled.
	if s.reload {
		if err := s.Load(); err != nil {
			return err
		}
	}

	if tmpl := s.fromCache(filename); tmpl != nil {
		return tmpl.Execute(w, bindingData)
	}

	return ErrNotExist{filename, false}
}
