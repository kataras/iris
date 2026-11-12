# Rewrite Middleware for Iris

This middleware matches an incoming URL against your rules and rewrites or redirects it. Use it when paths change and old links must keep working, when you want one canonical subdomain, or when short URLs should expand into real routes.

## Features of Iris Rewrite Middleware

The Iris rewrite middleware offers a variety of features to handle URL rewriting efficiently. These features include:

1. **Redirect Match**: Allows defining redirection rules using regular expressions.
2. **Primary Subdomain Redirection**: Automatically redirects root domain requests to a specified primary subdomain.
3. **Customizable Options**: Provides options to customize the rewrite rules and behavior.
4. **Support for Various File Formats**: Supports loading rewrite rules from JSON and YAML files.
5. **Debugging**: Provides logging capabilities for debugging rewrite rules.

## Installation

To use the Iris rewrite middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/rewrite"
```

## Usage

### Basic Setup

To use the rewrite middleware, you need to import the necessary packages and configure the middleware in your Iris application. Here is a basic example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/rewrite"
)

func main() {
	app := iris.New()
	opts := rewrite.Options{
		RedirectMatch: []string{
			"301 /old-path /new-path",
		},
	}
	engine, err := rewrite.New(opts)
	if err != nil {
		panic(err)
	}
	// Or, to panic on a bad rule in one line:
	//	engine := rewrite.Must(opts)
	app.UseRouter(engine.Handler)

	app.Get("/new-path", func(ctx iris.Context) {
		ctx.WriteString("New Path")
	})

	app.Listen(":8080")
}
```

### Redirect Match

The rewrite middleware allows defining redirection rules using regular expressions. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/rewrite"
)

func main() {
	app := iris.New()
	opts := rewrite.Options{
		RedirectMatch: []string{
			"301 /old-path/(.*) /new-path/$1",
		},
	}
	engine, err := rewrite.New(opts)
	if err != nil {
		panic(err)
	}
	app.UseRouter(engine.Handler)

	app.Get("/new-path/{param}", func(ctx iris.Context) {
		param := ctx.Params().Get("param")
		ctx.Writef("New Path with param: %s", param)
	})

	app.Listen(":8080")
}
```

### Primary Subdomain Redirection

The rewrite middleware can automatically redirect root domain requests to a specified primary subdomain. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/rewrite"
)

func main() {
	app := iris.New()
	opts := rewrite.Options{
		PrimarySubdomain: "www",
	}
	engine, err := rewrite.New(opts)
	if err != nil {
		panic(err)
	}
	app.UseRouter(engine.Handler)

	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Root Domain")
	})

	app.Listen(":8080")
}
```

### Loading Rewrite Rules from Files

The rewrite middleware supports loading rewrite rules from JSON and YAML files. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/rewrite"
)

func main() {
	app := iris.New()
	engine := rewrite.Load("redirects.yml")
	app.UseRouter(engine)

	app.Get("/new-path", func(ctx iris.Context) {
		ctx.WriteString("New Path")
	})

	app.Listen(":8080")
}
```

The `redirects.yml` file should contain the rewrite rules in YAML format:

```yaml
RedirectMatch: # REDIRECT_CODE_DIGITS | PATTERN_REGEX | TARGET_REPL
  # Redirects /seo/* to /*
  - 301 /seo/(.*) /$1

  # Redirects /docs/v14* to /docs
  - 301 /docs/v14(.*) /docs

  # Redirects /old(.*) to /
  - 301 /old(.*) /

  # Redirects http or https://test.* to http or https://newtest.*
  - 301 ^(http|https)://test.(.*) $1://newtest.$2

  # Handles /*.json or .xml as *?format=json or xml,
  # WITHOUT redirect. See /users route.
  # When Code is 0 then it does not redirect the request,
  # instead it changes the request URL
  # and leaves a route handle the request.
  - 0 /(.*).(json|xml) /$1?format=$2

# Redirects root domain to www.
# Creation of a www subdomain inside the Application is unnecessary,
# all requests are handled by the root Application itself.
PrimarySubdomain: www
```

### Debugging

The rewrite middleware provides logging capabilities for debugging rewrite rules. Here is an example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/rewrite"

	"github.com/kataras/golog"
)

func main() {
	app := iris.New()
	opts := rewrite.Options{
		RedirectMatch: []string{
			"301 /old-path /new-path",
		},
	}
	engine, err := rewrite.New(opts)
	if err != nil {
		panic(err)
	}

    golog.SetLevel("debug")
	engine.SetLogger(golog.Default)

	app.UseRouter(engine.Handler)

	app.Get("/new-path", func(ctx iris.Context) {
		ctx.WriteString("New Path")
	})

	app.Listen(":8080")
}
```
