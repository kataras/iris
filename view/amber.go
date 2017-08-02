package view

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/eknkc/amber"
)

// AmberEngine contains the amber view engine structure.
type AmberEngine struct {
	// files configuration
	directory string
	extension string
	assetFn   func(name string) ([]byte, error) // for embedded, in combination with directory & extension
	namesFn   func() []string                   // for embedded, in combination with directory & extension
	reload    bool
	//
	rmu           sync.RWMutex // locks for funcs
	funcs         map[string]interface{}
	mu            sync.Mutex // locks for template files load
	templateCache map[string]*template.Template
}

var _ Engine = &AmberEngine{}

// Amber creates and returns a new amber view engine.
func Amber(directory, extension string) *AmberEngine {
	s := &AmberEngine{
		directory:     directory,
		extension:     extension,
		templateCache: make(map[string]*template.Template, 0),
		funcs:         make(map[string]interface{}, 0),
	}

	return s
}

// Ext returns the file extension which this view engine is responsible to render.
func (s *AmberEngine) Ext() string {
	return s.extension
}

// Binary optionally, use it when template files are distributed
// inside the app executable (.go generated files).
//
// The assetFn and namesFn can come from the go-bindata library.
func (s *AmberEngine) Binary(assetFn func(name string) ([]byte, error), namesFn func() []string) *AmberEngine {
	s.assetFn, s.namesFn = assetFn, namesFn
	return s
}

// Reload if setted to true the templates are reloading on each render,
// use it when you're in development and you're boring of restarting
// the whole app when you edit a template file.
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
// It's alos responsible to add the necessary global functions.
//
// Returns an error if something bad happens, user is responsible to catch it.
func (s *AmberEngine) Load() error {
	if s.assetFn != nil && s.namesFn != nil {
		// embedded
		return s.loadAssets()
	}

	// load from directory, make the dir absolute here too.
	dir, err := filepath.Abs(s.directory)
	if err != nil {
		return err
	}
	// change the directory field configuration, load happens after directory has been setted, so we will not have any problems here.
	s.directory = dir
	return s.loadDirectory()
}

// loadDirectory loads the amber templates from directory.
func (s *AmberEngine) loadDirectory() error {
	dir, extension := s.directory, s.extension

	opt := amber.DirOptions{}
	opt.Recursive = true

	// prepare the global amber funcs
	funcs := template.FuncMap{}

	for k, v := range amber.FuncMap { // add the amber's default funcs
		funcs[k] = v
	}

	for k, v := range s.funcs {
		funcs[k] = v
	}

	amber.FuncMap = funcs //set the funcs
	opt.Ext = extension

	templates, err := amber.CompileDir(dir, opt, amber.DefaultOptions) // this returns the map with stripped extension, we want extension so we copy the map
	if err == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.templateCache = make(map[string]*template.Template)
		for k, v := range templates {
			name := filepath.ToSlash(k + opt.Ext)
			s.templateCache[name] = v
			delete(templates, k)
		}

	}
	return err
}

// loadAssets builds the templates by binary, embedded.
func (s *AmberEngine) loadAssets() error {
	virtualDirectory, virtualExtension := s.directory, s.extension
	assetFn, namesFn := s.assetFn, s.namesFn

	// prepare the global amber funcs
	funcs := template.FuncMap{}

	for k, v := range amber.FuncMap { // add the amber's default funcs
		funcs[k] = v
	}

	for k, v := range s.funcs {
		funcs[k] = v
	}

	if len(virtualDirectory) > 0 {
		if virtualDirectory[0] == '.' { // first check for .wrong
			virtualDirectory = virtualDirectory[1:]
		}
		if virtualDirectory[0] == '/' || virtualDirectory[0] == filepath.Separator { // second check for /something, (or ./something if we had dot on 0 it will be removed
			virtualDirectory = virtualDirectory[1:]
		}
	}
	amber.FuncMap = funcs //set the funcs

	s.mu.Lock()
	defer s.mu.Unlock()

	names := namesFn()

	for _, path := range names {
		if !strings.HasPrefix(path, virtualDirectory) {
			continue
		}
		ext := filepath.Ext(path)
		if ext == virtualExtension {

			rel, err := filepath.Rel(virtualDirectory, path)
			if err != nil {
				return err
			}

			buf, err := assetFn(path)
			if err != nil {
				return err
			}

			name := filepath.ToSlash(rel)
			tmpl, err := amber.CompileData(buf, name, amber.DefaultOptions)

			if err != nil {
				return err
			}

			s.templateCache[name] = tmpl
		}
	}

	return nil
}

func (s *AmberEngine) fromCache(relativeName string) *template.Template {
	s.mu.Lock()
	tmpl, ok := s.templateCache[relativeName]
	if ok {
		s.mu.Unlock()
		return tmpl
	}
	s.mu.Unlock()
	return nil
}

// ExecuteWriter executes a template and writes its result to the w writer.
// layout here is useless.
func (s *AmberEngine) ExecuteWriter(w io.Writer, filename string, layout string, bindingData interface{}) error {
	// reload the templates if reload configuration field is true
	if s.reload {
		if err := s.Load(); err != nil {
			return err
		}
	}

	if tmpl := s.fromCache(filename); tmpl != nil {
		return tmpl.ExecuteTemplate(w, filename, bindingData)
	}

	return fmt.Errorf("Template with name %s doesn't exists in the dir", filename)
}
