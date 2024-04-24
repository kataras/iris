# View

Iris supports 7 template engines out-of-the-box, developers can still use any external golang template engine,
as `Context.ResponseWriter()` is an `io.Writer`.

All template engines share a common API i.e.
Parse using embedded assets, Layouts and Party-specific layout, Template Funcs, Partial Render and more.

| #  | Name       | Parser   |
|:---|:-----------|----------|
| 1 | HTML       | [html/template](https://pkg.go.dev/html/template) |
| 2 | Blocks     | [kataras/blocks](https://github.com/kataras/blocks) |
| 3 | Django     | [flosch/pongo2](https://github.com/flosch/pongo2) |
| 4 | Pug        | [Joker/jade](https://github.com/Joker/jade) |
| 5 | Handlebars | [mailgun/raymond](https://github.com/mailgun/raymond) |
| 6 | Jet        | [CloudyKit/jet](https://github.com/CloudyKit/jet) |
| 7 | Ace        | [yosssi/ace](https://github.com/yosssi/ace) |

[List of Examples](https://github.com/kataras/iris/tree/main/_examples/view).

[Benchmarks](https://github.com/kataras/iris/tree/main/_benchmarks/view).

You can serve [quicktemplate](https://github.com/valyala/quicktemplate) files too, simply by using the `Context.ResponseWriter`, take a look at the [iris/_examples/view/quicktemplate](https://github.com/kataras/iris/tree/main/_examples/view/quicktemplate) example.

## Overview

```go
// file: main.go
package main

import "github.com/kataras/iris/v12"

func main() {
    app := iris.New()
    // Load all templates from the "./views" folder
    // where extension is ".html" and parse them
    // using the standard `html/template` package.
    app.RegisterView(iris.HTML("./views", ".html"))

    // Method:    GET
    // Resource:  http://localhost:8080
    app.Get("/", func(ctx iris.Context) {
        // Bind: {{.message}} with "Hello world!"
        ctx.ViewData("message", "Hello world!")
        // Render template file: ./views/hello.html
        if err := ctx.View("hello.html"); err != nil {
		    ctx.HTML("<h3>%s</h3>", err.Error())
		    return
	    }
    })

    // Method:    GET
    // Resource:  http://localhost:8080/user/42
    app.Get("/user/{id:int64}", func(ctx iris.Context) {
        userID, _ := ctx.Params().GetInt64("id")
        ctx.Writef("User ID: %d", userID)
    })

    // Start the server using a network address.
    app.Listen(":8080")
}
```

```html
<!-- file: ./views/hello.html -->
<html>
<head>
    <title>Hello Page</title>
</head>
<body>
    <h1>{{.message}}</h1>
</body>
</html>
```

## Template functions

```go
package main

import "github.com/kataras/iris/v12"

func main() {
    app := iris.New()
    tmpl := iris.HTML("./templates", ".html")

    // builtin template funcs are:
    //
    // - {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
    // - {{ render "header.html" . }}
    // - {{ render_r "header.html" . }} // partial relative path to current page
    // - {{ yield . }}
    // - {{ current . }}

    // register a custom template func.
    tmpl.AddFunc("greet", func(s string) string {
        return "Greetings " + s + "!"
    })

    // register the view engine to the views, this will load the templates.
    app.RegisterView(tmpl)

    app.Get("/", hi)

    // http://localhost:8080
    app.Listen(":8080")
}

func hi(ctx iris.Context) {
    // render the template file "./templates/hi.html"
    if err := ctx.View("hi.html"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
```

```html
<!-- file: ./templates/hi.html -->
<b>{{greet "kataras"}}</b> <!-- will be rendered as: <b>Greetings kataras!</b> -->
```

## Embedded

View engine supports bundled(https://github.com/go-bindata/go-bindata) template files too. Latest
`go-bindata` release gives you a compatible `http.FileSystem` that can be provided as the first argument of a view engine's initialization, e.g. `HTML(AssetFile(), ".html")`.


```sh
$ go install github.com/go-bindata/go-bindata/v3/go-bindata@latest
$ go-bindata -fs -prefix "templates" ./templates/...
$ go run .
```

Example Code:

```go
package main

import "github.com/kataras/iris/v12"

func main() {
    app := iris.New()
    app.RegisterView(iris.HTML(AssetFile(), ".html"))
    app.Get("/", hi)

    // http://localhost:8080
    app.Listen(":8080")
}

type page struct {
    Title, Name string
}

func hi(ctx iris.Context) {
    //                      {{.Page.Title}} and {{Page.Name}}
    ctx.ViewData("Page", page{Title: "Hi Page", Name: "iris"})
    if err := ctx.View("hi.html"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
```

Examples can be found here: https://github.com/kataras/iris/tree/main/_examples/view/embedding-templates-into-app and  https://github.com/kataras/iris/tree/main/_examples/view/embedding-templates-into-app-bindata.

## Reload

Enable auto-reloading of templates on each request. Useful while developers are in dev mode
as they no neeed to restart their app on every template edit.

Example code:

```go
pugEngine := iris.Pug("./templates", ".jade")
pugEngine.Reload(true) // <--- set to true to re-build the templates on each request.
app.RegisterView(pugEngine)
```