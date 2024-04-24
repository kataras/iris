package view

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kataras/iris/v12/context"
)

// HTMLEngine contains the html view engine structure.
type HTMLEngine struct {
	name string // the view engine's name, can be HTML, Ace or Pug.
	// the file system to load from.
	fs fs.FS
	// files configuration
	rootDir   string
	extension string
	// if true, each time the ExecuteWriter is called the templates will be reloaded,
	// each ExecuteWriter waits to be finished before writing to a new one.
	reload bool
	// parser configuration
	options     []string // (text) template options
	left        string
	right       string
	layout      string
	rmu         sync.RWMutex // locks for layoutFuncs and funcs
	layoutFuncs template.FuncMap
	funcs       template.FuncMap

	//
	middleware  func(name string, contents []byte) (string, error)
	onLoad      func()
	onLoaded    func()
	Templates   *template.Template
	customCache []customTmp // required to load them again if reload is true.
	bufPool     *sync.Pool
	//
}

type customTmp struct {
	name     string
	contents []byte
	funcs    template.FuncMap
}

var (
	_ Engine       = (*HTMLEngine)(nil)
	_ EngineFuncer = (*HTMLEngine)(nil)
)

// HTML creates and returns a new html view engine.
// The html engine used like the "html/template" standard go package
// but with a lot of extra features.
// The given "extension" MUST begin with a dot.
//
// Usage:
// HTML("./views", ".html") or
// HTML(iris.Dir("./views"), ".html") or
// HTML(embed.FS, ".html") or HTML(AssetFile(), ".html") for embedded data or
// HTML("","").ParseTemplate("hello", `[]byte("hello {{.Name}}")`, nil) for custom template parsing only.
func HTML(dirOrFS interface{}, extension string) *HTMLEngine {
	s := &HTMLEngine{
		name:      "HTML",
		fs:        getFS(dirOrFS),
		rootDir:   "/",
		extension: extension,
		reload:    false,
		left:      "{{",
		right:     "}}",
		layout:    "",
		layoutFuncs: template.FuncMap{
			"yield": func(binding interface{}) template.HTML {
				return template.HTML("")
			},
		},
		funcs: make(template.FuncMap),
		bufPool: &sync.Pool{New: func() interface{} {
			return new(bytes.Buffer)
		}},
	}

	return s
}

// RootDir sets the directory to be used as a starting point
// to load templates from the provided file system.
func (s *HTMLEngine) RootDir(root string) *HTMLEngine {
	if s.fs != nil && root != "" && root != "/" && root != "." && root != s.rootDir {
		sub, err := fs.Sub(s.fs, root)
		if err != nil {
			panic(err)
		}
		s.fs = sub // here so the "middleware" can work.
	}

	s.rootDir = filepath.ToSlash(root)
	return s
}

// FS change templates DIR
func (s *HTMLEngine) FS(dirOrFS interface{}) *HTMLEngine {
	s.fs = getFS(dirOrFS)
	return s
}

// Name returns the engine's name.
func (s *HTMLEngine) Name() string {
	return s.name
}

// Ext returns the file extension which this view engine is responsible to render.
// If the filename extension on ExecuteWriter is empty then this is appended.
func (s *HTMLEngine) Ext() string {
	return s.extension
}

// Reload if set to true the templates are reloading on each render,
// use it when you're in development and you're boring of restarting
// the whole app when you edit a template file.
//
// Note that if `true` is passed then only one `View -> ExecuteWriter` will be render each time,
// no concurrent access across clients, use it only on development status.
// It's good to be used side by side with the https://github.com/kataras/rizla reloader for go source files.
func (s *HTMLEngine) Reload(developmentMode bool) *HTMLEngine {
	s.reload = developmentMode
	return s
}

// Option sets options for the template. Options are described by
// strings, either a simple string or "key=value". There can be at
// most one equals sign in an option string. If the option string
// is unrecognized or otherwise invalid, Option panics.
//
// Known options:
//
// missingkey: Control the behavior during execution if a map is
// indexed with a key that is not present in the map.
//
//	"missingkey=default" or "missingkey=invalid"
//		The default behavior: Do nothing and continue execution.
//		If printed, the result of the index operation is the string
//		"<no value>".
//	"missingkey=zero"
//		The operation returns the zero value for the map type's element.
//	"missingkey=error"
//		Execution stops immediately with an error.
func (s *HTMLEngine) Option(opt ...string) *HTMLEngine {
	s.rmu.Lock()
	s.options = append(s.options, opt...)
	s.rmu.Unlock()
	return s
}

