package view

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	stdPath "path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kataras/iris/v12/context"

	"github.com/fatih/structs"
	"github.com/flosch/pongo2/v4"
)

type (
	// Value type alias for pongo2.Value
	Value = pongo2.Value
	// Error type alias for pongo2.Error
	Error = pongo2.Error
	// FilterFunction type alias for pongo2.FilterFunction
	FilterFunction = pongo2.FilterFunction

	// Parser type alias for pongo2.Parser
	Parser = pongo2.Parser
	// Token type alias for pongo2.Token
	Token = pongo2.Token
	// INodeTag type alias for pongo2.InodeTag
	INodeTag = pongo2.INodeTag
	// TagParser the function signature of the tag's parser you will have
	// to implement in order to create a new tag.
	//
	// 'doc' is providing access to the whole document while 'arguments'
	// is providing access to the user's arguments to the tag:
	//
	//     {% your_tag_name some "arguments" 123 %}
	//
	// start_token will be the *Token with the tag's name in it (here: your_tag_name).
	//
	// Please see the Parser documentation on how to use the parser.
	// See `RegisterTag` for more information about writing a tag as well.
	TagParser = pongo2.TagParser
)

// AsValue converts any given value to a pongo2.Value
// Usually being used within own functions passed to a template
// through a Context or within filter functions.
//
// Example:
//
//	AsValue("my string")
//
// Shortcut for `pongo2.AsValue`.
var AsValue = pongo2.AsValue

// AsSafeValue works like AsValue, but does not apply the 'escape' filter.
// Shortcut for `pongo2.AsSafeValue`.
var AsSafeValue = pongo2.AsSafeValue

type tDjangoAssetLoader struct {
	rootDir string
	fs      fs.FS
}

// Abs calculates the path to a given template. Whenever a path must be resolved
// due to an import from another template, the base equals the parent template's path.
func (l *tDjangoAssetLoader) Abs(base, name string) string {
	if stdPath.IsAbs(name) {
		return name
	}

	return stdPath.Join(l.rootDir, name)
}

// Get returns an io.Reader where the template's content can be read from.
func (l *tDjangoAssetLoader) Get(path string) (io.Reader, error) {
	if stdPath.IsAbs(path) {
		path = path[1:]
	}

	res, err := asset(l.fs, path)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(res), nil
}

// DjangoEngine contains the django view engine structure.
type DjangoEngine struct {
	fs fs.FS
	// files configuration
	rootDir   string
	extension string
	reload    bool
	//
	rmu sync.RWMutex // locks for filters, globals and `ExecuteWiter` when `reload` is true.
	// filters for pongo2, map[name of the filter] the filter function . The filters are auto register
	filters map[string]FilterFunction
	// globals share context fields between templates.
	globals       map[string]interface{}
	Set           *pongo2.TemplateSet
	templateCache map[string]*pongo2.Template
}

var (
	_ Engine       = (*DjangoEngine)(nil)
	_ EngineFuncer = (*DjangoEngine)(nil)
)

// Django creates and returns a new django view engine.
// The given "extension" MUST begin with a dot.
//
// Usage:
// Django("./views", ".html") or
// Django(iris.Dir("./views"), ".html") or
// Django(embed.FS, ".html") or Django(AssetFile(), ".html") for embedded data.
func Django(fs interface{}, extension string) *DjangoEngine {
	s := &DjangoEngine{
		fs:            getFS(fs),
		rootDir:       "/",
		extension:     extension,
		globals:       make(map[string]interface{}),
		filters:       make(map[string]FilterFunction),
		templateCache: make(map[string]*pongo2.Template),
	}

	return s
}

