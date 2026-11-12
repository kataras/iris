# Recovery Middleware

## Overview

The [recovery](https://github.com/kataras/iris/tree/main/middleware/recovery) middleware for the [Iris web framework](https://www.iris-go.com) provides a mechanism to recover from panics that occur during the handling of HTTP requests. It ensures that your application can gracefully handle unexpected errors and continue to operate without crashing. This middleware captures the panic, logs the error details, and sends an appropriate response to the client.

## Features

- **Automatic Panic Recovery**: Recovers from panics and logs the error details.
- **Custom Error Handlers**: Allows the registration of custom error handlers to handle recovery errors in a specific way.
- **Detailed Error Information**: Captures and logs detailed information about the panic, including the stack trace, registered handlers, and the HTTP request.

## Usage

### Installation

To use the Recovery middleware, you need to import it into your Iris application:

```go
import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/recovery"
)
```

### Basic Usage

To enable the Recovery middleware, simply use it in your Iris application:

```go
app := iris.New()
app.Use(recovery.Handler)
// OR (recommended)
// app.UseRouter(recovery.Handler)
```

### Custom Error Handler

You can register a custom error handler to handle recovery errors in a specific way. Use the `WithErrorHandler` function to set a custom error handler.

Render the failure through `github.com/kataras/iris/v14/rest/errors`: `errors.HandleError` sends the generic 500 envelope to the client and the panic itself to the logger and the collector, so a stack trace never reaches the caller. `errors.RecoveryHandler` is this whole middleware ready-made.

```go
app := iris.New()
app.Use(recovery.WithErrorHandler(customErrorHandler), recovery.Handler)

func customErrorHandler(ctx iris.Context, err error) {
	// Custom error handling logic.
	errors.HandleError(ctx, err)
}
```

## Example

Here is a complete example demonstrating the usage of the Recovery middleware:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/recovery"
)

func main() {
	app := iris.New()
	app.Use(recovery.Handler)

	i := 0
	app.Get("/", func(ctx iris.Context) {
		i++
		if i%2 == 0 {
			panic("a panic here")
		}
		ctx.Writef("Hello, refresh one time more to get panic!")
	})

	app.Listen(":8080")
}
```

### Example Output

When a panic occurs, the Recovery middleware captures the panic and logs the error details. Here is an example of the log output:

```
Recovered from a route's Handler('main.main.func1')
GET / HTTP/1.1
Host: localhost:8080
User-Agent: curl/7.64.1
Accept: */*

a panic here
main.main.func1
	/path/to/main.go:15
runtime.gopanic
	/usr/local/go/src/runtime/panic.go:965
```

## Detailed Error Information

The Recovery middleware captures detailed information about the panic, including:

- **Cause**: The cause of the panic.
- **Callers**: The stack trace of the panic.
- **Stack**: The full debug stack.
- **Registered Handlers**: The list of registered handlers.
- **Current Handler File Line**: The file and line number of the handler where the panic occurred.
- **Current Handler Name**: The name of the handler where the panic occurred.
- **Request**: The HTTP request that caused the panic.

## References

- [Panic and Recover in Go](https://blog.golang.org/defer-panic-and-recover)

## API Reference

### recovery.Handler

The `Handler` function returns a new recovery middleware that recovers from panics and logs the panic message to the application's logger at the "Warn" level.

### recovery.WithErrorHandler

The `WithErrorHandler` function returns a middleware that sets a custom recover error handler for the current request.

```go
func WithErrorHandler(errorHandler func(ctx iris.Context, err error)) iris.Handler
```

### recovery.PanicRecoveryError

The `PanicRecoveryError` function returns a new `ErrPanicRecovery` error.

```go
func PanicRecoveryError(ctx iris.Context, err any) *context.ErrPanicRecovery
```

### `context.ErrPanicRecovery`

The `ErrPanicRecovery` struct contains detailed information about the panic.

```go
type ErrPanicRecovery struct {
	ErrPrivate
	Cause                  any
	Callers                []string
	Stack                  []byte
	RegisteredHandlers     []string
	CurrentHandlerFileLine string
	CurrentHandlerName     string
	Request                string
}
```

### Context

The `iris.Context` struct provides a method to interact with panics.

```go
IsRecovered() (*ErrPanicRecovery, bool)
```
