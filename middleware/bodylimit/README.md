# Request Body Limit Middleware for Iris

This middleware caps the size of an incoming request body. A client that sends more than the limit gets a 413 response instead of your server reading the whole payload into memory. Reach for it on any route that accepts file uploads, JSON, or anything else whose size you do not control.

## Why Use Body Limit Middleware?

### Security

One of the primary reasons to use body limit middleware is to enhance the security of your application. By limiting the size of incoming request bodies, you can mitigate the risk of DoS attacks, where an attacker sends large payloads to exhaust server resources.

### Performance

Large payloads consume memory and processing time. A reasonable limit keeps one oversized request from slowing down every other one.

### Resource Management

In applications that handle file uploads or large JSON payloads, it's essential to manage resources effectively. By setting a body limit, you can prevent clients from uploading excessively large files or sending huge JSON objects that could strain your server's resources.

## Installation

To use the `bodylimit` middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/bodylimit"
```

## Usage

### Basic Setup

To use the body limit middleware, you need to create an Iris application and register the middleware. Below is an example of how to set up the middleware with a limit of 2 MB:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/bodylimit"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()
	app.Use(bodylimit.New(2 * iris.MB)) // set the limit to 2 MB.

	handler := func(ctx iris.Context) {
		body, err := ctx.Body()
		if err != nil {
			// HandleRequestError keeps the classification: a body past the
			// limit stays a 413, anything else is a 400.
			errors.HandleRequestError(ctx, err)
			return
		}

		ctx.Write(body) // write the request body back to the client.
	}

	app.Post("/", handler)
	app.Listen(":8080")
}
```

### Explanation

- **Limit**: The `bodylimit.New` function takes a single parameter, which is the maximum size of the request body in bytes. In the example above, the limit is set to 10 bytes.
- **Handler**: The handler reads the request body and writes it back to the response. If the request body exceeds the limit, the middleware will stop the request and return a `413 Request Entity Too Large` status.
- The body limit middleware uses a `sync.Pool` to manage `Reader` instances, which are used to read the request body and enforce the size limit. This approach ensures efficient memory usage and reduces the overhead of creating new `Reader` instances for each request.

## Testing Handlers with BodyLimit Middleware

To test handlers that use the BodyLimit middleware, you can use the `httptest` package provided by Iris. Here is an example of how to test a handler:

```go
package main_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/bodylimit"
	"github.com/kataras/iris/v14/rest/errors"
)

func TestBodyLimit(t *testing.T) {
	limit := int64(10) // set the limit to 10 bytes for the shake of the test.

	handler := func(ctx iris.Context) {
		body, err := ctx.Body()
		if err != nil {
			// HandleRequestError keeps the classification: a body past the
			// limit stays a 413, anything else is a 400.
			errors.HandleRequestError(ctx, err)
			return
		}
		ctx.Write(body)
	}

	app := iris.New()
	app.Use(bodylimit.New(limit))
	app.Post("/", handler)

	e := httptest.New(t, app)

	// Test with a body that is smaller than the limit.
	e.POST("/").WithText("123456789").Expect().Status(iris.StatusOK).Body().IsEqual("123456789")

	// Test with a body that is equal to the limit.
	e.POST("/").WithText("1234567890").Expect().Status(iris.StatusOK).Body().IsEqual("1234567890")

	// Test with a body that is bigger than the limit.
	e.POST("/").WithText("12345678910").Expect().Status(iris.StatusRequestEntityTooLarge)
}
```