// Delims sets the action delimiters to the specified strings, to be used in
// templates. An empty delimiter stands for the
// corresponding default: {{ or }}.
func (s *HTMLEngine) Delims(left, right string) *HTMLEngine {
	s.left, s.right = left, right
	return s
}

// Layout sets the layout template file which inside should use
// the {{ yield . }} func to yield the main template file
// and optionally {{partial/partial_r/render . }} to render other template files like headers and footers
//
// The 'tmplLayoutFile' is a relative path of the templates base directory,
// for the template file with its extension.
//
// Example: HTML("./templates", ".html").Layout("layouts/mainLayout.html")
//
//	// mainLayout.html is inside: "./templates/layouts/".
//
// Note: Layout can be changed for a specific call
// action with the option: "layout" on the iris' context.Render function.
func (s *HTMLEngine) Layout(layoutFile string) *HTMLEngine {
	s.layout = layoutFile
	return s
}

// AddLayoutFunc adds the function to the template's layout-only function map.
// It is legal to overwrite elements of the default layout actions:
// - yield func() (template.HTML, error)
// - current  func() (string, error)
// - partial func(partialName string) (template.HTML, error)
// - partial_r func(partialName string) (template.HTML, error)
// - render func(fullPartialName string) (template.HTML, error).
func (s *HTMLEngine) AddLayoutFunc(funcName string, funcBody interface{}) *HTMLEngine {
	s.rmu.Lock()
	s.layoutFuncs[funcName] = funcBody
	s.rmu.Unlock()
	return s
}

// AddFunc adds the function to the template's function map.
// It is legal to overwrite elements of the default actions:
// - url func(routeName string, args ...string) string
// - urlpath func(routeName string, args ...string) string
// - render func(fullPartialName string) (template.HTML, error).
// - tr func(lang, key string, args ...interface{}) string
func (s *HTMLEngine) AddFunc(funcName string, funcBody interface{}) {
	s.rmu.Lock()
	s.funcs[funcName] = funcBody
	s.rmu.Unlock()
}

// SetFuncs overrides the template funcs with the given "funcMap".
func (s *HTMLEngine) SetFuncs(funcMap template.FuncMap) *HTMLEngine {
	s.rmu.Lock()
	s.funcs = funcMap
	s.rmu.Unlock()

	return s
}

// Funcs adds the elements of the argument map to the template's function map.
// It is legal to overwrite elements of the map. The return
// value is the template, so calls can be chained.
func (s *HTMLEngine) Funcs(funcMap template.FuncMap) *HTMLEngine {
	s.rmu.Lock()
	for k, v := range funcMap {
		s.funcs[k] = v
	}
	s.rmu.Unlock()

	return s
}

// Load parses the templates to the engine.
// It's also responsible to add the necessary global functions.
//
// Returns an error if something bad happens, caller is responsible to handle that.
func (s *HTMLEngine) Load() error {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	return s.load()
}

func (s *HTMLEngine) load() error {
	if s.onLoad != nil {
		s.onLoad()
	}

	if err := s.reloadCustomTemplates(); err != nil {
		return err
	}

	// If only custom templates should be loaded.
	if (s.fs == nil || context.IsNoOpFS(s.fs)) && len(s.Templates.DefinedTemplates()) > 0 {
		return nil
	}

	rootDirName := getRootDirName(s.fs)

	err := walk(s.fs, "", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

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

		buf, err := asset(s.fs, path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}

		return s.parseTemplate(path, buf, nil)
	})

	if s.onLoaded != nil {
		s.onLoaded()
	}

	if err != nil {
		return err
	}

	if s.Templates == nil {
		return fmt.Errorf("no templates found")
	}

	return nil
}

func (s *HTMLEngine) reloadCustomTemplates() error {
	for _, tmpl := range s.customCache {
		if err := s.parseTemplate(tmpl.name, tmpl.contents, tmpl.funcs); err != nil {
			return err
		}
	}

	return nil
}

// ParseTemplate adds a custom template to the root template.
func (s *HTMLEngine) ParseTemplate(name string, contents []byte, funcs template.FuncMap) (err error) {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.customCache = append(s.customCache, customTmp{
		name:     name,
		contents: contents,
		funcs:    funcs,
	})

	return s.parseTemplate(name, contents, funcs)
}

