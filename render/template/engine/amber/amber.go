package amber

import (
	"html/template"

	"fmt"
	"io"
	"path/filepath"
	"sync"

	"github.com/eknkc/amber"
	"github.com/kataras/iris/config"
)

type Engine struct {
	Config        *config.Template
	templateCache map[string]*template.Template
	mu            sync.Mutex
}

func New(cfg config.Template) *Engine {
	return &Engine{Config: &cfg}
}

func (e *Engine) BuildTemplates() error {
	opt := amber.DirOptions{}
	opt.Recursive = true
	if e.Config.Extensions == nil || len(e.Config.Extensions) == 0 {
		e.Config.Extensions = []string{".html"}
	}

	// prepare the global amber funcs
	funcs := template.FuncMap{}
	for k, v := range amber.FuncMap { // add the amber's default funcs
		funcs[k] = v
	}
	if e.Config.Amber.Funcs != nil { // add the config's funcs
		for k, v := range e.Config.Amber.Funcs {
			funcs[k] = v
		}
	}

	amber.FuncMap = funcs //set the funcs

	opt.Ext = e.Config.Extensions[0]
	templates, err := amber.CompileDir(e.Config.Directory, opt, amber.DefaultOptions) // this returns the map with stripped extension, we want extension so we copy the map
	if err == nil {
		e.templateCache = make(map[string]*template.Template)
		for k, v := range templates {
			name := filepath.ToSlash(k + opt.Ext)
			e.templateCache[name] = v
			delete(templates, k)
		}

	}
	return err

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

func (e *Engine) ExecuteWriter(out io.Writer, name string, binding interface{}, layout string) error {
	if tmpl := e.fromCache(name); tmpl != nil {
		return tmpl.ExecuteTemplate(out, name, binding)
	}

	return fmt.Errorf("[IRIS TEMPLATES] Template with name %s doesn't exists in the dir %s", name, e.Config.Directory)
}
