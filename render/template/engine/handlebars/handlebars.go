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
	"github.com/kataras/iris/config"
)

type (
	// Engine the Handlebars engine
	Engine struct {
		Config        *config.Template
		templateCache map[string]*raymond.Template
		mu            sync.Mutex
	}
)

// New creates and returns the Handlebars template engine
func New(c config.Template) *Engine {
	s := &Engine{Config: &c, templateCache: make(map[string]*raymond.Template, 0)}
	return s
}

// BuildTemplates builds the handlebars templates
func (e *Engine) BuildTemplates() error {
	if e.Config.Extensions == nil || len(e.Config.Extensions) == 0 {
		e.Config.Extensions = []string{".html"}
	}

	// register the global helpers
	if e.Config.Handlebars.Helpers != nil {
		raymond.RegisterHelpers(e.Config.Handlebars.Helpers)
	}

	// the render works like {{ render "myfile.html" theContext.PartialContext}}
	// instead of the html/template engine which works like {{ render "myfile.html"}} and accepts the parent binding, with handlebars we can't do that because of lack of runtime helpers (dublicate error)
	raymond.RegisterHelper("render", func(partial string, binding interface{}) raymond.SafeString {
		contents, err := e.executeTemplateBuf(partial, binding)
		if err != nil {
			return raymond.SafeString("Template with name: " + partial + " couldn't not be found.")
		}
		return raymond.SafeString(contents)
	})

	var templateErr error

	dir := e.Config.Directory
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		ext := ""
		if strings.Index(rel, ".") != -1 {
			ext = filepath.Ext(rel)
		}

		for _, extension := range e.Config.Extensions {
			if ext == extension {

				buf, err := ioutil.ReadFile(path)
				contents := string(buf)

				if err != nil {
					templateErr = err
					break
				}

				name := filepath.ToSlash(rel)

				tmpl, err := raymond.Parse(contents)
				if err != nil {
					templateErr = err
					continue
				}
				e.mu.Lock()
				e.templateCache[name] = tmpl
				e.mu.Unlock()

				break
			}
		}
		return nil
	})

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
func (e *Engine) ExecuteWriter(out io.Writer, name string, binding interface{}, layout string) error {

	isLayout := false

	renderFilename := name
	if layout != "" {
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

	return fmt.Errorf("[IRIS TEMPLATES] Template with name %s[original name = %s] doesn't exists in the dir %s", renderFilename, name, e.Config.Directory)
}
