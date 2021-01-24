package view

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

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
	bufPool       *sync.Pool

	Options amber.Options
}

var (
	_ Engine       = (*AmberEngine)(nil)
	_ EngineFuncer = (*AmberEngine)(nil)
)

var amberOnce = new(uint32)

// Amber creates and returns a new amber view engine.
// The given "extension" MUST begin with a dot.
//
// Usage:
// Amber("./views", ".amber") or
// Amber(iris.Dir("./views"), ".amber") or
// Amber(AssetFile(), ".amber") for embedded data.
func Amber(fs interface{}, extension string) *AmberEngine {
	if atomic.LoadUint32(amberOnce) > 0 {
		panic("Amber: cannot be registered twice as its internal implementation share the same template functions across instances.")
	} else {
		atomic.StoreUint32(amberOnce, 1)
	}

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
		bufPool: &sync.Pool{New: func() interface{} {
			return new(bytes.Buffer)
		}},
	}

	builtinFuncs := template.FuncMap{
		"render": func(name string, binding interface{}) (template.HTML, error) {
			result, err := s.executeTemplateBuf(name, binding)
			return template.HTML(result), err
		},
	}

	for k, v := range builtinFuncs {
		amber.FuncMap[k] = v
	}

	return s
}

// RootDir sets the directory to be used as a starting point
// to load templates from the provided file system.
func (s *AmberEngine) RootDir(root string) *AmberEngine {
	s.rootDir = filepath.ToSlash(root)
	return s
}

// Name returns the amber engine's name.
func (s *AmberEngine) Name() string {
	return "Amber"
}

// Ext returns the file extension which this view engine is responsible to render.
// If the filename extension on ExecuteWriter is empty then this is appended.
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

// SetPrettyPrint if pretty printing is enabled.
// Pretty printing ensures that the output html is properly indented and in human readable form.
// Defaults to false, response is minified.
func (s *AmberEngine) SetPrettyPrint(pretty bool) *AmberEngine {
	s.Options.PrettyPrint = true
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
		if err != nil {
			return err
		}

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

	/*
		New(...).Funcs(s.builtinFuncs):
		This won't work on amber, it loads only amber.FuncMap which is global.
		Relative code:
		https://github.com/eknkc/amber/blob/cdade1c073850f4ffc70a829e31235ea6892853b/compiler.go#L771-L794
	*/

	tmpl := template.New(name).Funcs(amber.FuncMap)
	_, err = tmpl.Parse(data)
	if err == nil {
		s.templateCache[name] = tmpl
	}

	return err
}

func (s *AmberEngine) executeTemplateBuf(name string, binding interface{}) (string, error) {
	buf := s.bufPool.Get().(*bytes.Buffer)
	buf.Reset()

	tmpl := s.fromCache(name)
	if tmpl == nil {
		s.bufPool.Put(buf)
		return "", ErrNotExist{name, false, binding}
	}

	err := tmpl.ExecuteTemplate(buf, name, binding)
	result := strings.TrimSuffix(buf.String(), "\n") // on amber it adds a new line.
	s.bufPool.Put(buf)
	return result, err
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

	return ErrNotExist{filename, false, bindingData}
}
