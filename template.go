package iris

import (
	"compress/gzip"
	"io"

	"path/filepath"
	"sync"

	"github.com/iris-contrib/errors"
	"github.com/kataras/iris/utils"
)

var (
	builtinFuncs = [...]string{"url", "urlpath"}

	// DefaultTemplateDirectory the default directory if empty setted
	DefaultTemplateDirectory = "." + utils.PathSeparator + "templates"
)
var (
	// ContentTypeHTML the content type header for rendering
	// this can be changed
	ContentTypeHTML = "text/html"
	// Charset the charset header for rendering
	// this can be changed
	Charset = "UTF-8"
)

const (

	// DefaultTemplateExtension the default file extension if empty setted
	DefaultTemplateExtension = ".html"
	// NoLayout to disable layout for a particular template file
	NoLayout = "@.|.@iris_no_layout@.|.@"
	// TemplateLayoutContextKey is the name of the user values which can be used to set a template layout from a middleware and override the parent's
	TemplateLayoutContextKey = "templateLayout"
)

type (
	// TemplateEngine the interface that all template engines must implement
	TemplateEngine interface {
		// LoadDirectory builds the templates, usually by directory and extension but these are engine's decisions
		LoadDirectory(directory string, extension string) error
		// LoadAssets loads the templates by binary
		// assetFn is a func which returns bytes, use it to load the templates by binary
		// namesFn returns the template filenames
		LoadAssets(virtualDirectory string, virtualExtension string, assetFn func(name string) ([]byte, error), namesFn func() []string) error

		// ExecuteWriter finds, execute a template and write its result to the out writer
		// options are the optional runtime options can be passed by user and catched by the template engine when render
		// an example of this is the "layout" or "gzip" option
		ExecuteWriter(out io.Writer, name string, binding interface{}, options ...map[string]interface{}) error
	}

	// TemplateEngineFuncs is optional interface for the TemplateEngine
	// used to insert the Iris' standard funcs, see var 'usedFuncs'
	TemplateEngineFuncs interface {
		// Funcs should returns the context or the funcs,
		// this property is used in order to register the iris' helper funcs
		Funcs() map[string]interface{}
	}
)

type (
	// TemplateFuncs is is a helper type for map[string]interface{}
	TemplateFuncs map[string]interface{}
	// RenderOptions is a helper type for  the optional runtime options can be passed by user when Render
	// an example of this is the "layout" or "gzip" option
	// same as Map but more specific name
	RenderOptions map[string]interface{}
)

// IsFree returns true if a function can be inserted to this map
// return false if this key is already used by Iris
func (t TemplateFuncs) IsFree(key string) bool {
	for i := range builtinFuncs {
		if builtinFuncs[i] == key {
			return false
		}
	}
	return true
}

type (
	// TemplateEngineLocation contains the funcs to set the location for the templates by directory or by binary
	TemplateEngineLocation struct {
		directory string
		extension string
		assetFn   func(name string) ([]byte, error)
		namesFn   func() []string
	}
	// TemplateEngineBinaryLocation called after TemplateEngineLocation's Directory, used when files are distrubuted inside the app executable
	TemplateEngineBinaryLocation struct {
		location *TemplateEngineLocation
	}
)

// Directory sets the directory to load from
// returns the Binary location which is optional
func (t *TemplateEngineLocation) Directory(dir string, fileExtension string) TemplateEngineBinaryLocation {
	t.directory = dir
	t.extension = fileExtension
	return TemplateEngineBinaryLocation{location: t}
}

// Binary sets the asset(s) and asssets names to load from, works with Directory
func (t *TemplateEngineBinaryLocation) Binary(assetFn func(name string) ([]byte, error), namesFn func() []string) {
	t.location.assetFn = assetFn
	t.location.namesFn = namesFn
	// if extension is not static(setted by .Directory)
	if t.location.extension == "" {
		if names := namesFn(); len(names) > 0 {
			t.location.extension = filepath.Ext(names[0]) // we need the extension to get the correct template engine on the Render method
		}
	}
}

func (t *TemplateEngineLocation) isBinary() bool {
	return t.assetFn != nil && t.namesFn != nil
}

// TemplateEngineWrapper is the wrapper of a template engine
type TemplateEngineWrapper struct {
	TemplateEngine
	location            *TemplateEngineLocation
	buffer              *utils.BufferPool
	gzipWriterPool      sync.Pool
	reload              bool
	combiledContentType string
}

var (
	errMissingDirectoryOrAssets = errors.New("Missing Directory or Assets by binary for the template engine!")
	errNoTemplateEngineForExt   = errors.New("No template engine found to manage '%s' extensions")
)

