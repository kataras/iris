package view

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aymerick/raymond"
)

// HandlebarsEngine contains the handlebars view engine structure.
type HandlebarsEngine struct {
	fs http.FileSystem
	// files configuration
	rootDir   string
	extension string
	assetFn   func(name string) ([]byte, error) // for embedded, in combination with directory & extension
	namesFn   func() []string                   // for embedded, in combination with directory & extension
	reload    bool                              // if true, each time the ExecuteWriter is called the templates will be reloaded.
	// parser configuration
	layout        string
	rmu           sync.RWMutex
	helpers       map[string]interface{}
	templateCache map[string]*raymond.Template
}

var (
	_ Engine       = (*HandlebarsEngine)(nil)
	_ EngineFuncer = (*HandlebarsEngine)(nil)
)

// Handlebars creates and returns a new handlebars view engine.
// The given "extension" MUST begin with a dot.
//
// Usage:
// Handlebars("./views", ".html") or
// Handlebars(iris.Dir("./views"), ".html") or
// Handlebars(AssetFile(), ".html") for embedded data.
func Handlebars(fs interface{}, extension string) *HandlebarsEngine {
	s := &HandlebarsEngine{
		fs:            getFS(fs),
		rootDir:       "/",
		extension:     extension,
		templateCache: make(map[string]*raymond.Template),
		helpers:       make(map[string]interface{}),
	}

	// register the render helper here
	raymond.RegisterHelper("render", func(partial string, binding interface{}) raymond.SafeString {
		contents, err := s.executeTemplateBuf(partial, binding)
		if err != nil {
			return raymond.SafeString("template with name: " + partial + " couldn't not be found.")
		}
		return raymond.SafeString(contents)
	})

	return s
}

// RootDir sets the directory to be used as a starting point
// to load templates from the provided file system.
func (s *HandlebarsEngine) RootDir(root string) *HandlebarsEngine {
	s.rootDir = filepath.ToSlash(root)
	return s
}

// Ext returns the file extension which this view engine is responsible to render.
func (s *HandlebarsEngine) Ext() string {
	return s.extension
}

// Reload if set to true the templates are reloading on each render,
// use it when you're in development and you're boring of restarting
// the whole app when you edit a template file.
//
// Note that if `true` is passed then only one `View -> ExecuteWriter` will be render each time,
// no concurrent access across clients, use it only on development status.
// It's good to be used side by side with the https://github.com/kataras/rizla reloader for go source files.
func (s *HandlebarsEngine) Reload(developmentMode bool) *HandlebarsEngine {
	s.reload = developmentMode
	return s
}

// Layout sets the layout template file which should use
// the {{ yield }} func to yield the main template file
// and optionally {{partial/partial_r/render}} to render
// other template files like headers and footers.
func (s *HandlebarsEngine) Layout(layoutFile string) *HandlebarsEngine {
	s.layout = layoutFile
	return s
}

// AddFunc adds the function to the template's function map.
// It is legal to overwrite elements of the default actions:
// - url func(routeName string, args ...string) string
// - urlpath func(routeName string, args ...string) string
// - render func(fullPartialName string) (raymond.HTML, error).
func (s *HandlebarsEngine) AddFunc(funcName string, funcBody interface{}) {
	s.rmu.Lock()
	s.helpers[funcName] = funcBody
	s.rmu.Unlock()
}

// Load parses the templates to the engine.
// It is responsible to add the necessary global functions.
//
// Returns an error if something bad happens, user is responsible to catch it.
func (s *HandlebarsEngine) Load() error {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	// register the global helpers on the first load
	if len(s.templateCache) == 0 && s.helpers != nil {
		raymond.RegisterHelpers(s.helpers)
	}

	return walk(s.fs, s.rootDir, func(path string, info os.FileInfo, _ error) error {
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
			return err
		}

		name := strings.TrimPrefix(path, "/")
		tmpl, err := raymond.Parse(string(buf))
		if err != nil {
			return err
		}
		s.templateCache[name] = tmpl

		return nil
	})
}

func (s *HandlebarsEngine) fromCache(relativeName string) *raymond.Template {
	if s.reload {
		s.rmu.RLock()
		defer s.rmu.RUnlock()
	}

	if tmpl, ok := s.templateCache[relativeName]; ok {
		return tmpl
	}

	return nil
}

func (s *HandlebarsEngine) executeTemplateBuf(name string, binding interface{}) (string, error) {
	if tmpl := s.fromCache(name); tmpl != nil {
		return tmpl.Exec(binding)
	}
	return "", nil
}

// ExecuteWriter executes a template and writes its result to the w writer.
func (s *HandlebarsEngine) ExecuteWriter(w io.Writer, filename string, layout string, bindingData interface{}) error {
	// re-parse the templates if reload is enabled.
	if s.reload {
		if err := s.Load(); err != nil {
			return err
		}
	}

	isLayout := false
	layout = getLayout(layout, s.layout)
	renderFilename := filename

	if layout != "" {
		isLayout = true
		renderFilename = layout // the render becomes the layout, and the name is the partial.
	}

	if tmpl := s.fromCache(renderFilename); tmpl != nil {
		binding := bindingData
		if isLayout {
			var context map[string]interface{}
			if m, is := binding.(map[string]interface{}); is { // handlebars accepts maps,
				context = m
			} else {
				return fmt.Errorf("Please provide a map[string]interface{} type as the binding instead of the %#v", binding)
			}

			contents, err := s.executeTemplateBuf(filename, binding)
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
		_, err = fmt.Fprint(w, res)
		return err
	}

	return ErrNotExist{fmt.Sprintf("%s (file: %s)", renderFilename, filename), false}
}
