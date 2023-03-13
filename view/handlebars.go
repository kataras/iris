package view

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kataras/iris/v12/context"

	"github.com/mailgun/raymond/v2"
)

// HandlebarsEngine contains the handlebars view engine structure.
type HandlebarsEngine struct {
	fs fs.FS
	// files configuration
	rootDir   string
	extension string
	// Not used anymore.
	// assetFn   func(name string) ([]byte, error) // for embedded, in combination with directory & extension
	// namesFn   func() []string                   // for embedded, in combination with directory & extension
	reload bool // if true, each time the ExecuteWriter is called the templates will be reloaded.
	// parser configuration
	layout        string
	rmu           sync.RWMutex
	funcs         template.FuncMap
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
// Handlebars(embed.FS, ".html") or Handlebars(AssetFile(), ".html") for embedded data.
func Handlebars(fs interface{}, extension string) *HandlebarsEngine {
	s := &HandlebarsEngine{
		fs:            getFS(fs),
		rootDir:       "/",
		extension:     extension,
		templateCache: make(map[string]*raymond.Template),
		funcs:         make(template.FuncMap), // global
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
	if s.fs != nil && root != "" && root != "/" && root != "." && root != s.rootDir {
		sub, err := fs.Sub(s.fs, s.rootDir)
		if err != nil {
			panic(err)
		}

		s.fs = sub // here so the "middleware" can work.
	}

	s.rootDir = filepath.ToSlash(root)
	return s
}

// Name returns the handlebars engine's name.
func (s *HandlebarsEngine) Name() string {
	return "Handlebars"
}

// Ext returns the file extension which this view engine is responsible to render.
// If the filename extension on ExecuteWriter is empty then this is appended.
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
// the {{ yield . }} func to yield the main template file
// and optionally {{partial/partial_r/render . }} to render
// other template files like headers and footers.
func (s *HandlebarsEngine) Layout(layoutFile string) *HandlebarsEngine {
	s.layout = layoutFile
	return s
}

// AddFunc adds a function to the templates.
// It is legal to overwrite elements of the default actions:
// - url func(routeName string, args ...string) string
// - urlpath func(routeName string, args ...string) string
// - render func(fullPartialName string) (raymond.HTML, error).
func (s *HandlebarsEngine) AddFunc(funcName string, funcBody interface{}) {
	s.rmu.Lock()
	s.funcs[funcName] = funcBody
	s.rmu.Unlock()
}

// AddGlobalFunc registers a global template function for all Handlebars view engines.
func (s *HandlebarsEngine) AddGlobalFunc(funcName string, funcBody interface{}) {
	s.rmu.Lock()
	raymond.RegisterHelper(funcName, funcBody)
	s.rmu.Unlock()
}

// Load parses the templates to the engine.
// It is responsible to add the necessary global functions.
//
// Returns an error if something bad happens, user is responsible to catch it.
func (s *HandlebarsEngine) Load() error {
	// If only custom templates should be loaded.
	if (s.fs == nil || context.IsNoOpFS(s.fs)) && len(s.templateCache) > 0 {
		return nil
	}

	rootDirName := getRootDirName(s.fs)

	return walk(s.fs, "", func(path string, info os.FileInfo, _ error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		if s.extension != "" {
			if !strings.HasSuffix(path, s.extension) {
				return nil
			}
		}

		if s.rootDir == rootDirName {
			path = strings.TrimPrefix(path, rootDirName)
			path = strings.TrimPrefix(path, "/")
		}

		contents, err := asset(s.fs, path)
		if err != nil {
			return err
		}
		return s.ParseTemplate(path, string(contents), nil)
	})
}

// ParseTemplate adds a custom template from text.
func (s *HandlebarsEngine) ParseTemplate(name string, contents string, funcs template.FuncMap) error {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	name = strings.TrimPrefix(name, "/")
	tmpl, err := raymond.Parse(contents)
	if err == nil {
		// Add functions for this template.
		for k, v := range s.funcs {
			tmpl.RegisterHelper(k, v)
		}

		for k, v := range funcs {
			tmpl.RegisterHelper(k, v)
		}

		s.templateCache[name] = tmpl
	}

	return err
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
				return fmt.Errorf("please provide a map[string]interface{} type as the binding instead of the %#v", binding)
			}

			contents, err := s.executeTemplateBuf(filename, binding)
			if err != nil {
				return err
			}
			if context == nil {
				context = make(map[string]interface{}, 1)
			}
			// I'm implemented the {{ yield . }} as with the rest of template engines, so this is not inneed for iris, but the user can do that manually if want
			// there is no performance cost: raymond.RegisterPartialTemplate(name, tmpl)
			context["yield"] = raymond.SafeString(contents)
		}

		res, err := tmpl.Exec(binding)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(w, res)
		return err
	}

	return ErrNotExist{
		Name:     fmt.Sprintf("%s (file: %s)", renderFilename, filename),
		IsLayout: false,
		Data:     bindingData,
	}
}
