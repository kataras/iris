# HTTP Method Override

Some clients can only send GET and POST. An HTML form is the common case: it cannot issue a PUT or a DELETE. This middleware reads the intended method from a header, a form field, or a query parameter, and rewrites the request method before routing, so one handler registered on `PUT` serves both a real PUT and a POST that asked for one.

## Features of Iris Method Override Middleware

The Iris method override middleware offers a variety of features to handle HTTP method overriding efficiently. These features include:

1. **Customizable Methods**: Define which HTTP methods can be overridden.
2. **Custom Headers**: Specify custom headers to determine the method override.
3. **Form Field Support**: Use form fields to specify the method override.
4. **Query Parameter Support**: Use query parameters to specify the method override.
5. **Save Original Method**: Optionally save the original HTTP method for later use.

## Installation

To use the Iris method override middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/methodoverride"
```

## Usage

### Basic Setup

To use the method override middleware, you need to import the necessary packages and configure the middleware in your Iris application. Here is a basic example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/methodoverride"
)

func main() {
	app := iris.New()

	// Create the method override middleware
	mo := methodoverride.New(methodoverride.Options{
		SaveOriginalMethodContextKey: "_originalMethod",
	})

	// Register the middleware with UseRouter
	app.UseRouter(mo)

	app.Post("/path", func(ctx iris.Context) {
		ctx.WriteString("POST response")
	})

	app.Delete("/path", func(ctx iris.Context) {
		ctx.WriteString("DELETE response")
	})

	app.Listen(":8080")
}
```

### Customizable Methods

You can define which HTTP methods can be overridden using the `Methods` option. By default, only the POST method can be overridden. Here is an example:

```go
mo := methodoverride.New(methodoverride.Options{
	Methods: []string{"POST", "PUT"},
})
```

### Custom Headers

You can specify custom headers to determine the method override using the  `Headers` option. By default, the middleware checks the following headers: `X-HTTP-Method`, `X-HTTP-Method-Override`, and `X-Method-Override`. Here is an example:

```go
mo := methodoverride.New(methodoverride.Options{
	Headers: []string{"X-Custom-Method"},
})
```

### Form Field Support

You can use form fields to specify the method override using the `FormField` option. By default, the middleware checks the `_method` form field. Here is an example:

```go
mo := methodoverride.New(methodoverride.Options{
	FormField: "_method",
})
```

### Query Parameter Support

You can use query parameters to specify the method override using the `Query` option. By default, the middleware checks the `_method` query parameter. Here is an example:

```go
mo := methodoverride.New(methodoverride.Options{
	Query: "_method",
})
```

### Save Original Method

You can optionally save the original HTTP method for later use with `SaveOriginalMethodContextKey`. Here is an example:

```go
mo := methodoverride.New(methodoverride.Options{
	SaveOriginalMethodContextKey: "_originalMethod",
})
```

### Headers Only (recommended)

A cross-site form POST needs no preflight and carries the victim's cookies, so the default `_method` query and form lookups let a hostile page reach a `DELETE`-only route. Set `HeadersOnly` to accept the override from a request header alone, which a browser cannot set cross-site without a preflight:

```go
mo := methodoverride.New(methodoverride.Options{
	HeadersOnly: true,
})
```

### Example

Here is a complete example demonstrating how to use the method override middleware with an Iris application:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/methodoverride"
)

func main() {
	app := iris.New()

	// Create the method override middleware
	mo := methodoverride.New(methodoverride.Options{
		SaveOriginalMethodContextKey: "_originalMethod",
	})

	// Register the middleware with UseRouter
	app.UseRouter(mo)

	app.Post("/path", func(ctx iris.Context) {
		ctx.WriteString("POST response")
	})

	app.Delete("/path", func(ctx iris.Context) {
		ctx.WriteString("DELETE response")
	})

	app.Listen(":8080")
}
```
