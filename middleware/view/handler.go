package view

import (
	"fmt"
	"maps"
	"text/template"

	"github.com/kataras/iris/v14/context"
)

func init() {
	context.SetHandlerName("iris/middleware/view.*Handler", "iris.view")
}

const (
	viewEngineContextKey = "iris.view.engine"
	viewLayoutContextKey = "iris.view.layout"
	viewDataContextKey   = "iris.view.data"
)

// Handler returns a context.Handler that stores the view engine to the context,
// by overriding any previous ones for the handler (chain).
func Handler(engine Engine) context.Handler {
	return func(ctx *context.Context) {
		if engine == nil {
			ctx.Values().Remove(viewEngineContextKey)
		} else {
			ctx.Values().Set(viewEngineContextKey, engine)
		}

		ctx.Next()
	}
}

// getEngine returns the view engine from the context.
func getEngine(ctx *context.Context) Engine {
	engine, ok := ctx.ValueAs[Engine](viewEngineContextKey)
	if !ok {
		return nil
	}

	return engine
}

// Ext returns the final file extension (including the dot) which the view engine is responsible to render.
// It returns an empty string if the engine is not available in the current context.
func Ext(ctx *context.Context) string {
	engine, ok := ctx.ValueAs[EngineExtenser](viewEngineContextKey)
	if !ok {
		return ""
	}

	return engine.Ext()
}

// WithFuncMap returns a context.Handler that adds the given template.FuncMap
// to the view engine of the request, by calling AddFunc for each entry.
//
// The functions are registered on the engine and not on the request, so read
// AddFunc before you use it. The handler adds the same functions again on
// every request, which only helps an engine that has not loaded its templates
// yet. To register functions once, call the engine's own method before its
// Load.
func WithFuncMap(funcMap template.FuncMap) context.Handler {
	return func(ctx *context.Context) {
		for funcName, fn := range funcMap {
			AddFunc(ctx, funcName, fn)
		}

		ctx.Next()
	}
}

// AddFunc adds a template function to the view engine that the request holds.
// It reports whether that engine takes functions, which means that it
// implements EngineFuncer.
//
// The function is registered on the engine, so it is not per-request state:
//
//   - Every request and every route that uses the same engine can call it
//     from then on.
//   - Do not pass a function that closes over ctx. It stays on the engine
//     after the request has ended, while the Context it captured goes back to
//     the pool and serves other requests. A value that belongs to one request
//     goes into the view data (see Data and KeyValue), a function included: a
//     template calls it with {{ call .Name arguments }}.
//   - When the function becomes callable depends on the engine. One that
//     resolves function names while it parses, as html/template does, needs
//     the function before it loads its templates, and an engine that is
//     loaded at startup has done that before the first request arrives. For
//     such an engine call its own AddFunc or Funcs method before Load.
//
// Example:
//
//	rv := router.NewRoutePathReverser(app.RouterBuilder.Routes())
//	view.AddFunc(ctx, "urlpath", rv.Path)
func AddFunc(ctx *context.Context, funcName string, fn any) bool {
	engine, ok := ctx.ValueAs[EngineFuncer](viewEngineContextKey)
	if !ok {
		return false
	}

	engine.AddFunc(funcName, fn)
	return true
}

// WithData returns a context.Handler that stores the view data to the context,
// by overriding any previous ones for the handler (chain).
//
// Every request that passes through the handler gets the same value, so do
// not change it once the server runs. KeyValue does not: it copies a map
// before the first write of a request.
func WithData(data any) context.Handler {
	return func(ctx *context.Context) {
		Data(ctx, data)
		ctx.Next()
	}
}

// Data sets the view data to the context, by overriding any previous data.
func Data(ctx *context.Context, data any) {
	if data == nil {
		ctx.Values().Remove(viewDataContextKey)
	} else {
		ctx.Values().Set(viewDataContextKey, data)
	}
}

// requestData is the view data that KeyValue created or copied for the current
// request. The type is the mark of that: a map stored under any other type came
// from the caller, through Data or WithData, and other requests may hold it too.
// Data stores the value it is given, so it removes the mark by overwriting.
type requestData map[string]any

// KeyValue adds a key-value pair to the view data of the current request.
// The view data is created as a Map if it doesn't exist.
// If the view data is not a Map, it returns an error.
//
// KeyValue never writes into a map that was given to Data or WithData,
// because WithData stores the same map on every request. The first call of a
// request copies such a map and writes to the copy, and the calls that follow
// reuse it. The map of the caller stays as it was, so read the result with
// GetData. The copy is shallow: a value that is itself a map or a slice is
// still shared.
func KeyValue(ctx *context.Context, key string, value any) error {
	data, _ := ctx.Values().Lookup(viewDataContextKey)

	switch m := data.(type) {
	case nil:
		ctx.Values().Set(viewDataContextKey, requestData{key: value})
	case requestData:
		m[key] = value
	case map[string]any:
		own := make(requestData, len(m)+1)
		maps.Copy(own, m)
		own[key] = value
		ctx.Values().Set(viewDataContextKey, own)
	default:
		return fmt.Errorf("expected data to be of type Map, found: %T", data)
	}

	return nil
}

// GetData returns the view data from the context.
func GetData(ctx *context.Context) any {
	data, ok := ctx.Values().Lookup(viewDataContextKey)
	if !ok {
		return nil
	}

	if own, ok := data.(requestData); ok {
		// Engines and callers assert on the plain map type.
		return map[string]any(own)
	}

	return data
}

// WithLayout returns a context.Handler that stores the view layout to the context,
// by overriding any previous ones for the handler (chain).
func WithLayout(layout string) context.Handler {
	return func(ctx *context.Context) {
		Layout(ctx, layout)
		ctx.Next()
	}
}

// Layout sets the view layout to the context, by overriding any previous layout.
func Layout(ctx *context.Context, layout string) {
	if layout == "" {
		ctx.Values().Remove(viewLayoutContextKey)
	} else {
		ctx.Values().Set(viewLayoutContextKey, layout)
	}
}

// getLayout returns the view layout from the context.
func getLayout(ctx *context.Context) string {
	return ctx.Values().GetString(viewLayoutContextKey)
}
