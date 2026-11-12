# Request ID Middleware

This middleware gives every request a unique ID and puts it in the `X-Request-Id` response header. When a client sends that header, its value is kept instead. The ID is what lets you follow one request through your logs, and across service boundaries when several services pass it along.

## Features

- **Automatic Request ID Generation**: Generates a unique request ID for each incoming request if not provided by the client.
- **Customizable ID Generation**: Allows custom ID generation logic.
- **Header Management**: Automatically sets the request ID in the response headers (`X-Request-Id`).
- **Integration with Iris Context**: Reads the request ID straight off the Iris context.

## Installation

To use the `requestid` middleware, you need to import it in your Iris application:

```go
import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/requestid"
)
```

## Usage

### Basic Setup

Here is a basic example of how to use the `requestid` middleware in an Iris application:

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/requestid"
)

func main() {
    app := iris.New()
    app.Use(requestid.New())

    app.Get("/", func(ctx iris.Context) {
        ctx.WriteString(requestid.Get(ctx))
    })

    app.Listen(":8080")
}
```

### Custom ID Generation

You can provide a custom ID generation function to the middleware. This function can implement any logic to generate the request ID.

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/requestid"
)

func main() {
    app := iris.New()

    customGen := func(ctx iris.Context) string {
        return "custom-id"
    }

    app.Use(requestid.New(customGen))

    app.Get("/", func(ctx iris.Context) {
        ctx.WriteString(requestid.Get(ctx))
    })

    app.Listen(":8080")
}
```

### Hash-Based ID Generation

The middleware also provides a hash-based ID generator that uses the request's hash to generate a fixed-length request ID. This can be useful for generating consistent IDs for similar requests.

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/requestid"
)

func main() {
    app := iris.New()
    // True to include the request's body hash in the ID.
    app.Use(requestid.New(requestid.HashGenerator(true)))

    app.Get("/", func(ctx iris.Context) {
        ctx.WriteString(requestid.Get(ctx))
    })

    app.Listen(":8080")
}
```

## Server-Generated IDs (v14)

`DefaultGenerator` reuses a client-supplied `X-Request-ID` header when present. If request IDs end up in logs and clients should not be able to choose or forge them (log injection), use `ServerOnlyGenerator`, which always generates a new server-side UUID and ignores the inbound header:

```go
app.Use(requestid.New(requestid.ServerOnlyGenerator))
```
