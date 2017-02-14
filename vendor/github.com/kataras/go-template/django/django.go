package django

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/flosch/pongo2"
)

type (
	// Engine the pongo2 engine
	Engine struct {
		Config        Config
		templateCache map[string]*pongo2.Template
		mu            sync.Mutex
	}
)

const (
	templateErrorMessage = `
	<html><body>
	<h2>Error in template</h2>
	<h3>%s</h3>
	</body></html>`
)

// New creates and returns a Pongo template engine
func New(cfg ...Config) *Engine {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	// cuz mergo has a little bug on maps
	if c.Globals == nil {
		c.Globals = make(map[string]interface{}, 0)
	}
	if c.Filters == nil {
		c.Filters = make(map[string]FilterFunction, 0)
	}

	return &Engine{Config: c, templateCache: make(map[string]*pongo2.Template)}
}

// Funcs should returns the helper funcs
func (p *Engine) Funcs() map[string]interface{} {
	return p.Config.Globals
}

// this exists because of moving the pongo2 to the vendors without conflictitions if users
// wants to register pongo2 filters they can use this django.FilterFunc to do so.
func (p *Engine) convertFilters() map[string]pongo2.FilterFunction {
	filters := make(map[string]pongo2.FilterFunction, len(p.Config.Filters))
	for k, v := range p.Config.Filters {
		func(filterName string, filterFunc FilterFunction) {
			fn := pongo2.FilterFunction(func(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
				theOut, theErr := filterFunc((*Value)(in), (*Value)(param))
				return (*pongo2.Value)(theOut), (*pongo2.Error)(theErr)
			})
			filters[filterName] = fn
		}(k, v)
	}
	return filters
}

// LoadDirectory builds the templates
func (p *Engine) LoadDirectory(dir string, extension string) (templateErr error) {
	fsLoader, err := pongo2.NewLocalFileSystemLoader(dir) // I see that this doesn't read the content if already parsed, so do it manually via filepath.Walk
	if err != nil {
		return err
	}

	set := pongo2.NewSet("", fsLoader)
	set.Globals = getPongoContext(p.Config.Globals)

	// set the filters
	filters := p.convertFilters()
	for filterName, filterFunc := range filters {
		pongo2.RegisterFilter(filterName, filterFunc)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Walk the supplied directory and compile any files that match our extension list.
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// Fix same-extension-dirs bug: some dir might be named to: "users.tmpl", "local.html".
		// These dirs should be excluded as they are not valid golang templates, but files under
		// them should be treat as normal.
		// If is a dir, return immediately (dir is not a valid golang template).
		if info == nil || info.IsDir() {

		} else {

			rel, err := filepath.Rel(dir, path)
			if err != nil {
				templateErr = err
				return err
			}

			ext := filepath.Ext(rel)
			if ext == extension {

				buf, err := ioutil.ReadFile(path)
				if err != nil {
					templateErr = err
					return err
				}
				name := filepath.ToSlash(rel)

				p.templateCache[name], templateErr = set.FromString(string(buf))

				if templateErr != nil && p.Config.DebugTemplates {
					p.templateCache[name], _ = set.FromString(
						fmt.Sprintf(templateErrorMessage, templateErr.Error()))
				}
			}

		}
		return nil
	})

	return
}

// LoadAssets loads the templates by binary
func (p *Engine) LoadAssets(virtualDirectory string, virtualExtension string, assetFn func(name string) ([]byte, error), namesFn func() []string) error {
	var templateErr error
	/*fsLoader, err := pongo2.NewLocalFileSystemLoader(virtualDirectory)
	if err != nil {
		return err
	}*/
	set := pongo2.NewSet("", pongo2.DefaultLoader)
	set.Globals = getPongoContext(p.Config.Globals)

	// set the filters
	filters := p.convertFilters()
	for filterName, filterFunc := range filters {
		pongo2.RegisterFilter(filterName, filterFunc)
	}

	if len(virtualDirectory) > 0 {
		if virtualDirectory[0] == '.' { // first check for .wrong
			virtualDirectory = virtualDirectory[1:]
		}
		if virtualDirectory[0] == '/' || virtualDirectory[0] == os.PathSeparator { // second check for /something, (or ./something if we had dot on 0 it will be removed
			virtualDirectory = virtualDirectory[1:]
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	names := namesFn()
	for _, path := range names {
		if !strings.HasPrefix(path, virtualDirectory) {
			continue
		}

		rel, err := filepath.Rel(virtualDirectory, path)
		if err != nil {
			templateErr = err
			return err
		}

		ext := filepath.Ext(rel)
		if ext == virtualExtension {

			buf, err := assetFn(path)
			if err != nil {
				templateErr = err
				return err
			}
			name := filepath.ToSlash(rel)
			p.templateCache[name], err = set.FromString(string(buf))
			if err != nil {
				templateErr = err
				return err
			}
		}
	}
	return templateErr
}

// getPongoContext returns the pongo2.Context from map[string]interface{} or from pongo2.Context, used internaly
func getPongoContext(templateData interface{}) pongo2.Context {
	if templateData == nil {
		return nil
	}

	if contextData, isPongoContext := templateData.(pongo2.Context); isPongoContext {
		return contextData
	}

	return templateData.(map[string]interface{})
}

func (p *Engine) fromCache(relativeName string) *pongo2.Template {
	p.mu.Lock() // defer is slow

	tmpl, ok := p.templateCache[relativeName]

	if ok {
		p.mu.Unlock()
		return tmpl
	}
	p.mu.Unlock()
	return nil
}

// ExecuteWriter executes a templates and write its results to the out writer
// layout here is useless
func (p *Engine) ExecuteWriter(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) error {
	if tmpl := p.fromCache(name); tmpl != nil {
		return tmpl.ExecuteWriter(getPongoContext(binding), out)
	}

	return fmt.Errorf("[IRIS TEMPLATES] Template with name %s doesn't exists in the dir", name)
}

// ExecuteRaw receives, parse and executes raw source template contents
// it's super-simple function without options and funcs, it's not used widely
// implements the EngineRawExecutor interface
func (p *Engine) ExecuteRaw(src string, wr io.Writer, binding interface{}) (err error) {
	set := pongo2.NewSet("", pongo2.DefaultLoader)
	set.Globals = getPongoContext(p.Config.Globals)
	tmpl, err := set.FromString(src)
	if err != nil {
		return err
	}
	return tmpl.ExecuteWriter(getPongoContext(binding), wr)
}
