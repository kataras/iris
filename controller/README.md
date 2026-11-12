# Controller

A controller groups related routes into one type: shared middleware in one place, request handling next to the state it needs. Implement the `Routing(r iris.Router)` method to build one, then register it with `app.Router`. Example of a controller:

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/controller/fileserver"
)

func main() {
    app := iris.New()

    // Register the built-in fileserver controller.
	app.Router("/static", fileserver.New("./assets", fileserver.Options{
		IndexName: "index.html",
		ShowList:  true,
	}))

    // Register a custom controller.
    app.Router("/users", new(UserController))

    app.Listen(":8080")
}

type UserController struct{}

func (c *UserController) Routing(r iris.Router) {
    r.Get("/", c.List)
    r.Post("/", c.Create)
    r.Get("/{id:int}", c.Get)
    r.Put("/{id:int}", c.Update)
    r.Delete("/{id:int}", c.Delete)
}

func (c *UserController) List(ctx iris.Context) {
    ctx.WriteString("List users")
}

func (c *UserController) Create(ctx iris.Context) {
    ctx.WriteString("Create user")
}

func (c *UserController) Get(ctx iris.Context) {
    id := ctx.Params().GetInt("id")
    ctx.Writef("Get user %d", id)
}

func (c *UserController) Update(ctx iris.Context) {
    id := ctx.Params().GetInt("id")
    ctx.Writef("Update user %d", id)
}

func (c *UserController) Delete(ctx iris.Context) {
    id := ctx.Params().GetInt("id")
    ctx.Writef("Delete user %d", id)
}
```


## Built-in Controllers

| Controller | Description | Documentation |
| -----------|--------|-------------|
| [API Graph](apigraph) | The **API Graph** router Controller registers a handler that renders a graph of the application's routes. It generates an HTML page with a D3.js visualization of the routes, showing the paths, methods, and handlers. The graph is interactive: click a handler to open its file in Visual Studio Code, hover over a link or a node for more details. | [apigraph/README](https://github.com/kataras/iris/blob/main/controller/apigraph/README.md) |
| [File Server](fileserver) | The **File Server** router controller and handler serves static files from a specified directory. It is useful for delivering assets like images, CSS, JavaScript files, and downloadable content. | [fileserver/README](https://github.com/kataras/iris/blob/main/controller/fileserver/README.md) |
| [MVC](mvc) | The **MVC** router controller supports the Model-View-Controller design pattern: the application is split into separate layers, which are easier to maintain and test. | [mvc/README](https://github.com/kataras/iris/blob/main/controller/mvc/README.md) |
| [Sitemap](sitemap) | The **Sitemap** router Controller generates XML sitemaps for your application automatically. This helps search engines crawl and index your website more effectively. | [sitemap/README](https://github.com/kataras/iris/blob/main/controller/sitemap/README.md) |
