# Views

The view middleware renders templates. It works with any engine that implements the `Engine` interface, switches layouts per route or per request, binds data to a template from a handler or from middleware, and lets you register your own template functions.

## Key Features

- **Template Engine Integration**: Supports multiple template engines by implementing the `Engine` interface.
- **Layout Management**: Define and switch layouts dynamically for consistent page structure.
- **Data Binding**: Pass data to your templates via middleware or within handlers.
- **Custom Template Functions**: Extend your templates with custom functions for reusable logic.
- **Middleware Architecture**: Set the engine, the layout and the view data from middleware, so handlers stay short.

## Installation

First, make sure you have the Iris framework installed:

```sh
go get github.com/kataras/iris/v14@latest
```

> Look at https://github.com/iris-contrib/views for more template engines.
> 
> ```sh
> go get github.com/iris-contrib/views@latest
> ```

## Getting Started

### Defining the Engine

The middleware is built around the `Engine` interface, which any template engine must implement to be compatible:

```go
type Engine interface {
    ExecuteTemplate(w io.Writer, templateName string, layoutName string, data any) error
}
```

You can use any template engine that satisfies this interface. For example, using the `blocks` template engine:

```sh
go get github.com/kataras/blocks@latest
```

```go
import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/view"
    "github.com/kataras/iris/v14/rest/errors"

    "github.com/kataras/blocks"
)

func main() {
    app := iris.New()
    templates := blocks.New("./views")
    if err := templates.Load(); err != nil {
        panic(err)
    }

    // Register the view middleware at the router level
    app.UseRouter(view.Handler(templates))

    // Define your routes
    app.Get("/", indexHandler)

    app.Listen(":8080")
}
```

### Rendering Templates

In your handler functions, you can render templates using the `view.Render` function:

```go
func Render(ctx iris.Context, templateName, layoutName string, data any) error
```

```go
func indexHandler(ctx iris.Context) {
    data := iris.Map{
        "Title": "Home Page",
    }

    if err := view.Render(ctx, "index", "main", data); err != nil {
        errors.Internal.LogErr(ctx, err)
        return
    }
}
```

`errors` is the `rest/errors` package imported above. A template error names files, lines and data fields, so `errors.Internal.LogErr` writes it to the server log and the client receives a generic 500. Do not send `err.Error()` to the browser.

Alternatively, you can use the `Template` struct for more structured rendering:

```go
app.Get("/about", view.WithTemplate(&view.Template{
    Name:   "about",
    Layout: "main",
    Data: iris.Map{
        "Title": "About Us",
    },
}))
```

### Managing Layouts

Layouts provide a way to define a common structure for your views, such as headers and footers.

Passing layout directly in the `Render` function:

```go
view.Render(ctx, "index", "layout_name", data)
```

To set a layout for a specific request through the `Layout` function (useful when you want to change layouts dynamically):

```go
view.Layout(ctx, "layout_name")
```

To set a default layout for all requests using middleware:

```go
app.Use(view.WithLayout("default_layout"))
```

### Passing Data to Templates

You can pass data to your templates directly in your handlers or through middleware.

Passing data directly:

```go
data := iris.Map{
    "Title":   "Page Title",
    "Content": "Welcome to our site!",
}
view.Render(ctx, "index", "", data)
```

Passing data within handlers dynamically:

```go
// Set template data
view.Data(ctx, data)

// Add key-value pair
view.KeyValue(ctx, "key", value)
```

Using middleware to set data:

```go
app.Use(view.WithData(iris.Map{
    "AppName": "My Iris App",
}))
```

The map given to `view.WithData` is stored on every request, so all requests share it. `view.KeyValue` does not write into it. The first call of a request copies the map and adds the key to the copy, and `view.GetData(ctx)` returns the result. A key added for one request is never seen by another.

### Adding Custom Template Functions

A template function belongs to the engine, so register it on the engine before `Load`. With `blocks`:

```go
templates := blocks.New("./views").Funcs(template.FuncMap{
    "ToUpper": strings.ToUpper,
    "FormatDate": func(t time.Time) string {
        return t.Format("Jan 2, 2006")
    },
})
if err := templates.Load(); err != nil {
    panic(err)
}

app.UseRouter(view.Handler(templates))
```

