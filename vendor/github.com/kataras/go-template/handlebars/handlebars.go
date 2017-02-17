// Package handlebars the HandlebarsEngine's functionality
package handlebars

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aymerick/raymond"
)

const (
	// NoLayout to disable layout for a particular template file
	NoLayout = "@.|.@no_layout@.|.@"
)

type (
	// Engine the Handlebars engine
	Engine struct {
		Config        Config
		templateCache map[string]*raymond.Template
		mu            sync.Mutex
	}
)

// New creates and returns the Handlebars template engine
func New(cfg ...Config) *Engine {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}

	// cuz mergo has a little bug on maps
	if c.Helpers == nil {
		c.Helpers = make(map[string]interface{}, 0)
	}

	e := &Engine{Config: c, templateCache: make(map[string]*raymond.Template, 0)}

	raymond.RegisterHelper("render", func(partial string, binding interface{}) raymond.SafeString {
		contents, err := e.executeTemplateBuf(partial, binding)
		if err != nil {
			return raymond.SafeString("Template with name: " + partial + " couldn't not be found.")
		}
		return raymond.SafeString(contents)
	})

	return e
}

// Funcs should returns the helper funcs
func (e *Engine) Funcs() map[string]interface{} {
	return e.Config.Helpers
}

// LoadDirectory builds the handlebars templates
func (e *Engine) LoadDirectory(dir string, extension string) error {

	// register the global helpers on the first load
	if len(e.templateCache) == 0 && e.Config.Helpers != nil {
		raymond.RegisterHelpers(e.Config.Helpers)

	}

	// the render works like {{ render "myfile.html" theContext.PartialContext}}
	// instead of the html/template engine which works like {{ render "myfile.html"}} and accepts the parent binding, with handlebars we can't do that because of lack of runtime helpers (dublicate error)
	e.mu.Lock()
	defer e.mu.Unlock()
	var templateErr error
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		ext := filepath.Ext(rel)
		if ext == extension {
			buf, err := ioutil.ReadFile(path)
			contents := string(buf)

			if err != nil {
				templateErr = err
				return err
			}

			name := filepath.ToSlash(rel)

			tmpl, err := raymond.Parse(contents)
			if err != nil {
				templateErr = err
				return err
			}
			e.templateCache[name] = tmpl
		}
		return nil
	})

	return templateErr
}

// LoadAssets loads the templates by binary
func (e *Engine) LoadAssets(virtualDirectory string, virtualExtension string, assetFn func(name string) ([]byte, error), namesFn func() []string) error {
	// register the global helpers
	if len(e.templateCache) == 0 && e.Config.Helpers != nil {
		raymond.RegisterHelpers(e.Config.Helpers)
	}
	if len(virtualDirectory) > 0 {
		if virtualDirectory[0] == '.' { // first check for .wrong
			virtualDirectory = virtualDirectory[1:]
		}
		if virtualDirectory[0] == '/' || virtualDirectory[0] == os.PathSeparator { // second check for /something, (or ./something if we had dot on 0 it will be removed
			virtualDirectory = virtualDirectory[1:]
		}
	}
	var templateErr error

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
				templateErr = err
				return err
			}

			buf, err := assetFn(path)
			if err != nil {
				templateErr = err
				return err
			}
			contents := string(buf)
			name := filepath.ToSlash(rel)

			tmpl, err := raymond.Parse(contents)
			if err != nil {
				templateErr = err
				return err
			}
			e.templateCache[name] = tmpl

		}
	}
	return templateErr
}

func (e *Engine) fromCache(relativeName string) *raymond.Template {
	e.mu.Lock()
	tmpl, ok := e.templateCache[relativeName]
	if ok {
		e.mu.Unlock()
		return tmpl
	}
	e.mu.Unlock()
	return nil
}

func (e *Engine) executeTemplateBuf(name string, binding interface{}) (string, error) {
	if tmpl := e.fromCache(name); tmpl != nil {
		return tmpl.Exec(binding)
	}
	return "", nil
}

// ExecuteWriter executes a templates and write its results to the out writer
func (e *Engine) ExecuteWriter(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) error {

	isLayout := false

	layout := e.Config.Layout
	renderFilename := name
	if len(options) > 0 {
		layoutOpt := options[0]["layout"]
		if layoutOpt != nil {
			if l, ok := layoutOpt.(string); ok {
				if l != "" {
					layout = l
				}
			}
		}
	}

	if layout != "" && layout != NoLayout {
		isLayout = true
		renderFilename = layout // the render becomes the layout, and the name is the partial.
	}

	if tmpl := e.fromCache(renderFilename); tmpl != nil {
		if isLayout {
			var context map[string]interface{}
			if m, is := binding.(map[string]interface{}); is { //handlebars accepts maps,
				context = m
			} else {
				return fmt.Errorf("Please provide a map[string]interface{} type as the binding instead of the %#v", binding)
			}

			contents, err := e.executeTemplateBuf(name, binding)
			if err != nil {
				return err
			}
			if context == nil {
				context = make(map[string]interface{}, 1)
			}
			// I'm implemented the {{ yield }} as with the rest of template engines, so this is not inneed for iris, but the user can do that manually if want
			// there is no performanrce different: raymond.RegisterPartialTemplate(name, tmpl)
			context["yield"] = raymond.SafeString(contents)
		}

		res, err := tmpl.Exec(binding)

		if err != nil {
			return err
		}
		_, err = fmt.Fprint(out, res)
		return err
	}

	return fmt.Errorf("[IRIS TEMPLATES] Template with name %s[original name = %s] doesn't exists in the dir", renderFilename, name)
}

// ExecuteRaw receives, parse and executes raw source template contents
// it's super-simple function without options and funcs, it's not used widely
// implements the EngineRawExecutor interface
func (e *Engine) ExecuteRaw(src string, wr io.Writer, binding interface{}) (err error) {
	tmpl, err := raymond.Parse(src)
	if err != nil {
		return err
	}
	parsed, err := tmpl.Exec(binding)
	if err != nil {
		return err
	}
	_, err = wr.Write([]byte(parsed))
	return
}
