package view

import (
	"html/template"
	"io"

	"github.com/kataras/blocks"
)

// BlocksEngine is an Iris view engine adapter for the blocks view engine.
// The blocks engine is based on the html/template standard Go package.
//
// To initialize a fresh one use the `Blocks` function.
// To wrap an existing one use the `WrapBlocks` function.
//
// It contains the following four default template functions:
// - url "routename" parameters...
// - urlpath "routename" parameters...
// - tr "language" "key" arguments...
// - partial "template_name" data
//
// Read more at: https://github.com/kataras/blocks.
type BlocksEngine struct {
	Engine *blocks.Blocks
}

var _ Engine = (*BlocksEngine)(nil)

// WrapBlocks wraps an initialized blocks engine and returns its Iris adapter.
// See `Blocks` package-level function too.
func WrapBlocks(v *blocks.Blocks) *BlocksEngine {
	return &BlocksEngine{Engine: v}
}

// Blocks returns a new blocks view engine.
// The given "extension" MUST begin with a dot.
//
// See `WrapBlocks` package-level function too.
func Blocks(directory, extension string) *BlocksEngine {
	return WrapBlocks(blocks.New(directory).Extension(extension))
}

// Ext returns empty ext as this template engine
// supports template blocks without file suffix.
// Note that, if more than one view engine is registered to a single
// Iris application then, this Blocks engine should be the last entry one.
func (s *BlocksEngine) Ext() string {
	return ""
}

// AddFunc implements the `EngineFuncer` which is being used
// by the framework to add template functions like:
// - url func(routeName string, args ...string) string
// - urlpath func(routeName string, args ...string) string
// - tr func(lang, key string, args ...interface{}) string
func (s *BlocksEngine) AddFunc(funcName string, funcBody interface{}) *BlocksEngine {
	s.Engine.Funcs(template.FuncMap{funcName: funcBody})
	return s
}

// AddLayoutFunc adds a template function for templates that are marked as layouts.
func (s *BlocksEngine) AddLayoutFunc(funcName string, funcBody interface{}) *BlocksEngine {
	s.Engine.LayoutFuncs(template.FuncMap{funcName: funcBody})
	return s
}

// Binary sets the function which reads contents based on a filename
// and a function that returns all the filenames.
// These functions are used to parse the corresponding files into templates.
// You do not need to set them when the given "rootDir" was a system directory.
// It's mostly useful when the application contains embedded template files,
// e.g. pass go-bindata's `Asset` and `AssetNames` functions
// to load templates from go-bindata generated content.
func (s *BlocksEngine) Binary(asset blocks.AssetFunc, assetNames blocks.AssetNamesFunc) *BlocksEngine {
	s.Engine.Assets(asset, assetNames)
	return s
}

// Layout sets the default layout which inside should use
// the {{ template "content" . }} to render the main template.
//
// Example for ./views/layouts/main.html:
// Blocks("./views", ".html").Layout("layouts/main")
func (s *BlocksEngine) Layout(layoutName string) *BlocksEngine {
	s.Engine.DefaultLayout(layoutName)
	return s
}

// Reload if called with a true parameter,
// each `ExecuteWriter` call will re-parse the templates.
// Useful when the application is at a development stage.
func (s *BlocksEngine) Reload(b bool) *BlocksEngine {
	s.Engine.Reload(b)
	return s
}

// Load parses the files into templates.
func (s *BlocksEngine) Load() error {
	return s.Engine.Load()
}

// ExecuteWriter renders a template on "w".
func (s *BlocksEngine) ExecuteWriter(w io.Writer, tmplName, layoutName string, data interface{}) error {
	if layoutName == NoLayout {
		layoutName = ""
	}

	return s.Engine.ExecuteTemplate(w, tmplName, layoutName, data)
}