A template then calls `{{ ToUpper .Title }}`. The order matters. `html/template` resolves function names while it parses, so `Load` fails with `function "ToUpper" not defined` when the function is not there yet.

The middleware can register functions as well, on an engine that implements the `EngineFuncer` interface:

```go
type EngineFuncer interface {
    AddFunc(funcName string, funcBody any)
}
```

```go
view.AddFunc(ctx, "ToUpper", strings.ToUpper)
```

```go
app.Use(view.WithFuncMap(template.FuncMap{
    "ToUpper": strings.ToUpper,
}))
```

Both call `AddFunc` on the engine that the request holds, and they do it when a request arrives. Two things follow from that:

- The function is not per-request state. Every request and every route that uses the engine can call it from then on. Never pass a function that closes over `ctx`, because it stays on the engine after the request has ended.
- It is too late for an engine that was loaded at startup, as `blocks` is in every example here. It is in time only for an engine that has not loaded its templates yet. The engines of [iris-contrib/views](https://github.com/iris-contrib/views) load on the first render when you did not call `Load` yourself.

A function that depends on the request goes into the view data, like any other value:

```go
view.KeyValue(ctx, "Greet", func(name string) string {
    return ctx.Query().Get("lang") + ": " + name
})
```

The template calls it with `{{ call .Greet .Name }}`.

## Advanced Topics

### Template Engines

The middleware works with any template engine that implements the `Engine` interface. That is the [Strategy Pattern](https://en.wikipedia.org/wiki/Strategy_pattern): swapping engines does not touch your application code.

### Error Handling

`view.Render` returns an error when the template or the layout does not exist, when the engine fails to execute it, and when no engine is registered for the route (`view.ErrNoEngine`). Log it and answer with a generic 500:

```go
if err := view.Render(ctx, "index", "main", data); err != nil {
    errors.Internal.LogErr(ctx, err)
    return
}
```

`view.WithTemplate` needs no such branch. It stops the request and keeps the error private by itself.

### Load Errors

The framework does not load templates for you, so a broken template cannot stop `app.Listen` on its own. `Load` returns the error to your code and you choose what it means. The examples here panic, which is right when a release with a broken page must not start. To keep the rest of the application serving, log it instead:

```go
templates := blocks.New("./views")
if err := templates.Load(); err != nil {
    app.Logger().Errorf("views: %v", err)
}

app.UseRouter(view.Handler(templates))
```

Every other route answers as usual. A route that renders gets the error back from `view.Render` and answers 500 through the branch above. Call `templates.Load()` again once the file is fixed and the same running application renders it, with no restart.

## Example Application

Below is a simple example demonstrating how to set up the view middleware with routes and templates:

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/view"
    "github.com/kataras/iris/v14/rest/errors"

    "github.com/kataras/blocks"
)

func main() {
    app := iris.New()
    templates := blocks.New("./views")
    if err := templates.Load(); err != nil {
        panic(err)
    }

    app.UseRouter(view.Handler(templates))
    app.Use(view.WithLayout("main")) // Set default layout.

    app.Get("/", indexHandler)
    app.Get("/500", errorHandler)

    app.Listen(":8080")
}

func indexHandler(ctx iris.Context) {
    data := iris.Map{
        "Title": "Welcome",
    }

    if err := view.Render(ctx, "index", "", data); err != nil {
        errors.Internal.LogErr(ctx, err)
        return
    }
}

func errorHandler(ctx iris.Context) {
    ctx.StatusCode(iris.StatusInternalServerError)
    data := iris.Map{
        "Code":    iris.StatusInternalServerError,
        "Message": "Something went wrong!",
    }

    if err := view.Render(ctx, "500", "error", data); err != nil {
        errors.Internal.LogErr(ctx, err)
        return
    }
}
```

## Best Practices

- **Modular Design**: Keep your template logic modular by using layouts and partials.
- **Data Separation**: Separate your data preparation from your rendering logic.
- **Error Handling**: Always check for errors when rendering templates.
- **Custom Functions**: Extend your templates with custom functions for reusable logic.

## References

- [Go's `html/template` Package](https://pkg.go.dev/html/template)
- [Blocks Engine](https://github.com/kataras/blocks)
