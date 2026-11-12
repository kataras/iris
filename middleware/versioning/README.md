# API Versioning

This middleware routes a request to the handler that matches the API version the client asked for, so several versions of the same endpoint can live side by side.

## What is API Versioning?

API versioning lets you change an API without breaking the clients already using it. Existing clients keep calling the version they were built against while you ship the next one.

For more information, you can refer to the [API versioning Wikipedia page](https://en.wikipedia.org/wiki/API_versioning).

## How to use the Iris Versioning Middleware?

To use the Iris API versioning middleware, we need to do the following:

- Import the `github.com/kataras/iris/v14/middleware/versioning` package, which provides an Iris middleware for versioning.
- Use the `versioning.New` function to create an `iris.Handler` and register it in our Iris app.

First, we need to install the Iris Web Framework, which contains the middleware too:

```sh
go get github.com/kataras/iris/v14@latest
```

Then, we need to import it in our `main.go` file:

```go
import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/versioning"
    "github.com/kataras/iris/v14/rest/errors"
)
```

Next, we need to use the `versioning.New` function to create an 

`iris.Handler`. The function takes an `Options` struct that allows us to configure the versioning middleware. The `Options` struct includes fields such as `Default`, `Latest`, `Strict`, `Versions`, `Deprecation`, `Header`, `QueryParam`, `Subdomain`, `NotFoundHandler`, and `Skipper`. We can use the following code to create the handler:

```go
opts := versioning.Options{
    Default: "1.0.0",
    Latest:  "2.0.0",
    Strict:  false,
    Header:  "Accept-Version",
    QueryParam: "v",
    Subdomain: false,
    NotFoundHandler: errors.DefaultVersioningNotFoundHandler,
    Deprecation: versioning.DeprecationOptions{
        WarnMessage:     "WARNING! You are using a deprecated version of this API.",
        DeprecationDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
        DeprecationInfo: "A newer version is available.",
    },
    Versions: versioning.Versions{
        {
            Range:      ">=1.0.0 <2.0.0",
            Path:       "/v1",
            Deprecated: true,
        },
        {
            Range: ">=2.0.0",
            Path:  "/v2",
        },
    },
}

versioningMiddleware := versioning.New(opts)
```

Finally, we need to register the versioning middleware in our Iris app. We can use the `app.UseRouter` method to apply the middleware on every route. For example:

```go
// Create an Iris app instance.
app := iris.New()

// Apply the versioning middleware.
app.UseRouter(versioningMiddleware)

// Define versioned routes.
v1 := app.Router("/v1")
{
    v1.Get("/", func(ctx iris.Context) {
        ctx.WriteString("Hello from version 1")
    })
}

v2 := app.Router("/v2")
{
    v2.Get("/", func(ctx iris.Context) {
        ctx.WriteString("Hello from version 2")
    })
}

// Start the server.
app.Listen(":8080")
```

## Example

Here is a complete example of how to use versioning middleware with Iris:

```go
package main

import (
    "time"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/versioning"
    "github.com/kataras/iris/v14/rest/errors"
)

func main() {
    app := iris.New()

    opts := versioning.Options{
        Default: "1.0.0",
        Latest:  "2.0.0",
        Strict:  false,
        Header:  "Accept-Version",
        QueryParam: "v",
        Subdomain: false,
        NotFoundHandler: errors.DefaultVersioningNotFoundHandler,
        Deprecation: versioning.DeprecationOptions{
            WarnMessage:     "WARNING! You are using a deprecated version of this API.",
            DeprecationDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
            DeprecationInfo: "A newer version is available.",
        },
        Versions: versioning.Versions{
            {
                Range:      ">=1.0.0 <2.0.0",
                Path:       "/v1",
                Deprecated: true,
            },
            {
                Range: ">=2.0.0",
                Path:  "/v2",
            },
        },
    }

    versioningMiddleware := versioning.New(opts)
    app.UseRouter(versioningMiddleware)

    v1 := app.Router("/v1")
    {
        v1.Get("/", func(ctx iris.Context) {
            ctx.WriteString("Hello from version 1")
        })
    }

    v2 := app.Router("/v2")
    {
        v2.Get("/", func(ctx iris.Context) {
            ctx.WriteString("Hello from version 2")
        })
    }

    app.Listen(":8080")
}
```
