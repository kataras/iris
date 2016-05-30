package config

import (
	"html/template"

	"github.com/flosch/pongo2"
	"github.com/imdario/mergo"
)

const (
	NoEngine       EngineType = -1
	HTMLEngine     EngineType = 0
	PongoEngine    EngineType = 1
	MarkdownEngine EngineType = 2
	JadeEngine     EngineType = 3
	AmberEngine    EngineType = 4

	DefaultEngine EngineType = HTMLEngine

	// to disable layout for a particular file
	NoLayout = "@.|.@iris_no_layout@.|.@"
)

var (
	// Charset character encoding.
	Charset = "UTF-8"
)

type (
	// Rest is a struct for specifying configuration options for the rest.Render object.
	Rest struct {
		// Appends the given character set to the Content-Type header. Default is "UTF-8".
		Charset string
		// Gzip enable it if you want to render with gzip compression. Default is false
		Gzip bool
		// Outputs human readable JSON.
		IndentJSON bool
		// Outputs human readable XML. Default is false.
		IndentXML bool
		// Prefixes the JSON output with the given bytes. Default is false.
		PrefixJSON []byte
		// Prefixes the XML output with the given bytes.
		PrefixXML []byte
		// Unescape HTML characters "&<>" to their original values. Default is false.
		UnEscapeHTML bool
		// Streams JSON responses instead of marshalling prior to sending. Default is false.
		StreamingJSON bool
		// Disables automatic rendering of http.StatusInternalServerError when an error occurs. Default is false.
		DisableHTTPErrorRendering bool
		// MarkdownSanitize sanitizes the markdown. Default is false.
		MarkdownSanitize bool
	}

	EngineType int8

	Template struct {
		// contains common configs for both HTMLTemplate & Pongo
		Engine EngineType
		Gzip   bool
		// Minify minifies the html result,
		// Note: according to this https://github.com/tdewolff/minify/issues/35, also it removes some </tags> when minify on writer, remove this from Iris until fix.
		// Default is false
		//Minify        bool
		IsDevelopment bool
		Directory     string
		Extensions    []string
		ContentType   string
		Charset       string
		Asset         func(name string) ([]byte, error)
		AssetNames    func() []string
		Layout        string

		HTMLTemplate HTMLTemplate // contains specific configs for  HTMLTemplate standard html/template
		Pongo        Pongo        // contains specific configs for pongo2
		// Markdown template engine it doesn't supports Layout & binding context
		Markdown Markdown // contains specific configs for markdown
		Jade     Jade     // contains specific configs for Jade
		Amber    Amber    // contains specific configs for Amber
	}

	HTMLTemplate struct {
		RequirePartials bool
		// Delims
		Left  string
		Right string
		// Funcs for HTMLTemplate html/template
		Funcs template.FuncMap
	}

	Pongo struct {
		// Filters for pongo2, map[name of the filter] the filter function . The filters are auto register
		Filters map[string]pongo2.FilterFunction
		// Globals share context fields between templates. https://github.com/flosch/pongo2/issues/35
		Globals map[string]interface{}
	}

	Markdown struct {
		Sanitize bool // if true then returns safe html, default is false
	}

	// Jade empty for now
	// stay tuned
	Jade struct {
	}

	Amber struct {
		// Funcs for the html/template result, amber default funcs are not overrided so use it without worries
		Funcs template.FuncMap
	}
)

// DefaultRest returns the default config for rest
func DefaultRest() Rest {
	return Rest{
		Charset:                   Charset,
		IndentJSON:                false,
		IndentXML:                 false,
		PrefixJSON:                []byte(""),
		PrefixXML:                 []byte(""),
		UnEscapeHTML:              false,
		StreamingJSON:             false,
		DisableHTTPErrorRendering: false,
		MarkdownSanitize:          false,
	}
}

// Merge merges the default with the given config and returns the result
func (c Rest) Merge(cfg []Rest) (config Rest) {

	if len(cfg) > 0 {
		config = cfg[0]
		mergo.Merge(&config, c)
	} else {
		_default := c
		config = _default
	}

	return
}

// MergeSingle merges the default with the given config and returns the result
func (c Rest) MergeSingle(cfg Rest) (config Rest) {

	config = cfg
	mergo.Merge(&config, c)

	return
}

func DefaultTemplate() Template {
	return Template{
		Engine:        DefaultEngine, //or HTMLTemplate
		Gzip:          false,
		IsDevelopment: false,
		Directory:     "templates",
		Extensions:    []string{".html"},
		ContentType:   "text/html",
		Charset:       "UTF-8",
		Layout:        "", // currently this is the only config which not working for pongo2 yet but I will find a way
		HTMLTemplate:  HTMLTemplate{Left: "{{", Right: "}}", Funcs: template.FuncMap{}},
		Pongo:         Pongo{Filters: make(map[string]pongo2.FilterFunction, 0), Globals: make(map[string]interface{}, 0)},
		Markdown:      Markdown{Sanitize: false},
		Amber:         Amber{Funcs: template.FuncMap{}},
		Jade:          Jade{},
	}
}

// Merge merges the default with the given config and returns the result
func (c Template) Merge(cfg []Template) (config Template) {

	if len(cfg) > 0 {
		config = cfg[0]
		mergo.Merge(&config, c)
	} else {
		_default := c
		config = _default
	}

	return
}

// MergeSingle merges the default with the given config and returns the result
func (c Template) MergeSingle(cfg Template) (config Template) {

	config = cfg
	mergo.Merge(&config, c)

	return
}
