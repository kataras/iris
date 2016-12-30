# httpexpect [![GoDoc](https://godoc.org/github.com/gavv/httpexpect?status.svg)](https://godoc.org/github.com/gavv/httpexpect) [![Gitter](https://badges.gitter.im/gavv/httpexpect.svg)](https://gitter.im/gavv/httpexpect?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge) [![Travis](https://img.shields.io/travis/gavv/httpexpect.svg)](https://travis-ci.org/gavv/httpexpect) [![Coveralls](https://coveralls.io/repos/github/gavv/httpexpect/badge.svg?branch=master)](https://coveralls.io/github/gavv/httpexpect?branch=master)

Concise, declarative, and easy to use end-to-end HTTP and REST API testing for Go (golang).

Basically, httpexpect is a set of chainable *builders* for HTTP requests and *assertions* for HTTP responses and payload, on top of net/http and several utility packages.

Workflow:

* Incrementally build HTTP requests.
* Inspect HTTP responses.
* Inspect response payload recursively.

## Features

##### Request builder

* URL path construction, with simple string interpolation provided by [`go-interpol`](https://github.com/imkira/go-interpol) package.
* URL query parameters (encoding using [`go-querystring`](https://github.com/google/go-querystring) package).
* Headers, cookies, payload: JSON,  urlencoded or multipart forms (encoding using [`form`](https://github.com/ajg/form) package), plain text.
* Create custom [request builders](#reusable-builders) that can be reused.

##### Response assertions

* Response status, predefined status ranges.
* Headers, cookies, payload: JSON, JSONP, forms, text.
* Round-trip time.

##### Payload assertions

* Type-specific assertions, supported types: object, array, string, number, boolean, null, datetime.
* Regular expressions.
* Simple JSON queries (using subset of [JSONPath](http://goessner.net/articles/JsonPath/)), provided by [`jsonpath`](https://github.com/yalp/jsonpath) package.
* [JSON Schema](http://json-schema.org/) validation, provided by [`gojsonschema`](https://github.com/xeipuuv/gojsonschema) package.

##### Pretty printing

* Verbose error messages.
* JSON diff is produced on failure using [`gojsondiff`](https://github.com/yudai/gojsondiff/) package.
* Failures are reported using [`testify`](https://github.com/stretchr/testify/) (`assert` or `require` package) or standard `testing` package.
* Dumping requests and responses in various formats, using [`httputil`](https://golang.org/pkg/net/http/httputil/), [`http2curl`](https://github.com/moul/http2curl), or simple compact logger.

##### Tuning

* Tests can communicate with server via HTTP client or invoke HTTP handler (Go function) directly.
* Custom request factory can be provided, e.g. from the Google App Engine testing.
* Integration with [`fasthttp`](https://github.com/valyala/fasthttp/) HTTP handler is available too.
* Custom HTTP client, logger, and failure reporter may be provided by user.

## Status

Stable branches are available on [`gopkg.in`](http://labix.org/gopkg.in) and will not introduce backward-incompatible changes.

Current stable branch is [`v1`](http://gopkg.in/gavv/httpexpect.v1):

```go
import "gopkg.in/gavv/httpexpect.v1"
```

Development is done in `master` branch on github:

```go
import "github.com/gavv/httpexpect"
```

## Documentation

Documentation is available on [GoDoc](https://godoc.org/github.com/gavv/httpexpect). It contains an overview and reference.

## Examples

See [`_examples`](_examples) directory for complete standalone examples.

* [`fruits_test.go`](_examples/fruits_test.go)

    Testing simple CRUD server written using bare `net/http`.

* [`iris_test.go`](_examples/iris_test.go)

    Testing various examples written using [`iris`](https://github.com/kataras/iris/) framework: JSON queries and validation, url and form parameters, basic auth, sessions, streaming. Running tests via `fasthttp.RequestHandler`.

* [`echo_test.go`](_examples/echo_test.go)

    Testing JWT authentication implemented with [`echo`](https://github.com/labstack/echo/) framework. Running tests via http client or `http.Handler`.

* [`gae_test.go`](_examples/gae_test.go)

    Testing an application running in [Google App Engine](https://ru.wikipedia.org/wiki/Google_App_Engine).

## Quick start

##### Hello, world!

```go
package example

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gavv/httpexpect"
)

func TestFruits(t *testing.T) {
	// create http.Handler
	handler := FruitServer()

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	// create httpexpect instance
	e := httpexpect.New(t, server.URL)

	// is it working?
	e.GET("/fruits").
		Expect().
		Status(http.StatusOK).JSON().Array().Empty()
}
```

##### JSON

```go
	orange := map[string]interface{}{
		"weight": 100,
	}

	e.PUT("/fruits/orange").WithJSON(orange).
		Expect().
		Status(http.StatusNoContent).NoContent()

	e.GET("/fruits/orange").
		Expect().
		Status(http.StatusOK).
		JSON().Object().ContainsKey("weight").ValueEqual("weight", 100)

	apple := map[string]interface{}{
		"colors": []interface{}{"green", "red"},
		"weight": 200,
	}

	e.PUT("/fruits/apple").WithJSON(apple).
		Expect().
		Status(http.StatusNoContent).NoContent()

	obj := e.GET("/fruits/apple").
		Expect().
		Status(http.StatusOK).JSON().Object()

	obj.Keys().ContainsOnly("colors", "weight")

	obj.Value("colors").Array().Elements("green", "red")
	obj.Value("colors").Array().Element(0).String().Equal("green")
	obj.Value("colors").Array().Element(1).String().Equal("red")
	obj.Value("colors").Array().First().String().Equal("green")
	obj.Value("colors").Array().Last().String().Equal("red")
```

##### JSON Schema and JSON Path

```go
	schema := `{
		"type": "array",
		"items": {
			"type": "object",
			"properties": {
				...
				"private": {
					"type": "boolean"
				}
			}
		}
	}`

	repos := e.GET("/repos/octocat").
		Expect().
		Status(http.StatusOK).JSON()

	// validate JSON schema
	repos.Schema(schema)

	// run JSONPath query and iterate results
	for _, private := range repos.Path("$..private").Array().Iter() {
		private.Boolean().False()
	}
```

##### Forms

```go
	// post form encoded from struct or map
	e.POST("/form").WithForm(structOrMap).
		Expect().
		Status(http.StatusOK)

	// set individual fields
	e.POST("/form").WithFormField("foo", "hello").WithFormField("bar", 123).
		Expect().
		Status(http.StatusOK)

	// multipart form
	e.POST("/form").WithMultipart().
		WithFile("avatar", "./john.png").WithFormField("username", "john").
		Expect().
		Status(http.StatusOK)
```

##### URL construction

```go
	// construct path using ordered parameters
	e.GET("/repos/{user}/{repo}", "octocat", "hello-world").
		Expect().
		Status(http.StatusOK)

	// construct path using named parameters
	e.GET("/repos/{user}/{repo}").
		WithPath("user", "octocat").WithPath("repo", "hello-world").
		Expect().
		Status(http.StatusOK)

	// set query parameters
	e.GET("/repos/{user}", "octocat").WithQuery("sort", "asc").
		Expect().
		Status(http.StatusOK)    // "/repos/octocat?sort=asc"
```

##### Headers

```go
	// set If-Match
	e.POST("/users/john").WithHeader("If-Match", etag).WithJSON(john).
		Expect().
		Status(http.StatusOK)

	// check ETag
	e.GET("/users/john").
		Expect().
		Status(http.StatusOK).Header("ETag").NotEmpty()

	// check Date
	t := time.Now()

	e.GET("/users/john").
		Expect().
		Status(http.StatusOK).Header("Date").DateTime().InRange(t, time.Now())
```

##### Cookies

```go
	// set cookie
	t := time.Now()

	e.POST("/users/john").WithCookie("session", sessionID).WithJSON(john).
		Expect().
		Status(http.StatusOK)

	// check cookies
	c := e.GET("/users/john").
		Expect().
		Status(http.StatusOK).Cookie("session")

	c.Value().Equal(sessionID)
	c.Domain().Equal("example.com")
	c.Path().Equal("/")
	c.Expires().InRange(t, t.Add(time.Hour * 24))
```

##### Regular expressions

```go
	// simple match
	e.GET("/users/john").
		Expect().
		Header("Location").
		Match("http://(.+)/users/(.+)").Values("example.com", "john")

	// check capture groups by index or name
	m := e.GET("/users/john").
		Expect().
		Header("Location").Match("http://(?P<host>.+)/users/(?P<user>.+)")

	m.Index(0).Equal("http://example.com/users/john")
	m.Index(1).Equal("example.com")
	m.Index(2).Equal("john")

	m.Name("host").Equal("example.com")
	m.Name("user").Equal("john")
```

##### Subdomains and per-request URL

```go
	e.GET("/path").WithURL("http://example.com").
	   Expect().
	   Status(http.StatusOK)

	e.GET("/path").WithURL("http://subdomain.example.com").
	   Expect().
	   Status(http.StatusOK)
```

##### Reusable builders

```go
	e := httpexpect.New(t, "http://example.com")

	r := e.POST("/login").WithForm(Login{"ford", "betelgeuse7"}).
		Expect().
		Status(http.StatusOK).JSON().Object()

	token := r.Value("token").String().Raw()

	auth := e.Builder(func (req *httpexpect.Request) {
		req.WithHeader("Authorization", "Bearer "+token)
	})

	auth.GET("/restricted").
	   Expect().
	   Status(http.StatusOK)

	e.GET("/restricted").
	   Expect().
	   Status(http.StatusUnauthorized)
```

##### Custom config

```go
	e := httpexpect.WithConfig(httpexpect.Config{
		// prepend this url to all requests
		BaseURL: "http://example.com",

		// use http.Client with a cookie jar and timeout
		Client: &http.Client{
			Jar:     httpexpect.NewJar(),
			Timeout: time.Second * 30,
		},

		// use fatal failures
		Reporter: httpexpect.NewRequireReporter(t),

		// use verbose logging
		Printers: []httpexpect.Printer{
			httpexpect.NewCurlPrinter(t),
			httpexpect.NewDebugPrinter(t, true),
		},
	})
```

##### Session support

```go
	// cookie jar is used to store cookies from server
	e := httpexpect.WithConfig(httpexpect.Config{
		Reporter: httpexpect.NewAssertReporter(t),
		Client: &http.Client{
			Jar: httpexpect.NewJar(), // used by default if Client is nil
		},
	})

	// cookies are disabled
	e := httpexpect.WithConfig(httpexpect.Config{
		Reporter: httpexpect.NewAssertReporter(t),
		Client: &http.Client{
			Jar: nil,
		},
	})
```

##### Use HTTP handler directly

```go
	// invoke http.Handler directly using httpexpect.Binder
	var handler http.Handler = myHandler()

	e := httpexpect.WithConfig(httpexpect.Config{
		Reporter: httpexpect.NewAssertReporter(t),
		Client: &http.Client{
			Transport: httpexpect.NewBinder(handler),
			Jar:       httpexpect.NewJar(),
		},
	})

	// invoke fasthttp.RequestHandler directly using httpexpect.FastBinder
	var handler fasthttp.RequestHandler = myHandler()

	e := httpexpect.WithConfig(httpexpect.Config{
		Reporter: httpexpect.NewAssertReporter(t),
		Client: &http.Client{
			Transport: httpexpect.NewFastBinder(handler),
			Jar:       httpexpect.NewJar(),
		},
	})
```

## Similar packages

* [`gorequest`](https://github.com/parnurzeal/gorequest)
* [`gabs`](https://github.com/Jeffail/gabs)
* [`gofight`](https://github.com/appleboy/gofight)
* [`baloo`](https://github.com/h2non/baloo)
* [`frisby`](https://github.com/verdverm/frisby)
* [`forest`](https://github.com/emicklei/forest)
* [`restit`](https://github.com/go-restit/restit)
* [`http-test`](https://github.com/vsco/http-test)
* [`go-json-rest/rest/test`](https://godoc.org/github.com/ant0ine/go-json-rest/rest/test)

## Contributing

Feel free to report bugs, suggest improvements, and send pull requests! Don't forget to add documentation and tests for new features and run all tests before submitting pull requests:

```
go test github.com/gavv/httpexpect
go test github.com/gavv/httpexpect/_examples
```

## License

[MIT](LICENSE)
