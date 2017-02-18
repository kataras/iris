package view

import (
	"github.com/kataras/go-template/html"
)

// HTMLAdaptor is the html engine policy adaptor
// used like the "html/template" standard go package
// but with a lot of extra features by.
//
// This is just a wrapper of kataras/go-template/html.
type HTMLAdaptor struct {
	*Adaptor
	engine *html.Engine
}

// HTML creates and returns a new kataras/go-template/html engine.
// The html engine used like the "html/template" standard go package
// but with a lot of extra features.
func HTML(directory string, extension string) *HTMLAdaptor {
	engine := html.New()
	return &HTMLAdaptor{
		Adaptor: NewAdaptor(directory, extension, engine),
		// create the underline engine with the default configuration,
		// which can be changed by this type.
		//The whole funcs should called only before Iris' build & listen state.
		engine: engine, // we need that for the configuration only.
	}

}

// Delims sets the action delimiters to the specified strings, to be used in
// subsequent calls to Parse, ParseFiles, or ParseGlob. Nested template
// definitions will inherit the settings. An empty delimiter stands for the
// corresponding default: {{ or }}.
func (h *HTMLAdaptor) Delims(left string, right string) *HTMLAdaptor {
	if left != "" && right != "" {
		h.engine.Config.Left = left
		h.engine.Config.Right = right
	}
	return h
}

// Layout sets the layout template file which inside should use
// the {{ yield }} func to yield the main template file
// and optionally {{partial/partial_r/render}} to render other template files like headers and footers
//
// The 'tmplLayoutFile' is a relative path of the templates base directory,
// for the template file with its extension.
//
// Example: HTML("./templates", ".html").Layout("layouts/mainLayout.html")
//         // mainLayout.html is inside: "./templates/layouts/".
//
// Note: Layout can be changed for a specific call
// action with the option: "layout" on the Iris' context.Render function.
func (h *HTMLAdaptor) Layout(tmplLayoutFile string) *HTMLAdaptor {
	h.engine.Config.Layout = tmplLayoutFile
	return h
}

// LayoutFuncs adds the elements of the argument map to the template's layout-only function map.
// It is legal to overwrite elements of the default layout actions:
// - yield func() (template.HTML, error)
// - current  func() (string, error)
// - partial func(partialName string) (template.HTML, error)
// - partial_r func(partialName string) (template.HTML, error)
// - render func(fullPartialName string) (template.HTML, error).
func (h *HTMLAdaptor) LayoutFuncs(funcMap map[string]interface{}) *HTMLAdaptor {
	if funcMap != nil {
		h.engine.Config.LayoutFuncs = funcMap
	}
	return h
}

// Funcs adds the elements of the argument map to the template's function map.
// It is legal to overwrite elements of the default actions:
// - url func(routeName string, args ...string) string
// - urlpath func(routeName string, args ...string) string
// - render func(fullPartialName string) (template.HTML, error).
func (h *HTMLAdaptor) Funcs(funcMap map[string]interface{}) *HTMLAdaptor {
	if len(funcMap) == 0 {
		return h
	}
	// configuration maps are never nil, because
	// they are initialized at each of the engine's New func
	// so we're just passing them inside it.
	for k, v := range funcMap {
		h.engine.Config.Funcs[k] = v
	}

	return h
}