func (s *HTMLEngine) parseTemplate(name string, contents []byte, funcs template.FuncMap) (err error) {
	s.initRootTmpl()

	name = strings.TrimPrefix(name, "/")
	tmpl := s.Templates.New(name)
	// tmpl.Option("missingkey=error")
	tmpl.Option(s.options...)

	var text string

	if s.middleware != nil {
		text, err = s.middleware(name, contents)
		if err != nil {
			return
		}
	} else {
		text = string(contents)
	}

	tmpl.Funcs(s.getBuiltinFuncs(name)).Funcs(s.funcs)

	if strings.Contains(name, "layout") {
		tmpl.Funcs(s.layoutFuncs)
	}

	if len(funcs) > 0 {
		tmpl.Funcs(funcs) // custom for this template.
	}
	_, err = tmpl.Parse(text)
	return
}

func (s *HTMLEngine) initRootTmpl() { // protected by the caller.
	if s.Templates == nil {
		// the root template should be the same,
		// no matter how many reloads as the
		// following unexported fields cannot be modified.
		// However, on reload they should be cleared otherwise we get an error.
		s.Templates = template.New(s.rootDir)
		s.Templates.Delims(s.left, s.right)
	}
}

func (s *HTMLEngine) executeTemplateBuf(name string, binding interface{}) (string, error) {
	buf := s.bufPool.Get().(*bytes.Buffer)
	buf.Reset()

	err := s.Templates.ExecuteTemplate(buf, name, binding)
	result := buf.String()
	s.bufPool.Put(buf)
	return result, err
}

func (s *HTMLEngine) getBuiltinRuntimeLayoutFuncs(name string) template.FuncMap {
	funcs := template.FuncMap{
		"yield": func(binding interface{}) (template.HTML, error) {
			result, err := s.executeTemplateBuf(name, binding)
			// Return safe HTML here since we are rendering our own template.
			return template.HTML(result), err
		},
	}

	return funcs
}

func (s *HTMLEngine) getBuiltinFuncs(name string) template.FuncMap {
	funcs := template.FuncMap{
		"part": func(partName string, binding interface{}) (template.HTML, error) {
			nameTemp := strings.ReplaceAll(name, s.extension, "")
			fullPartName := fmt.Sprintf("%s-%s", nameTemp, partName)
			result, err := s.executeTemplateBuf(fullPartName, binding)
			if err != nil {
				return "", nil
			}
			return template.HTML(result), err
		},
		"current": func() (string, error) {
			return name, nil
		},
		"partial": func(partialName string, binding interface{}) (template.HTML, error) {
			fullPartialName := fmt.Sprintf("%s-%s", partialName, name)
			if s.Templates.Lookup(fullPartialName) != nil {
				result, err := s.executeTemplateBuf(fullPartialName, binding)
				return template.HTML(result), err
			}
			return "", nil
		},
		// partial related to current page,
		// it would be easier for adding pages' style/script inline
		// for example when using partial_r '.script' in layout.html
		// templates/users/index.html would load templates/users/index.script.html
		"partial_r": func(partialName string, binding interface{}) (template.HTML, error) {
			ext := filepath.Ext(name)
			root := name[:len(name)-len(ext)]
			fullPartialName := fmt.Sprintf("%s%s%s", root, partialName, ext)
			if s.Templates.Lookup(fullPartialName) != nil {
				result, err := s.executeTemplateBuf(fullPartialName, binding)
				return template.HTML(result), err
			}
			return "", nil
		},
		"render": func(fullPartialName string, binding interface{}) (template.HTML, error) {
			result, err := s.executeTemplateBuf(fullPartialName, binding)
			return template.HTML(result), err
		},
	}

	return funcs
}

// ExecuteWriter executes a template and writes its result to the w writer.
func (s *HTMLEngine) ExecuteWriter(w io.Writer, name string, layout string, bindingData interface{}) error {
	// re-parse the templates if reload is enabled.
	if s.reload {
		s.rmu.Lock()
		defer s.rmu.Unlock()

		s.Templates = nil
		// we lose the templates parsed manually, so store them when it's called
		// in order for load to take care of them too.

		if err := s.load(); err != nil {
			return err
		}
	}

	if layout = getLayout(layout, s.layout); layout != "" {
		lt := s.Templates.Lookup(layout)
		if lt == nil {
			return ErrNotExist{Name: layout, IsLayout: true, Data: bindingData}
		}

		return lt.Funcs(s.getBuiltinRuntimeLayoutFuncs(name)).Execute(w, bindingData)
	}

	t := s.Templates.Lookup(name)
	if t == nil {
		return ErrNotExist{Name: name, IsLayout: false, Data: bindingData}
	}

	return t.Execute(w, bindingData)
}
