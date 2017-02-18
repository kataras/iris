package amber

import (
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/eknkc/amber"
)

// Engine the amber template engine
type Engine struct {
	Config        Config
	templateCache map[string]*template.Template
	mu            sync.Mutex
}

// New creates and returns a new amber engine
func New(cfg ...Config) *Engine {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}

	// cuz mergo has a little bug on maps
	if c.Funcs == nil {
		c.Funcs = make(map[string]interface{}, 0)
	}

	return &Engine{Config: c}
}

// Funcs should returns the helper funcs
func (e *Engine) Funcs() map[string]interface{} {
	return e.Config.Funcs
}

// LoadDirectory builds the amber templates
func (e *Engine) LoadDirectory(directory string, extension string) error {
	opt := amber.DirOptions{}
	opt.Recursive = true

	// prepare the global amber funcs
	funcs := template.FuncMap{}

	for k, v := range amber.FuncMap { // add the amber's default funcs
		funcs[k] = v
	}

	if e.Config.Funcs != nil { // add the config's funcs
		for k, v := range e.Config.Funcs {
			funcs[k] = v
		}
	}

	amber.FuncMap = funcs //set the funcs
	opt.Ext = extension

	templates, err := amber.CompileDir(directory, opt, amber.DefaultOptions) // this returns the map with stripped extension, we want extension so we copy the map
	if err == nil {
		e.mu.Lock()
		defer e.mu.Unlock()
		e.templateCache = make(map[string]*template.Template)
		for k, v := range templates {
			name := filepath.ToSlash(k + opt.Ext)
			e.templateCache[name] = v
			delete(templates, k)
		}

	}
	return err
}

// LoadAssets builds the templates from binary assets
func (e *Engine) LoadAssets(virtualDirectory string, virtualExtension string, assetFn func(name string) ([]byte, error), namesFn func() []string) error {
	e.templateCache = make(map[string]*template.Template)
	// prepare the global amber funcs
	funcs := template.FuncMap{}

	for k, v := range amber.FuncMap { // add the amber's default funcs
		funcs[k] = v
	}

	if e.Config.Funcs != nil { // add the config's funcs
		for k, v := range e.Config.Funcs {
			funcs[k] = v
		}
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

	e.mu.Lock()
	defer e.mu.Unlock()

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

			e.templateCache[name] = tmpl
		}
	}

	return nil
}

func (e *Engine) fromCache(relativeName string) *template.Template {
	e.mu.Lock()
	tmpl, ok := e.templateCache[relativeName]
	if ok {
		e.mu.Unlock()
		return tmpl
	}
	e.mu.Unlock()
	return nil
}

// ExecuteWriter executes a templates and write its results to the out writer
func (e *Engine) ExecuteWriter(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) error {
	if tmpl := e.fromCache(name); tmpl != nil {
		return tmpl.ExecuteTemplate(out, name, binding)
	}

	return fmt.Errorf("[IRIS TEMPLATES] Template with name %s doesn't exists in the dir", name)
}

// ExecuteRaw receives, parse and executes raw source template contents
// it's super-simple function without options and funcs, it's not used widely
// implements the EngineRawExecutor interface
func (e *Engine) ExecuteRaw(src string, wr io.Writer, binding interface{}) (err error) {
	tmpl, err := amber.Compile(src, amber.DefaultOptions)
	if err != nil {
		return err
	}
	return tmpl.Execute(wr, binding)
}
