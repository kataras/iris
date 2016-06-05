package config

import (
	"html/template"

	"github.com/flosch/pongo2"
	"github.com/imdario/mergo"
)

const (
	// NoEngine is a Template's config for engine type
	// when use this, the templates are disabled
	NoEngine EngineType = -1
	// HTMLEngine is a Template's config for engine type
	// when use this, the templates are html/template
	HTMLEngine EngineType = 0
	// PongoEngine is a Template's config for engine type
	// when use this, the templates are flosch/pongo2
	PongoEngine EngineType = 1
	// MarkdownEngine is a Template's config for engine type
	// when use this, the templates are .md files
	MarkdownEngine EngineType = 2
	// JadeEngine is a Template's config for engine type
	// when use this, the templates are joker/jade
	JadeEngine EngineType = 3
	// AmberEngine is a Template's config for engine type
	// when use this, the templates are eknkc/amber
	AmberEngine EngineType = 4
	// DefaultEngine is the HTMLEngine
	DefaultEngine EngineType = HTMLEngine

	// NoLayout to disable layout for a particular template file
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

	// EngineType is the type of template engine
	EngineType int8

	// Template the configs for templates (template/view engines)
	// contains common configs for all template engines
	Template struct {
		// Engine the type of template engine
		// default is DefaultEngine (HTMLEngine)
		Engine EngineType
		// Gzip enable gzip compression
		// default is false
		Gzip bool
		// Minify minifies the html result,
		// Note: according to this https://github.com/tdewolff/minify/issues/35, also it removes some </tags> when minify on writer, remove this from Iris until fix.
		// Default is false
		//Minify        bool

		// IsDevelopment re-builds the templates on each request
		// default is false
		IsDevelopment bool
		// Directory the system path which the templates live
		// default is ./templates
		Directory string
		// Extensions the allowed file extension
		// default is []string{".html"}
		Extensions []string
		// ContentType is the Content-Type response header
		// default is text/html but you can change if if needed
		ContentType string
		// Charset the charset, default is UTF-8
		Charset string
		// Asset is a func which returns bytes, use it to load the templates by binary
		Asset func(name string) ([]byte, error)
		// AssetNames should returns the template filenames, look Asset
		AssetNames func() []string
		// Layout the template file ( with its extension) which is the mother of all
		// use it to have it as a root file, and include others with {{ yield }}, refer  the docs
		Layout string

		// HTMLTemplate contains specific configs for HTMLTemplate standard html/template
		HTMLTemplate HTMLTemplate
		// Pongo contains specific configs for  for pongo2
		Pongo Pongo
		// Markdown contains specific configs for  for markdown
		// this doesn't supports Layout & binding context
		Markdown Markdown
		// Jade contains specific configs for jade
		Jade Jade
		// Amber contains specific configs for amber
		Amber Amber
	}

	// HTMLTemplate the configs for HTMLEngine
	HTMLTemplate struct {
		// RequirePartials default is false
		RequirePartials bool
		// Delims
		// Left delimeter, default is {{
		Left string
		// Right delimeter, default is }}
		Right string
		// Funcs like html/template
		Funcs map[string]interface{}
		// LayoutFuncs like html/template
		// the difference from Funcs is that these funcs
		// can be used inside a layout and can override the predefined (yield,partial...) or add more custom funcs
		// these can override the Funcs inside no-layout templates also, use it when you know what you're doing
		LayoutFuncs map[string]interface{}
	}
	// Pongo the configs for PongoEngine
	Pongo struct {
		// Filters for pongo2, map[name of the filter] the filter function . The filters are auto register
		Filters map[string]pongo2.FilterFunction
		// Globals share context fields between templates. https://github.com/flosch/pongo2/issues/35
		Globals map[string]interface{}
	}

	// Markdown the configs for MarkdownEngine
	Markdown struct {
		Sanitize bool // if true then returns safe html, default is false
	}

	// Jade the configs for JadeEngine
	Jade struct {
		// Funcs like html/template
		Funcs map[string]interface{}
		// LayoutFuncs like html/template
		// the difference from Funcs is that these funcs
		// can be used inside a layout and can override the predefined (yield,partial...) or add more custom funcs
		// these can override the Funcs inside no-layout templates also, use it when you know what you're doing
		LayoutFuncs map[string]interface{}
	}

	// Amber the configs for AmberEngine
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

// DefaultTemplate returns the default template configs
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
		HTMLTemplate:  HTMLTemplate{Left: "{{", Right: "}}", Funcs: make(map[string]interface{}, 0), LayoutFuncs: make(map[string]interface{}, 0)},
		Pongo:         Pongo{Filters: make(map[string]pongo2.FilterFunction, 0), Globals: make(map[string]interface{}, 0)},
		Markdown:      Markdown{Sanitize: false},
		Amber:         Amber{Funcs: template.FuncMap{}},
		Jade:          Jade{Funcs: template.FuncMap{}, LayoutFuncs: template.FuncMap{}},
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