// RootDir sets the directory to be used as a starting point
// to load templates from the provided file system.
func (s *DjangoEngine) RootDir(root string) *DjangoEngine {
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

// Name returns the django engine's name.
func (s *DjangoEngine) Name() string {
	return "Django"
}

// Ext returns the file extension which this view engine is responsible to render.
// If the filename extension on ExecuteWriter is empty then this is appended.
func (s *DjangoEngine) Ext() string {
	return s.extension
}

// Reload if set to true the templates are reloading on each render,
// use it when you're in development and you're boring of restarting
// the whole app when you edit a template file.
//
// Note that if `true` is passed then only one `View -> ExecuteWriter` will be render each time,
// no concurrent access across clients, use it only on development status.
// It's good to be used side by side with the https://github.com/kataras/rizla reloader for go source files.
func (s *DjangoEngine) Reload(developmentMode bool) *DjangoEngine {
	s.reload = developmentMode
	return s
}

// AddFunc adds the function to the template's Globals.
// It is legal to overwrite elements of the default actions:
// - url func(routeName string, args ...string) string
// - urlpath func(routeName string, args ...string) string
// - render func(fullPartialName string) (template.HTML, error).
func (s *DjangoEngine) AddFunc(funcName string, funcBody interface{}) {
	s.rmu.Lock()
	s.globals[funcName] = funcBody
	s.rmu.Unlock()
}

// AddFilter registers a new filter. If there's already a filter with the same
// name, RegisterFilter will panic. You usually want to call this
// function in the filter's init() function:
// http://golang.org/doc/effective_go.html#init
//
// Same as `RegisterFilter`.
func (s *DjangoEngine) AddFilter(filterName string, filterBody FilterFunction) *DjangoEngine {
	return s.registerFilter(filterName, filterBody)
}

// RegisterFilter registers a new filter. If there's already a filter with the same
// name, RegisterFilter will panic. You usually want to call this
// function in the filter's init() function:
// http://golang.org/doc/effective_go.html#init
//
// See http://www.florian-schlachter.de/post/pongo2/ for more about
// writing filters and tags.
func (s *DjangoEngine) RegisterFilter(filterName string, filterBody FilterFunction) *DjangoEngine {
	return s.registerFilter(filterName, filterBody)
}

func (s *DjangoEngine) registerFilter(filterName string, fn FilterFunction) *DjangoEngine {
	pongo2.RegisterFilter(filterName, fn)
	return s
}

// RegisterTag registers a new tag. You usually want to call this
// function in the tag's init() function:
// http://golang.org/doc/effective_go.html#init
//
// See http://www.florian-schlachter.de/post/pongo2/ for more about
// writing filters and tags.
func (s *DjangoEngine) RegisterTag(tagName string, fn TagParser) error {
	return pongo2.RegisterTag(tagName, fn)
}

// Load parses the templates to the engine.
// It is responsible to add the necessary global functions.
//
// Returns an error if something bad happens, user is responsible to catch it.
func (s *DjangoEngine) Load() error {
	// If only custom templates should be loaded.
	if (s.fs == nil || context.IsNoOpFS(s.fs)) && len(s.templateCache) > 0 {
		return nil
	}

	rootDirName := getRootDirName(s.fs)

	return walk(s.fs, "", func(path string, info os.FileInfo, err error) error {
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

		contents, err := asset(s.fs, path)
		if err != nil {
			return err
		}

		return s.ParseTemplate(path, contents)
	})
}

// ParseTemplate adds a custom template from text.
// This parser does not support funcs per template. Use the `AddFunc` instead.
func (s *DjangoEngine) ParseTemplate(name string, contents []byte) error {
	s.rmu.Lock()
	defer s.rmu.Unlock()

	s.initSet()

	name = strings.TrimPrefix(name, "/")
	tmpl, err := s.Set.FromBytes(contents)
	if err == nil {
		s.templateCache[name] = tmpl
	}

	return err
}

func (s *DjangoEngine) initSet() { // protected by the caller.
	if s.Set == nil {
		s.Set = pongo2.NewSet("", &tDjangoAssetLoader{fs: s.fs, rootDir: s.rootDir})
		s.Set.Globals = getPongoContext(s.globals)
	}
}

// getPongoContext returns the pongo2.Context from map[string]interface{} or from pongo2.Context, used internaly
func getPongoContext(templateData interface{}) pongo2.Context {
	if templateData == nil {
		return nil
	}

	switch data := templateData.(type) {
	case pongo2.Context:
		return data
	case context.Map:
		return pongo2.Context(data)
	default:
		// if struct, convert it to map[string]interface{}
		if structs.IsStruct(data) {
			return pongo2.Context(structs.Map(data))
		}

		panic("django: template data: should be a map or struct")
	}
}

func (s *DjangoEngine) fromCache(relativeName string) *pongo2.Template {
	if s.reload {
		s.rmu.RLock()
		defer s.rmu.RUnlock()
	}

	if tmpl, ok := s.templateCache[relativeName]; ok {
		return tmpl
	}
	return nil
}

// ExecuteWriter executes a templates and write its results to the w writer
// layout here is useless.
func (s *DjangoEngine) ExecuteWriter(w io.Writer, filename string, _ string, bindingData interface{}) error {
	// re-parse the templates if reload is enabled.
	if s.reload {
		if err := s.Load(); err != nil {
			return err
		}
	}

	if tmpl := s.fromCache(filename); tmpl != nil {
		return tmpl.ExecuteWriter(getPongoContext(bindingData), w)
	}

	return ErrNotExist{Name: filename, IsLayout: false, Data: bindingData}
}