func (t *TemplateEngineWrapper) load() error {
	if t.location.isBinary() {
		t.LoadAssets(t.location.directory, t.location.extension, t.location.assetFn, t.location.namesFn)
	} else if t.location.directory != "" {
		t.LoadDirectory(t.location.directory, t.location.extension)
	} else {
		return errMissingDirectoryOrAssets.Return()
	}
	return nil
}

// Execute execute a template and write its result to the context's body
// options are the optional runtime options can be passed by user and catched by the template engine when render
// an example of this is the "layout"
// note that gzip option is an iris dynamic option which exists for all template engines
func (t *TemplateEngineWrapper) Execute(ctx *Context, filename string, binding interface{}, options ...map[string]interface{}) (err error) {
	if t == nil {
		//file extension, but no template engine registered, this caused by context, and TemplateEngines. GetBy
		return errNoTemplateEngineForExt.Format(filepath.Ext(filename))
	}
	if t.reload {
		if err = t.load(); err != nil {
			return
		}
	}

	// we do all these because we don't want to initialize a new map for each execution...
	gzipEnabled := false
	if len(options) > 0 {
		gzipOpt := options[0]["gzip"] // we only need that, so don't create new map to keep the options.
		if b, isBool := gzipOpt.(bool); isBool {
			gzipEnabled = b
		}
	}

	ctxLayout := ctx.GetString(TemplateLayoutContextKey)
	if ctxLayout != "" {
		if len(options) > 0 {
			options[0]["layout"] = ctxLayout
		} else {
			options = []map[string]interface{}{map[string]interface{}{"layout": ctxLayout}}
		}
	}

	var out io.Writer
	if gzipEnabled {
		ctx.Response.Header.Add("Content-Encoding", "gzip")
		gzipWriter := t.gzipWriterPool.Get().(*gzip.Writer)
		gzipWriter.Reset(ctx.Response.BodyWriter())
		defer gzipWriter.Close()
		defer t.gzipWriterPool.Put(gzipWriter)
		out = gzipWriter
	} else {
		out = ctx.Response.BodyWriter()
	}
	ctx.SetHeader("Content-Type", t.combiledContentType)

	return t.ExecuteWriter(out, filename, binding, options...)
}

// ExecuteToString executes a template from a specific template engine and returns its contents result as string, it doesn't renders
func (t *TemplateEngineWrapper) ExecuteToString(filename string, binding interface{}, opt ...map[string]interface{}) (result string, err error) {
	if t == nil {
		//file extension, but no template engine registered, this caused by context, and TemplateEngines. GetBy
		return "", errNoTemplateEngineForExt.Format(filepath.Ext(filename))
	}
	if t.reload {
		if err = t.load(); err != nil {
			return
		}
	}

	out := t.buffer.Get()
	defer t.buffer.Put(out)
	err = t.ExecuteWriter(out, filename, binding, opt...)
	if err == nil {
		result = out.String()
	}
	return
}

// TemplateEngines is the container and manager of the template engines
type TemplateEngines struct {
	Helpers map[string]interface{}
	Engines []*TemplateEngineWrapper
	Reload  bool
}

// GetBy receives a filename, gets its extension and returns the template engine responsible for that file extension
func (t *TemplateEngines) GetBy(filename string) *TemplateEngineWrapper {
	extension := filepath.Ext(filename)
	for i, n := 0, len(t.Engines); i < n; i++ {
		e := t.Engines[i]

		if e.location.extension == extension {
			return e
		}
	}
	return nil
}

// Add adds but not loads a template engine
func (t *TemplateEngines) Add(e TemplateEngine) *TemplateEngineLocation {
	location := &TemplateEngineLocation{}
	// add the iris helper funcs
	if funcer, ok := e.(TemplateEngineFuncs); ok {
		if funcer.Funcs() != nil {
			for k, v := range t.Helpers {
				funcer.Funcs()[k] = v
			}
		}
	}

	tmplEngine := &TemplateEngineWrapper{
		TemplateEngine: e,
		location:       location,
		buffer:         utils.NewBufferPool(20),
		gzipWriterPool: sync.Pool{New: func() interface{} {
			return &gzip.Writer{}
		}},
		reload:              t.Reload,
		combiledContentType: ContentTypeHTML + "; " + Charset,
	}

	t.Engines = append(t.Engines, tmplEngine)
	return location
}

// LoadAll loads all templates using all template engines, returns the first error
// called on iris' initialize
func (t *TemplateEngines) LoadAll() error {
	for i, n := 0, len(t.Engines); i < n; i++ {
		e := t.Engines[i]
		if e.location.directory == "" {
			e.location.directory = DefaultTemplateDirectory // the defualt dir ./templates
		}
		if e.location.extension == "" {
			e.location.extension = DefaultTemplateExtension // the default file ext .html
		}

		if err := e.load(); err != nil {
			return err
		}
	}
	return nil
}
