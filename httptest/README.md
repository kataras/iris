# HTTP End-To-End Testing

## Overview

The `httptest` subpackage provides utilities for testing HTTP servers and handlers in the Iris web framework. It includes tools for creating test servers, making HTTP requests, and validating responses. This subpackage is designed to simplify the process of writing and running tests for your Iris applications.

## Features

- **HTTP Status Codes**: Constants for HTTP status codes as registered with IANA.
- **Test Server**: Utilities for creating and managing test servers.
- **TLS Support**: Functions for creating TLS listeners and certificates.
- **Request and Response Handling**: Tools for making HTTP requests and validating responses.
- **Configuration Options**: Customizable options for configuring the test environment.
- **Integration with httpexpect**: Assertions come from the [httpexpect](https://github.com/gavv/httpexpect) package, so tests read as chained expectations.

## Installation

To use the `httptest` subpackage, you need to have the Iris framework installed. You can install Iris using the following command:

```sh
go get github.com/kataras/iris/v14@latest
```

## Creating a Test Server

The `httptest` subpackage provides utilities for creating and managing test servers. The `NewServer` function creates a new test server for an Iris application. It is rarely required for your tests but is available when you need to migrate tests from the standard Go's `net/http/httptest` package.

```go
package main

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

func TestMyApp(t *testing.T) {
	app := iris.New()
	// Configure your app...

	server := httptest.NewServer(t, app)
	defer server.Close()

    // Write your tests using the server...
}
```

### Testing a Single Controller

`NewController` builds a throwaway application with one controller registered under a prefix and returns the same `*httpexpect.Expect` that `New` does, so a `Routing` value can be tested on its own, without the application or the builder chain the production code goes through. A failure answers what an application built with `iris.NewBuilder` answers, so the test asserts the body production sends: the path parameter type and context error handlers are installed, and `rest/errors.Handler` renders a bare failure status (`ctx.StopWithStatus(403)`) and an unknown path (404) as the `rest/errors` JSON envelope. A method the controller does not serve answers that 404 envelope too, not a 405: the option that turns 405 on (`Options.Router.FireMethodNotAllowed`) is off by default and `NewController` takes no options. The variadic middleware is registered on the controller's own group. Nothing else the builder chain adds is there: no CORS, no access log, no recovery, no compression. It takes no `OptionSetter`, because Go allows one variadic parameter and the middleware is the one a controller test needs; a test which needs `httptest.URL` or a non-default log level builds the application itself and calls `New`.

```go
func NewController(t IrisTesty, prefix string, controller any, mw ...context.Handler) *httpexpect.Expect
```

```go
func TestPlans(t *testing.T) {
	e := httptest.NewController(t, "/plans", &api.PlansController{Plans: repo})

	e.GET("/plans/1").Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("name", "starter")
}
```

## Making HTTP Requests

The `httptest` subpackage integrates with the [httpexpect](https://github.com/gavv/httpexpect) package to provide a fluent API for making HTTP requests and validating responses.

```go
package main

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

func TestMyHandler(t *testing.T) {
	app := iris.New()
	// Configure your app...

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("Hello, World!")
}
```

## TLS Support

The `httptest` subpackage provides functions for creating TLS listeners and certificates. The `NewLocalTLSListener` function creates a new TLS listener with a self-signed certificate.

```go
package main

import (
	"testing"

	"github.com/kataras/iris/v14/httptest"
)

func TestTLS(t *testing.T) {
	listener := httptest.NewLocalTLSListener(nil)
	defer listener.Close()

	// Use the listener in your tests...
}
```

## Configuration Options

Configuration options set up the test environment: the base URL, debug mode, and the log level.

```go
package main

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

func TestWithOptions(t *testing.T) {
	app := iris.New()
	// Configure your app...

	e := httptest.New(t, app, httptest.URL("http://localhost:8080"), httptest.Debug(true), httptest.LogLevel("debug"))
	e.GET("/").Expect().Status(httptest.StatusOK)
}
```

## Examples

### Basic Example

```go
package main

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

func TestBasic(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("Hello, World!")
}
```

### Testing with Basic Authentication

```go
package main

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/basicauth"
)

func TestBasicAuth(t *testing.T) {
	app := iris.New()
	auth := basicauth.Default(map[string]string{"user": "pass"})
	app.Use(auth)
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Hello, Authenticated User!")
	})

	e := httptest.New(t, app)
	e.GET("/").WithBasicAuth("user", "pass").Expect().Status(httptest.StatusOK).Body().IsEqual("Hello, Authenticated User!")
	e.GET("/").Expect().Status(httptest.StatusUnauthorized)
}
```

### Testing with JSON Requests and Responses

```go
package main

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestJSON(t *testing.T) {
	app := iris.New()
	app.Post("/user", func(ctx iris.Context) {
		user, ok := ctx.BindJSON[User]()
		if !ok {
			return // ctx.BindJSON wrote the 400 itself.
		}
		ctx.OK(user, nil)
	})

	e := httptest.New(t, app)
	e.POST("/user").WithJSON(User{Name: "Gerasimos", Age: 31}).Expect().Status(httptest.StatusOK).JSON().Object().ValueEqual("name", "Gerasimos").ValueEqual("age", 31)
}
```

```sh
$ go test -v
```

For more examples, read the `_test.go` files across the [Iris repository](https://github.com/kataras/iris).
