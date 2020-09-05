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
	funcs         map[string]interface{}
	templateCache map[string]*template.Template
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
	s := &AmberEngine{
		fs:            getFS(fs),
		rootDir:       "/",
		extension:     extension,
		templateCache: make(map[string]*template.Template),
		funcs:         make(map[string]interface{}),
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
func (s *AmberEngine) AddFunc(funcName string, funcBody interface{}) {
	s.rmu.Lock()
	s.funcs[funcName] = funcBody
	s.rmu.Unlock()
}

// Load parses the templates to the engine.
// It is responsible to add the necessary global functions.
//
// Returns an error if something bad happens, user is responsible to catch it.
func (s *AmberEngine) Load() error {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	// prepare the global amber funcs
	funcs := template.FuncMap{}

	for k, v := range amber.FuncMap { // add the amber's default funcs
		funcs[k] = v
	}

	for k, v := range s.funcs {
		funcs[k] = v
	}

	amber.FuncMap = funcs // set the funcs

	opts := amber.Options{
		PrettyPrint:       false,
		LineNumbers:       false,
		VirtualFilesystem: s.fs,
	}

	return walk(s.fs, s.rootDir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		if s.extension != "" {
			if !strings.HasSuffix(path, s.extension) {
				return nil
			}
		}

		buf, err := asset(s.fs, path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}

		name := strings.TrimPrefix(path, "/")

		tmpl, err := amber.CompileData(buf, name, opts)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}

		s.templateCache[name] = tmpl

		return nil
	})
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
