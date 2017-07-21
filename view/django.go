package view

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	stdPath "path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/flosch/pongo2"
	"github.com/kataras/iris/context"
)

type (
	// Value conversion for pongo2.Value
	Value pongo2.Value
	// Error conversion for pongo2.Error
	Error pongo2.Error
	// FilterFunction conversion for pongo2.FilterFunction
	FilterFunction func(in *Value, param *Value) (out *Value, err *Error)
)

type tDjangoAssetLoader struct {
	baseDir  string
	assetGet func(name string) ([]byte, error)
}

// Abs calculates the path to a given template. Whenever a path must be resolved
// due to an import from another template, the base equals the parent template's path.
func (dal *tDjangoAssetLoader) Abs(base, name string) string {
	if stdPath.IsAbs(name) {
		return name
	}

	return stdPath.Join(dal.baseDir, name)
}

// Get returns an io.Reader where the template's content can be read from.
func (dal *tDjangoAssetLoader) Get(path string) (io.Reader, error) {
	if stdPath.IsAbs(path) {
		path = path[1:]
	}

	res, err := dal.assetGet(path)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(res), nil
}

// DjangoEngine contains the amber view engine structure.
type DjangoEngine struct {
	// files configuration
	directory string
	extension string
	assetFn   func(name string) ([]byte, error) // for embedded, in combination with directory & extension
	namesFn   func() []string                   // for embedded, in combination with directory & extension
	reload    bool
	//
	rmu sync.RWMutex // locks for filters and globals
	// filters for pongo2, map[name of the filter] the filter function . The filters are auto register
	filters map[string]FilterFunction
	// globals share context fields between templates. https://github.com/flosch/pongo2/issues/35
	globals       map[string]interface{}
	mu            sync.Mutex // locks for template cache
	templateCache map[string]*pongo2.Template
}

var _ Engine = &DjangoEngine{}

// Django creates and returns a new amber view engine.
func Django(directory, extension string) *DjangoEngine {
	s := &DjangoEngine{
		directory:     directory,
		extension:     extension,
		globals:       make(map[string]interface{}, 0),
		filters:       make(map[string]FilterFunction, 0),
		templateCache: make(map[string]*pongo2.Template, 0),
	}

	return s
}

// Ext returns the file extension which this view engine is responsible to render.
func (s *DjangoEngine) Ext() string {
	return s.extension
}

// Binary optionally, use it when template files are distributed
// inside the app executable (.go generated files).
//
// The assetFn and namesFn can come from the go-bindata library.
func (s *DjangoEngine) Binary(assetFn func(name string) ([]byte, error), namesFn func() []string) *DjangoEngine {
	s.assetFn, s.namesFn = assetFn, namesFn
	return s
}

// Reload if setted to true the templates are reloading on each render,
// use it when you're in development and you're boring of restarting
// the whole app when you edit a template file.
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

// AddFilter adds a filter to the template.
func (s *DjangoEngine) AddFilter(filterName string, filterBody FilterFunction) *DjangoEngine {
	s.rmu.Lock()
	s.filters[filterName] = filterBody
	s.rmu.Unlock()
	return s
}

// Load parses the templates to the engine.
// It's alos responsible to add the necessary global functions.
//
// Returns an error if something bad happens, user is responsible to catch it.
func (s *DjangoEngine) Load() error {
	if s.assetFn != nil && s.namesFn != nil {
		// embedded
		return s.loadAssets()
	}

	// load from directory, make the dir absolute here too.
	dir, err := filepath.Abs(s.directory)
	if err != nil {
		return err
	}
	// change the directory field configuration, load happens after directory has been setted, so we will not have any problems here.
	s.directory = dir
	return s.loadDirectory()
}

// this exists because of moving the pongo2 to the vendors without conflictitions if users
// wants to register pongo2 filters they can use this django.FilterFunc to do so.
func (s *DjangoEngine) convertFilters() map[string]pongo2.FilterFunction {
	filters := make(map[string]pongo2.FilterFunction, len(s.filters))
	for k, v := range s.filters {
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

// LoadDirectory loads the templates from directory.
func (s *DjangoEngine) loadDirectory() (templateErr error) {
	dir, extension := s.directory, s.extension

	fsLoader, err := pongo2.NewLocalFileSystemLoader(dir) // I see that this doesn't read the content if already parsed, so do it manually via filepath.Walk
	if err != nil {
		return err
	}

	set := pongo2.NewSet("", fsLoader)
	set.Globals = getPongoContext(s.globals)

	// set the filters
	filters := s.convertFilters()
	for filterName, filterFunc := range filters {
		pongo2.RegisterFilter(filterName, filterFunc)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

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

				s.templateCache[name], templateErr = set.FromString(string(buf))

				if templateErr != nil {
					return templateErr
				}
			}

		}
		return nil
	})

	return
}

// loadAssets loads the templates by binary (go-bindata for embedded).
func (s *DjangoEngine) loadAssets() error {

	virtualDirectory, virtualExtension := s.directory, s.extension
	assetFn, namesFn := s.assetFn, s.namesFn

	// Make a file set with a template loader based on asset function
	set := pongo2.NewSet("", &tDjangoAssetLoader{baseDir: s.directory, assetGet: s.assetFn})
	set.Globals = getPongoContext(s.globals)

	// set the filters
	filters := s.convertFilters()
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

	s.mu.Lock()
	defer s.mu.Unlock()

	var templateErr error

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
			s.templateCache[name], err = set.FromString(string(buf))
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

	if contextData, isContextViewData := templateData.(context.Map); isContextViewData {
		return pongo2.Context(contextData)
	}

	return templateData.(map[string]interface{})
}

func (s *DjangoEngine) fromCache(relativeName string) *pongo2.Template {
	s.mu.Lock()

	tmpl, ok := s.templateCache[relativeName]

	if ok {
		s.mu.Unlock()
		return tmpl
	}
	s.mu.Unlock()
	return nil
}

// ExecuteWriter executes a templates and write its results to the w writer
// layout here is useless.
func (s *DjangoEngine) ExecuteWriter(w io.Writer, filename string, layout string, bindingData interface{}) error {
	// reload the templates if reload configuration field is true
	if s.reload {
		if err := s.Load(); err != nil {
			return err
		}
	}

	if tmpl := s.fromCache(filename); tmpl != nil {
		return tmpl.ExecuteWriter(getPongoContext(bindingData), w)
	}

	return fmt.Errorf("template with name %s doesn't exists in the dir", filename)
}
