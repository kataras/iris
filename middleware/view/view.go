// Package view registers a template engine so that Render, WithTemplate and
// Context.Render can render its templates. The engines live outside the
// framework: HTML, Django, Handlebars, Jet, Pug and Ace are in
// github.com/iris-contrib/views and Blocks is github.com/kataras/blocks.
// Any type that implements Engine can be registered.
//
// Its entry point is Handler(engine), which returns a Handler that binds the
// engine to the requests that follow it. The engine itself is the single input,
// so it is a plain argument and there is no Options struct.
package view

import (
	"errors"

	"github.com/kataras/iris/v14/context"
)

// ErrNoEngine reports whether a view engine is not available in the current context.
// It's used when trying to render a view without a view engine.
var ErrNoEngine = errors.New("view engine is not available in the current context")

// Render renders a template by its filename with an optional layout and data.
// It writes the response headers and body.
// It returns an error if the engine is not available in the current context.
// If the given "layoutName" is empty then it tries to get the layout from the context's view layout (see `Layout` function).
// If the given "data" is nil then it tries to get the data from the context's view data (see `Data` and `KeyValue` functions).
func Render(ctx *context.Context, templateName, layoutName string, data any) error {
	engine := getEngine(ctx)
	if engine == nil {
		return ErrNoEngine
	}

	if data == nil {
		data = GetData(ctx)
	}

	if layoutName == "" {
		layoutName = getLayout(ctx)
	}

	ctx.SetContentType(context.ContentHTMLHeaderValue) // set html content type even if execute writer fails, because it writes directly to the context.
	err := engine.ExecuteTemplate(ctx, templateName, layoutName, data)
	// if err != nil {
	// ctx.OptionsReadOnly().HandleError(ctx, err) // Already called from ctx.Render if so.
	// if ctx.IsDebug() {
	// 	ctx.Application().Logger().Warn("Template execution failed: " + err.Error())
	// }
	// if ctx.ResponseWriter().Written() <= 0 {
	// 	// ...
	// }
	// }
	return err
}
