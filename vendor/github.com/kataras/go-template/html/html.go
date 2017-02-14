package html

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// NoLayout to disable layout for a particular template file
	NoLayout = "@.|.@no_layout@.|.@"
)

type (
	// Engine the html/template engine
	Engine struct {
		Config     Config
		Middleware func(name string, contents string) (string, error)
		Templates  *template.Template
		mu         sync.Mutex
	}
)

var emptyFuncs = template.FuncMap{
	"yield": func() (string, error) {
		return "", fmt.Errorf("yield was called, yet no layout defined")
	},
	"partial": func() (string, error) {
		return "", fmt.Errorf("block was called, yet no layout defined")
	},
	"partial_r": func() (string, error) {
		return "", fmt.Errorf("block was called, yet no layout defined")
	},
	"current": func() (string, error) {
		return "", nil
	}, "render": func() (string, error) {
		return "", nil
	},
}

// New creates and returns the HTMLTemplate template engine
func New(cfg ...Config) *Engine {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}
	// cuz mergo has a little bug on maps
	if c.Funcs == nil {
		c.Funcs = make(map[string]interface{}, 0)
	}
	if c.LayoutFuncs == nil {
		c.LayoutFuncs = make(map[string]interface{}, 0)
	}
	e := &Engine{Config: c}
	return e
}

// Funcs should returns the helper funcs
func (s *Engine) Funcs() map[string]interface{} {
	return s.Config.Funcs
}

// LoadDirectory builds the templates
func (s *Engine) LoadDirectory(dir string, extension string) error {

	var templateErr error
	s.Templates = template.New(dir)
	s.Templates.Delims(s.Config.Left, s.Config.Right)

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {

		} else {

			rel, err := filepath.Rel(dir, path)
			if err != nil {
				templateErr = err
				return err
			}

			ext := filepath.Ext(path)
			if ext == extension {

				buf, err := ioutil.ReadFile(path)
				if err != nil {
					templateErr = err
					return err
				}

				contents := string(buf)

				if err == nil {

					name := filepath.ToSlash(rel)
					tmpl := s.Templates.New(name)

					if s.Middleware != nil {
						contents, err = s.Middleware(name, contents)
					}
					if err != nil {
						templateErr = err
						return err
					}
					s.mu.Lock()
					// Add our funcmaps.
					if s.Config.Funcs != nil {
						tmpl.Funcs(s.Config.Funcs)
					}

					tmpl.Funcs(emptyFuncs).Parse(contents)
					s.mu.Unlock()
				}
			}

		}
		return nil
	})

	return templateErr
}

// LoadAssets loads the templates by binary
func (s *Engine) LoadAssets(virtualDirectory string, virtualExtension string, assetFn func(name string) ([]byte, error), namesFn func() []string) error {
	var templateErr error
	s.Templates = template.New(virtualDirectory)
	s.Templates.Delims(s.Config.Left, s.Config.Right)
	names := namesFn()
	if len(virtualDirectory) > 0 {
		if virtualDirectory[0] == '.' { // first check for .wrong
			virtualDirectory = virtualDirectory[1:]
		}
		if virtualDirectory[0] == '/' || virtualDirectory[0] == os.PathSeparator { // second check for /something, (or ./something if we had dot on 0 it will be removed
			virtualDirectory = virtualDirectory[1:]
		}
	}

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
			tmpl := s.Templates.New(name)

			if s.Middleware != nil {
				contents, err = s.Middleware(name, contents)
			}
			if err != nil {
				templateErr = err
				return err
			}

			// Add our funcmaps.
			if s.Config.Funcs != nil {
				tmpl.Funcs(s.Config.Funcs)
			}

			tmpl.Funcs(emptyFuncs).Parse(contents)
		}
	}
	return templateErr
}

func (s *Engine) executeTemplateBuf(name string, binding interface{}) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	err := s.Templates.ExecuteTemplate(buf, name, binding)

	return buf, err
}

func (s *Engine) layoutFuncsFor(name string, binding interface{}) {
	funcs := template.FuncMap{
		"yield": func() (template.HTML, error) {
			buf, err := s.executeTemplateBuf(name, binding)
			// Return safe HTML here since we are rendering our own template.
			return template.HTML(buf.String()), err
		},
		"current": func() (string, error) {
			return name, nil
		},
		"partial": func(partialName string) (template.HTML, error) {
			fullPartialName := fmt.Sprintf("%s-%s", partialName, name)
			if s.Templates.Lookup(fullPartialName) != nil {
				buf, err := s.executeTemplateBuf(fullPartialName, binding)
				return template.HTML(buf.String()), err
			}
			return "", nil
		},
		//partial related to current page,
		//it would be easier for adding pages' style/script inline
		//for example when using partial_r '.script' in layout.html
		//templates/users/index.html would load templates/users/index.script.html
		"partial_r": func(partialName string) (template.HTML, error) {
			ext := filepath.Ext(name)
			root := name[:len(name)-len(ext)]
			fullPartialName := fmt.Sprintf("%s%s%s", root, partialName, ext)
			if s.Templates.Lookup(fullPartialName) != nil {
				buf, err := s.executeTemplateBuf(fullPartialName, binding)
				return template.HTML(buf.String()), err
			}
			return "", nil
		},
		"render": func(fullPartialName string) (template.HTML, error) {
			buf, err := s.executeTemplateBuf(fullPartialName, binding)
			return template.HTML(buf.String()), err
		},
	}
	_userLayoutFuncs := s.Config.LayoutFuncs
	if _userLayoutFuncs != nil && len(_userLayoutFuncs) > 0 {
		for k, v := range _userLayoutFuncs {
			funcs[k] = v
		}
	}
	if tpl := s.Templates.Lookup(name); tpl != nil {
		tpl.Funcs(funcs)
	}
}

func (s *Engine) runtimeFuncsFor(name string, binding interface{}) {
	funcs := template.FuncMap{
		"render": func(fullPartialName string) (template.HTML, error) {
			buf, err := s.executeTemplateBuf(fullPartialName, binding)
			return template.HTML(buf.String()), err
		},
	}

	if tpl := s.Templates.Lookup(name); tpl != nil {
		tpl.Funcs(funcs)
	}
}

// ExecuteWriter executes a templates and write its results to the out writer
func (s *Engine) ExecuteWriter(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) error {
	layout := s.Config.Layout

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
		s.layoutFuncsFor(name, binding)
		name = layout
	} else {
		s.runtimeFuncsFor(name, binding)
	}

	return s.Templates.ExecuteTemplate(out, name, binding)
}

// ExecuteRaw receives, parse and executes raw source template contents
// it's super-simple function without options and funcs, it's not used widely
// implements the EngineRawExecutor interface
func (s *Engine) ExecuteRaw(src string, wr io.Writer, binding interface{}) (err error) {
	if s.Middleware != nil {
		src, err = s.Middleware("", src)
		if err != nil {
			return err
		}
	}

	tmpl, err := template.New("").Parse(src)
	if err != nil {
		return err
	}
	return tmpl.Execute(wr, binding)
}
