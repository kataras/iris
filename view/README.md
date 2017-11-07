# View

Iris supports 5 template engines out-of-the-box, developers can still use any external golang template engine,
as `context/context#ResponseWriter()` is an `io.Writer`.

All of these five template engines have common features with common API,
like Layout, Template Funcs, Party-specific layout, partial rendering and more.

- The standard html, its template parser is the [golang.org/pkg/html/template/](https://golang.org/pkg/html/template/)
- Django, its template parser is the [github.com/flosch/pongo2](https://github.com/flosch/pongo2)
- Pug(Jade), its template parser is the [github.com/Joker/jade](https://github.com/Joker/jade)
- Handlebars, its template parser is the [github.com/aymerick/raymond](https://github.com/aymerick/raymond)
- Amber, its template parser is the [github.com/eknkc/amber](https://github.com/eknkc/amber)

## Overview

```go
// file: main.go
package main

import "github.com/kataras/iris"

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
        ctx.View("hello.html")
    })

    // Method:    GET
    // Resource:  http://localhost:8080/user/42
    app.Get("/user/{id:long}", func(ctx iris.Context) {
        userID, _ := ctx.Params().GetInt64("id")
        ctx.Writef("User ID: %d", userID)
    })

    // Start the server using a network address.
    app.Run(iris.Addr(":8080"))
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

import "github.com/kataras/iris"

func main() {
    app := iris.New()

    // - standard html  | iris.HTML(...)
    // - django         | iris.Django(...)
    // - pug(jade)      | iris.Pug(...)
    // - handlebars     | iris.Handlebars(...)
    // - amber          | iris.Amber(...)
    tmpl := iris.HTML("./templates", ".html")

    // built'n template funcs are:
    //
    // - {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
    // - {{ render "header.html" }}
    // - {{ render_r "header.html" }} // partial relative path to current page
    // - {{ yield }}
    // - {{ current }}

    // register a custom template func.
    tmpl.AddFunc("greet", func(s string) string {
        return "Greetings " + s + "!"
    })

    // register the view engine to the views, this will load the templates.
    app.RegisterView(tmpl)

    app.Get("/", hi)

    // http://localhost:8080
    app.Run(iris.Addr(":8080"))
}

func hi(ctx iris.Context) {
    // render the template file "./templates/hi.html"
    ctx.View("hi.html")
}
```

```html
<!-- file: ./templates/hi.html -->
<b>{{greet "kataras"}}</b> <!-- will be rendered as: <b>Greetings kataras!</b> -->
```

## Embedded

View engine supports bundled(https://github.com/jteeuwen/go-bindata) template files too.
`go-bindata` gives you two functions, `Assset` and `AssetNames`,
these can be setted to each of the template engines using the `.Binary` function.

Example code:

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // $ go get -u github.com/jteeuwen/go-bindata/...
    // $ go-bindata ./templates/...
    // $ go build
    // $ ./embedding-templates-into-app
    // html files are not used, you can delete the folder and run the example
    app.RegisterView(iris.HTML("./templates", ".html").Binary(Asset, AssetNames))
    app.Get("/", hi)

    // http://localhost:8080
    app.Run(iris.Addr(":8080"))
}

type page struct {
    Title, Name string
}

func hi(ctx iris.Context) {
    //                      {{.Page.Title}} and {{Page.Name}}
    ctx.ViewData("Page", page{Title: "Hi Page", Name: "iris"})
    ctx.View("hi.html")
}
```

A real example can be found here: https://github.com/kataras/iris/tree/master/_examples/view/embedding-templates-into-app.

## Reload

Enable auto-reloading of templates on each request. Useful while developers are in dev mode
as they no neeed to restart their app on every template edit.

Example code:

```go
pugEngine := iris.Pug("./templates", ".jade")
pugEngine.Reload(true) // <--- set to true to re-build the templates on each request.
app.RegisterView(pugEngine)
```

## Examples

- [Overview](https://github.com/kataras/iris/blob/master/_examples/view/overview/main.go)
- [Hi](https://github.com/kataras/iris/blob/master/_examples/view/template_html_0/main.go)
- [A simple Layout](https://github.com/kataras/iris/blob/master/_examples/view/template_html_1/main.go)
- [Layouts: `yield` and `render` tmpl funcs](https://github.com/kataras/iris/blob/master/_examples/view/template_html_2/main.go)
- [The `urlpath` tmpl func](https://github.com/kataras/iris/blob/master/_examples/view/template_html_3/main.go)
- [The `url` tmpl func](https://github.com/kataras/iris/blob/master/_examples/view/template_html_4/main.go)
- [Inject Data Between Handlers](https://github.com/kataras/iris/blob/master/_examples/view/context-view-data/main.go)
- [Embedding Templates Into App Executable File](https://github.com/kataras/iris/blob/master/_examples/view/embedding-templates-into-app/main.go)

You can serve [quicktemplate](https://github.com/valyala/quicktemplate) files too, simply by using the `context#ResponseWriter`, take a look at the [iris/_examples/http_responsewriter/quicktemplate](https://github.com/kataras/iris/tree/master/_examples/http_responsewriter/quicktemplate) example.