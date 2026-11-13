/*
Package iris is a web framework for Go.

It gives you routing with typed path parameters, dependency injection that keeps
handlers testable, one place to map every error your application returns, and
defaults that are safe to run in production. Handlers are plain functions over a
pooled [Context]. After [Application.Build] the application is an ordinary
http.Handler, so anything that speaks net/http still fits around it.

Source code and other details for the project are available at GitHub:

	https://github.com/kataras/iris

# Installation

The only requirement is the Go Programming Language, version 1.27 or later.

	$ go get github.com/kataras/iris/v14@latest

The release you are running is the [Version] constant. HISTORY.md records what
changed in each one and MIGRATION.md covers the upgrade from v12, which is the
largest break in the project's history.

# Quick Start

	package main

	import "github.com/kataras/iris/v14"

	func main() {
		app := iris.New()

		app.Get("/", func(ctx iris.Context) {
			ctx.JSON(iris.Map{"message": "Hello, World!"})
		})

		app.Get("/greet/{name:string}", func(ctx iris.Context) {
			ctx.WriteString("Hello, " + ctx.Params().GetString("name"))
		})

		app.Listen(":8080")
	}

[Default] returns the same application with a debug logger, permissive CORS,
panic recovery and request IDs already registered. Use it while you explore,
not in production.

# Application Setup

There are three ways in. [New] builds a bare application that you configure
yourself. [Default] pre-registers the middleware above. [NewBuilder] is the one
to reach for in production: the steps return narrowing interfaces, so the
compiler rejects a chain written in the wrong order.

	app := iris.NewBuilder().
		Prefix("/api").
		AllowOrigin("*").
		Compression(true).
		LogRequests(true).
		Health(true, "production", "you@example.com").
		Errors(errors.NewOptions().
			MapErrors(errors.NotFound, sql.ErrNoRows)).
		Services(catalog.NewRepository, catalog.NewService). // your constructors
		RouterMiddlewares().
		Middlewares().
		API("/products", api.NewProductsAPI). // your controller
		Build()

	app.Listen(":8080")

Configuration lives in [Options], which is itself a [Configurator], so a partial
literal sets only the fields you name. [Defaults] takes edit callbacks, [YAML]
reads a file, and [Options.Bind] reads the SERVER_CONFIG environment variable.
[Application.EditOptions] changes them after construction.

[Application.Listen] serves HTTP on an address. [Application.Run] takes
[ServerOptions] when you need TLS, a custom tls.Config or a prepared listener.
[Application.Shutdown] drains the connections and then runs the closers of
[Application.AddCloser]. Listen and Run return once it has. Ctrl-C alone drains
the server and exits: pass Shutdown to [RegisterOnInterrupt] and the signal runs
the closers too.

# Routing

Path parameters carry a type, and a request whose parameter does not match is
rejected before your handler runs:

	app.Get("/products/{id:uuid}", show)
	app.Get("/users/{age:int min(18)}", create)
	app.Get("/assets/{file:path}", download)

The built-in macros are string, int and its sized forms, uint and its sized
forms, bool, alphabetical, file, path, uuid, uuidv4, mail, email, date and
weekday. Each accepts constraint functions such as min, max and regexp. Register
your own through app.Macros().

Group routes with app.Router("/prefix"). For controllers, implement [Routing]
and let the dependency container resolve the constructor's arguments.

# The Context

[Context] is an alias for *context.Context. One is acquired from a pool per
request and released when the chain ends, so it must not be retained past the
handler's return or used from another goroutine. Clone it first when you need to
hand it to one.

The request side reads through Params, the Read family (ReadJSON, ReadForm,
ReadQuery and friends) and RemoteAddr. The response side writes through JSON,
Text, HTML, Negotiate and the cookie helpers. FromStd wraps an existing
net/http handler; [NewContextWrapper] swaps in your own context type.

The full method set is documented in
[github.com/kataras/iris/v14/context].

# Request Binding and Validation

ctx.BindJSON decodes the body into your type and, when that type has a
Validate method, runs it. Both failures write the 400 for you, so a handler that
gets ok == false has nothing left to do:

	type CreateUserInput struct {
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	func (in *CreateUserInput) Validate() error {
		if in.Age < 18 {
			return errors.New("age must be 18 or older")
		}

		return nil
	}

	func create(ctx iris.Context) {
		input, ok := ctx.BindJSON[CreateUserInput]()
		if !ok {
			return // 400 already written.
		}

		id, err := users.Create(ctx, input)
		ctx.Created(id, err) // 201, or the centrally mapped error.
	}

Set Options.Context.RequestValidator to run a validator after every decode
regardless of type, and Options.Context.BodyMaxMemory to cap how much of a body
any reader will accept.

See [github.com/kataras/iris/v14/context] for the full list of Context methods.

# Error Handling

Register your error codes once and map your domain errors onto them. Responses
follow RFC 7807, and an error you have not mapped renders as a generic 500
rather than leaking its message. Errors that implement ErrPrivate never reach
the client at all.

See [github.com/kataras/iris/v14/rest/errors], which documents the model in
full.

# Security Defaults

  - Uploads. UploadFormFiles runs every filename through SafeFilename, which
    rejects traversal, symlink escapes and reserved Windows device names such as
    NUL and COM1. A rejected file is skipped rather than saved. Call
    SafeFilename yourself if you write files through SaveFormFile instead.
  - JSONP. A callback is validated before it is written into the script
    response. An invalid one fails with context.ErrInvalidJSONPCallback rather
    than reflecting attacker-supplied script.
  - Client address. Context.RemoteAddr filters proxy headers against the private
    subnets you configure, and trusts no X- header until
    Options.Context.RemoteAddrHeaders names one. Context.Addr is the client's own
    unfiltered claim, so use it for logging and never for authorization.
  - Response headers. Context.SetHeader replaces, so a middleware registered
    twice cannot emit the duplicate that browsers reject. Context.AddHeader
    appends, for list-valued headers such as Vary.
  - Body size. Options.Context.BodyMaxMemory is off by default: 0 means no
    limit. Set it and every body reader is capped, answering 413 past the
    limit. Context.ContentLength reports -1 when the length is unknown, as it
    is for a chunked request, so enforce the limit rather than trusting the
    header.
  - Sessions and cookies. Session cookies default to Secure with SameSite=Lax,
    and Context.AddCookie defaults to SameSite=Lax.

SECURITY.md lists the opt-in hardening worth turning on, and MIGRATION.md
explains each default that changed since v12.

# Middleware

Middleware registered with UseRouter runs for every request, including the ones
that match no route. Middleware registered with Use runs for matched routes
only, and Done runs after the handler.

About thirty middleware ship with the framework, among them accesslog,
basicauth, bodylimit, cache, cors, jwt, rate, recovery, requestid and sessions:

	https://github.com/kataras/iris/tree/main/middleware
	https://github.com/iris-contrib/middleware

# Testing

The httptest package runs requests against your application in memory. No port
is bound, no network is touched and nothing sleeps:

	func TestIndex(t *testing.T) {
		app := newApp()

		e := httptest.New(t, app)
		e.GET("/").Expect().Status(httptest.StatusOK)
	}

# Learning Iris

Tutorial:

	https://github.com/kataras/iris/blob/main/TUTORIAL.md

Examples:

	https://github.com/kataras/iris/tree/main/_examples

Upgrading from v12:

	https://github.com/kataras/iris/blob/main/MIGRATION.md

Frequently asked questions:

	https://github.com/kataras/iris/blob/main/FAQ.md

Home Page:

	https://www.iris-go.com
*/
package iris
