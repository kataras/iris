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

var (
	_ Engine       = (*BlocksEngine)(nil)
	_ EngineFuncer = (*BlocksEngine)(nil)
)

// WrapBlocks wraps an initialized blocks engine and returns its Iris adapter.
// See `Blocks` package-level function too.
func WrapBlocks(v *blocks.Blocks) *BlocksEngine {
	return &BlocksEngine{Engine: v}
}

// Blocks returns a new blocks view engine.
// The given "extension" MUST begin with a dot.
//
// See `WrapBlocks` package-level function too.
//
// Usage:
// Blocks("./views", ".html") or
// Blocks(iris.Dir("./views"), ".html") or
// Blocks(AssetFile(), ".html") for embedded data.
func Blocks(fs interface{}, extension string) *BlocksEngine {
	return WrapBlocks(blocks.New(fs).Extension(extension))
}

// Name returns the blocks engine's name.
func (s *BlocksEngine) Name() string {
	return "Blocks"
}

// RootDir sets the directory to use as the root one inside the provided File System.
func (s *BlocksEngine) RootDir(root string) *BlocksEngine {
	s.Engine.RootDir(root)
	return s
}

// LayoutDir sets a custom layouts directory,
// always relative to the "rootDir" one.
// Layouts are recognised by their prefix names.
// Defaults to "layouts".
func (s *BlocksEngine) LayoutDir(relToDirLayoutDir string) *BlocksEngine {
	s.Engine.LayoutDir(relToDirLayoutDir)
	return s
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
func (s *BlocksEngine) AddFunc(funcName string, funcBody interface{}) {
	s.Engine.Funcs(template.FuncMap{funcName: funcBody})
}

// AddLayoutFunc adds a template function for templates that are marked as layouts.
func (s *BlocksEngine) AddLayoutFunc(funcName string, funcBody interface{}) *BlocksEngine {
	s.Engine.LayoutFuncs(template.FuncMap{funcName: funcBody})
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
