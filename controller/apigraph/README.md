# API Graph

The `apigraph` controller provides a visual representation of your application's routes using D3.js. It generates an interactive HTML page that displays the routes, methods, and handlers in a tree layout. This visualization helps developers understand the structure and relationships of the routes in their application.

## Features

- Visual representation of routes using D3.js
- Interactive graph with hover and click functionality
- Customizable themes (light and dark)
- Displays route paths, methods, and handlers
- Opens handler files in Visual Studio Code using the `vscode://file/` URL scheme

## Installation

To use the `apigraph` controller, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/controller/apigraph"
```

## Usage

To use the `apigraph` controller, you need to create an implementation of `router.Controller` and register it with your Iris application. You can customize the theme using the `Options` struct.

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/apigraph"
)

func main() {
	app := iris.New()
	app.Router("/apigraph", apigraph.New(apigraph.Options{Theme: "dark"}))

	// Example routes with the same path but different methods:
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("GET /") })
	app.Get("/users", func(ctx iris.Context) { ctx.WriteString("GET /users") })
	app.Get("/users/list", func(ctx iris.Context) { ctx.WriteString("GET /users/list") })
	app.Post("/users", func(ctx iris.Context) { ctx.WriteString("POST /users") })
	app.Get("/users/{id:int}", func(ctx iris.Context) { ctx.WriteString("GET /users/{id:int}") })
	app.Put("/users/{id:int}", func(ctx iris.Context) { ctx.WriteString("PUT /users/{id:int}") })
	app.Get("/users/{id:int}/posts", func(ctx iris.Context) { ctx.WriteString("GET /users/{id:int}/posts") })

	app.Listen(":8080")
}
```

## Options

The `Options` struct configures the `Controller`. It currently has one field:

- `Theme`: The theme to be used for the graph. Available themes are "light" and "dark".

## Themes

The `apigraph` controller supports two themes: light and dark. You can register additional themes using the `RegisterTheme` function.

### Light Theme

The light theme is the default theme. It uses a white background with black text.

### Dark Theme

The dark theme uses a dark background with light text.

### `Theme` Interface

The `Theme` interface represents a theme for the graph.

```go
type Theme interface {
	GetName() string
	GetStyle() string
}
```
