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

	"github.com/kataras/iris/config"
)

type (
	Engine struct {
		Config    *config.Template
		Templates *template.Template
		// Middleware
		// Note:
		// I see that many template engines returns html/template as result
		// so I decided that the HTMLTemplate should accept a middleware for the final string content will be parsed to the main *html/template.Template
		// for example user of this property is Jade, currently
		Middleware func(string, string) (string, error)
	}
)

var emptyFuncs = template.FuncMap{
	"yield": func() (string, error) {
		return "", fmt.Errorf("yield was called, yet no layout defined")
	},
	"partial": func() (string, error) {
		return "", fmt.Errorf("block was called, yet no layout defined")
	},
	"current": func() (string, error) {
		return "", nil
	}, "render": func() (string, error) {
		return "", nil
	},
	// just for test with jade
	/*"bold": func() (string, error) {
		return "", nil
	},*/
}

// New creates and returns the HTMLTemplate template engine
func New(c config.Template) *Engine {
	return &Engine{Config: &c}
}

func (s *Engine) BuildTemplates() error {

	if s.Config.Asset == nil || s.Config.AssetNames == nil {
		return s.buildFromDir()

	}
	return s.buildFromAsset()

}

func (s *Engine) buildFromDir() error {
	if s.Config.Directory == "" {
		return nil //we don't return fill error here(yet)
	}

	var templateErr error
	/*var minifier *minify.M
	if s.Config.Minify {
		minifier = minify.New()
		minifier.AddFunc("text/html", htmlMinifier.Minify)
	} // Note: minifier has bugs, I complety remove this from Iris.
	*/
	dir := s.Config.Directory
	s.Templates = template.New(dir)
	s.Templates.Delims(s.Config.HTMLTemplate.Left, s.Config.HTMLTemplate.Right)
	hasMiddleware := s.Middleware != nil
	// Walk the supplied directory and compile any files that match our extension list.
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

		for _, extension := range s.Config.Extensions {
			if ext == extension {

				buf, err := ioutil.ReadFile(path)
				contents := string(buf)
				/*if s.Config.Minify {
					buf, err = minifier.Bytes("text/html", buf)
				}*/

				if err != nil {
					templateErr = err
					break
				}

				name := filepath.ToSlash(rel)
				tmpl := s.Templates.New(name)

				if hasMiddleware {
					contents, err = s.Middleware(name, contents)
				}

				if err != nil {
					templateErr = err
					break
				}

				// Add our funcmaps.
				if s.Config.HTMLTemplate.Funcs != nil {
					tmpl.Funcs(s.Config.HTMLTemplate.Funcs)
				}

				tmpl.Funcs(emptyFuncs).Parse(contents)
				break
			}
		}
		return nil
	})

	return templateErr
}

func (s *Engine) buildFromAsset() error {
	var templateErr error
	dir := s.Config.Directory
	s.Templates = template.New(dir)
	s.Templates.Delims(s.Config.HTMLTemplate.Left, s.Config.HTMLTemplate.Right)

	for _, path := range s.Config.AssetNames() {
		if !strings.HasPrefix(path, dir) {
			continue
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			panic(err)
		}

		ext := ""
		if strings.Index(rel, ".") != -1 {
			ext = "." + strings.Join(strings.Split(rel, ".")[1:], ".")
		}

		for _, extension := range s.Config.Extensions {
			if ext == extension {

				buf, err := s.Config.Asset(path)
				if err != nil {
					panic(err)
				}
				name := filepath.ToSlash(rel)
				tmpl := s.Templates.New(name)

				// Add our funcmaps.
				//for _, funcs := range s.Config.HTMLTemplate.Funcs {
				if s.Config.HTMLTemplate.Funcs != nil {
					tmpl.Funcs(s.Config.HTMLTemplate.Funcs)
				}

				tmpl.Funcs(emptyFuncs).Parse(string(buf))
				break
			}
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
			if s.Config.HTMLTemplate.RequirePartials || s.Templates.Lookup(fullPartialName) != nil {
				buf, err := s.executeTemplateBuf(fullPartialName, binding)
				// Return safe HTML here since we are rendering our own template.
				return template.HTML(buf.String()), err
			}
			return "", nil
		},
		"render": func(fullPartialName string) (template.HTML, error) {
			buf, err := s.executeTemplateBuf(fullPartialName, binding)
			// Return safe HTML here since we are rendering our own template.
			return template.HTML(buf.String()), err

		},
		// just for test with jade
		/*"bold": func(content string) (template.HTML, error) {
			return template.HTML("<b>" + content + "</b>"), nil
		},*/
	}
	if tpl := s.Templates.Lookup(name); tpl != nil {
		tpl.Funcs(funcs)
	}
}

func (s *Engine) ExecuteWriter(out io.Writer, name string, binding interface{}, layout string) error {
	if layout != "" && layout != config.NoLayout {
		s.layoutFuncsFor(name, binding)
		name = layout
	}
	return s.Templates.ExecuteTemplate(out, name, binding)
}
