package view

import (
	"github.com/kataras/iris/v14/context"
)

// Template represents a view.
// It's used to render a view with a specific status code, layout and data.
// Usage:
//
//	ctx.Render(&Template{Name: "index.html", Layout: "layouts/main.html", Data: someData})
type Template struct { // [T any] doesn't work if want it to run as initialization type.
	Code   int    `json:"code" yaml:"Code"`
	Layout string `json:"layout" yaml:"Layout"`
	Name   string `json:"name" yaml:"Name"`
	Data   any    `json:"data" yaml:"Data"`

	// DataFunc can be used in order to return some `Data` based on the iris Context,
	// from outside the context. E.g. when declaring the View outside of the handler scope.
	//
	// When DataFunc is not nil its result is rendered and the Data field is
	// ignored. The result belongs to the request: Render does not store it in
	// the Template, because WithTemplate gives the same Template to every
	// request of a route.
	DataFunc func(*context.Context) (any, error) `json:"-" yaml:"-"`
}

// Render renders the template.
// It writes the response headers and body.
// It completes the `Renderer` interface.
//
// Render does not change the Template, so one Template can render many
// requests at the same time.
func (t *Template) Render(ctx *context.Context) error {
	if t.Code > 0 {
		ctx.StatusCode(t.Code)
	}

	data := t.Data
	if t.DataFunc != nil {
		var err error
		data, err = t.DataFunc(ctx)
		if err != nil {
			return err
		}
	}

	return Render(ctx, t.Name, t.Layout, data)
}

// WithTemplate returns a Handler that renders the given template.
func WithTemplate(t *Template) context.Handler {
	return func(ctx *context.Context) {
		if err := ctx.Render(t); err != nil {
			// Private: a template engine error names files, lines and data fields.
			ctx.StopWithPlainError(t.Code, context.PrivateError(err))
			return
		}

		if ctx.HandlerIndex(-1) < len(ctx.Handlers())-1 {
			ctx.Next()
		}
	}
}
