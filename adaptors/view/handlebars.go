package view

import (
	"github.com/kataras/go-template/handlebars"
)

// HandlebarsAdaptor is the  adaptor for the Handlebars engine.
// Read more about the Handlebars Go Template at:
// https://github.com/aymerick/raymond
// and https://github.com/kataras/go-template/tree/master/handlebars
type HandlebarsAdaptor struct {
	*Adaptor
	engine *handlebars.Engine
}

// Handlebars returns a new kataras/go-template/handlebars template engine
// with the same features as all iris' view engines have:
// Binary assets load (templates inside your executable with .go extension)
// Layout, Funcs, {{ url }} {{ urlpath}} for reverse routing and much more.
//
// Read more: https://github.com/aymerick/raymond
func Handlebars(directory string, extension string) *HandlebarsAdaptor {
	e := handlebars.New()
	return &HandlebarsAdaptor{
		Adaptor: NewAdaptor(directory, extension, e),
		engine:  e,
	}
}

// Layout sets the layout template file which inside should use
// the {{ yield }} func to yield the main template file
// and optionally {{partial/partial_r/render}} to render other template files like headers and footers
//
// The 'tmplLayoutFile' is a relative path of the templates base directory,
// for the template file with its extension.
//
// Example: Handlebars("./templates", ".html").Layout("layouts/mainLayout.html")
//         // mainLayout.html is inside: "./templates/layouts/".
//
// Note: Layout can be changed for a specific call
// action with the option: "layout" on the Iris' context.Render function.
func (h *HandlebarsAdaptor) Layout(tmplLayoutFile string) *HandlebarsAdaptor {
	h.engine.Config.Layout = tmplLayoutFile
	return h
}

// Funcs adds the elements of the argument map to the template's function map.
// It is legal to overwrite elements of the default actions:
// - url func(routeName string, args ...string) string
// - urlpath func(routeName string, args ...string) string
// - and handlebars specific, read more: https://github.com/aymerick/raymond.
func (h *HandlebarsAdaptor) Funcs(funcMap map[string]interface{}) *HandlebarsAdaptor {
	if funcMap == nil {
		return h
	}

	for k, v := range funcMap {
		h.engine.Config.Helpers[k] = v
	}

	return h
}
