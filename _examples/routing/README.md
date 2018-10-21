# Routing

## Handlers

A Handler, as the name implies, handle requests.

```go
// A Handler responds to an HTTP request.
// It writes reply headers and data to the
// Context.ResponseWriter() and then return.
// Returning signals that the request is finished;
// it is not valid to use the Context after or
// concurrently with the completion of the Handler call.
//
// Depending on the HTTP client software, HTTP protocol version,
// and any intermediaries between the client and the iris server,
// it may not be possible to read from the
// Context.Request().Body after writing to the context.ResponseWriter().
// Cautious handlers should read the Context.Request().Body first, and then reply.
//
// Except for reading the body, handlers should not modify the provided Context.
//
// If Handler panics, the server (the caller of Handler) assumes that
// the effect of the panic was isolated to the active request.
// It recovers the panic, logs a stack trace to the server error log
// and hangs up the connection.
type Handler func(Context)
```

Once the handler is registered, we can use the returned [`Route`](https://godoc.org/github.com/kataras/iris/core/router#Route) instance to give a name to the handler registration for easier lookup in code or in templates.

For more information, checkout the [Routing and reverse lookups](routing_reverse.md) section.

## API

All HTTP methods are supported, developers can also register handlers for same paths for different methods.

The first parameter is the HTTP Method,
second parameter is the request path of the route,
third variadic parameter should contains one or more iris.Handler executed
by the registered order when a user requests for that specific resouce path from the server.

Example code:

```go
app := iris.New()

app.Handle("GET", "/contact", func(ctx iris.Context) {
    ctx.HTML("<h1> Hello from /contact </h1>")
})
```

In order to make things easier for the end-developer, iris provides functions for all HTTP Methods.
The first parameter is the request path of the route,
second variadic parameter should contains one or more iris.Handler executed
by the registered order when a user requests for that specific resouce path from the server.

Example code:

```go
app := iris.New()

// Method: "GET"
app.Get("/", handler)

// Method: "POST"
app.Post("/", handler)

// Method: "PUT"
app.Put("/", handler)

// Method: "DELETE"
app.Delete("/", handler)

// Method: "OPTIONS"
app.Options("/", handler)

// Method: "TRACE"
app.Trace("/", handler)

// Method: "CONNECT"
app.Connect("/", handler)

// Method: "HEAD"
app.Head("/", handler)

// Method: "PATCH"
app.Patch("/", handler)

// register the route for all HTTP Methods
app.Any("/", handler)

func handler(ctx iris.Context){
    ctx.Writef("Hello from method: %s and path: %s", ctx.Method(), ctx.Path())
}
```

## Grouping Routes

A set of routes that are being groupped by path prefix can (optionally) share the same middleware handlers and template layout.
A group can have a nested group too.

`.Party` is being used to group routes, developers can declare an unlimited number of (nested) groups.

Example code:

```go
app := iris.New()

users := app.Party("/users", myAuthMiddlewareHandler)

// http://localhost:8080/users/42/profile
users.Get("/{id:uint64}/profile", userProfileHandler)
// http://localhost:8080/users/messages/1
users.Get("/inbox/{id:uint64}", userMessageHandler)
```

The same could be also written using a function which accepts the child router(the Party).

```go
app := iris.New()

app.PartyFunc("/users", func(users iris.Party) {
    users.Use(myAuthMiddlewareHandler)

    // http://localhost:8080/users/42/profile
    users.Get("/{id:uint64}/profile", userProfileHandler)
    // http://localhost:8080/users/messages/1
    users.Get("/inbox/{id:uint64}", userMessageHandler)
})
```

> `id:uint` is a (typed) dynamic path parameter, learn more by scrolling down.

# Dynamic Path Parameters

Iris has the easiest and the most powerful routing process you have ever meet.

At the same time,
Iris has its own interpeter(yes like a programming language)
for route's path syntax and their dynamic path parameters parsing and evaluation.
We call them "macros" for shortcut.

How? It calculates its needs and if not any special regexp needed then it just
registers the route with the low-level path syntax,
otherwise it pre-compiles the regexp and adds the necessary middleware(s). That means that you have zero performance cost compared to other routers or web frameworks.

Standard macro types for route path parameters

```sh
+------------------------+
| {param:string}         |
+------------------------+
string type
anything (single path segmnent)

+-------------------------------+
| {param:int}                   |
+-------------------------------+
int type
-9223372036854775808 to 9223372036854775807 (x64) or -2147483648 to 2147483647 (x32), depends on the host arch

+------------------------+
| {param:int8}           |
+------------------------+
int8 type
-128 to 127

+------------------------+
| {param:int16}          |
+------------------------+
int16 type
-32768 to 32767

+------------------------+
| {param:int32}          |
+------------------------+
int32 type
-2147483648 to 2147483647

+------------------------+
| {param:int64}          |
+------------------------+
int64 type
-9223372036854775808 to 9223372036854775807

+------------------------+
| {param:uint}           |
+------------------------+
uint type
0 to 18446744073709551615 (x64) or 0 to 4294967295 (x32)

+------------------------+
| {param:uint8}          |
+------------------------+
uint8 type
0 to 255

+------------------------+
| {param:uint16}         |
+------------------------+
uint16 type
0 to 65535

+------------------------+
| {param:uint32}          |
+------------------------+
uint32 type
0 to 4294967295

+------------------------+
| {param:uint64}         |
+------------------------+
uint64 type
0 to 18446744073709551615

+---------------------------------+
| {param:bool} or {param:boolean} |
+---------------------------------+
bool type
only "1" or "t" or "T" or "TRUE" or "true" or "True"
or "0" or "f" or "F" or "FALSE" or "false" or "False"

+------------------------+
| {param:alphabetical}   |
+------------------------+
alphabetical/letter type
letters only (upper or lowercase)

+------------------------+
| {param:file}           |
+------------------------+
file type
letters (upper or lowercase)
numbers (0-9)
underscore (_)
dash (-)
point (.)
no spaces ! or other character

+------------------------+
| {param:path}           |
+------------------------+
path type
anything, should be the last part, can be more than one path segment,
i.e: "/test/*param" and request: "/test/path1/path2/path3" , ctx.Params().Get("param") == "path1/path2/path3"
```

If type is missing then parameter's type is defaulted to string, so
`{param} == {param:string}`.

If a function not found on that type then the `string` macro type's functions are being used.

Besides the fact that iris provides the basic types and some default "macro funcs"
you are able to register your own too!.

Register a named path parameter function

```go
app.Macros().Get("int").RegisterFunc("min", func(argument int) func(paramValue int) bool {
    // [...]
    return func(paramValue int) bool {
         // -> true means valid, false means invalid fire 404
         // or if "else 500" is appended to the macro syntax then internal server error.
        return true 
    }
})
```

At the `func(argument ...)` you can have any standard type, it will be validated before the server starts so don't care about any performance cost there, the only thing it runs at serve time is the returning `func(paramValue <T>) bool`.

```go
{param:string equal(iris)}
```
The "iris" will be the argument here:
```go
app.Macros().Get("string").RegisterFunc("equal", func(argument string) func(paramValue string) bool {
    return func(paramValue string){ return argument == paramValue }
})
```

Example Code:

```go
app := iris.New()
// you can use the "string" type which is valid for a single path parameter that can be anything.
app.Get("/username/{name}", func(ctx iris.Context) {
    ctx.Writef("Hello %s", ctx.Params().Get("name"))
}) // type is missing = {name:string}

// Let's register our first macro attached to int macro type.
// "min" = the function
// "minValue" = the argument of the function
// func(<T>) bool = the macro's path parameter evaluator, this executes in serve time when
// a user requests a path which contains the :int macro type with the min(...) macro parameter function.
app.Macros().Get("int").RegisterFunc("min", func(minValue int) func(int) bool {
    // do anything before serve here [...]
    // at this case we don't need to do anything
    return func(paramValue int) bool {
        return paramValue >= minValue
    }
})

// http://localhost:8080/profile/id>=1
// this will throw 404 even if it's found as route on : /profile/0, /profile/blabla, /profile/-1
// macro parameter functions are optional of course.
app.Get("/profile/{id:uint64 min(1)}", func(ctx iris.Context) {
    // second parameter is the error but it will always nil because we use macros,
    // the validaton already happened.
    id, _ := ctx.Params().GetUint64("id")
    ctx.Writef("Hello id: %d", id)
})

// to change the error code per route's macro evaluator:
app.Get("/profile/{id:uint64 min(1)}/friends/{friendid:uint64 min(1) else 504}", func(ctx iris.Context) {
    id, _ := ctx.Params().GetUint64("id")
    friendid, _ := ctx.Params().GetUint64("friendid")
    ctx.Writef("Hello id: %d looking for friend id: ", id, friendid)
}) // this will throw e 504 error code instead of 404 if all route's macros not passed.

// http://localhost:8080/game/a-zA-Z/level/42
// remember, alphabetical is lowercase or uppercase letters only.
app.Get("/game/{name:alphabetical}/level/{level:int}", func(ctx iris.Context) {
    ctx.Writef("name: %s | level: %s", ctx.Params().Get("name"), ctx.Params().Get("level"))
})

// let's use a trivial custom regexp that validates a single path parameter
// which its value is only lowercase letters.

// http://localhost:8080/lowercase/anylowercase
app.Get("/lowercase/{name:string regexp(^[a-z]+)}", func(ctx iris.Context) {
    ctx.Writef("name should be only lowercase, otherwise this handler will never executed: %s", ctx.Params().Get("name"))
})

// http://localhost:8080/single_file/app.js
app.Get("/single_file/{myfile:file}", func(ctx iris.Context) {
    ctx.Writef("file type validates if the parameter value has a form of a file name, got: %s", ctx.Params().Get("myfile"))
})

// http://localhost:8080/myfiles/any/directory/here/
// this is the only macro type that accepts any number of path segments.
app.Get("/myfiles/{directory:path}", func(ctx iris.Context) {
    ctx.Writef("path type accepts any number of path segments, path after /myfiles/ is: %s", ctx.Params().Get("directory"))
})

app.Run(iris.Addr(":8080"))
}
```

A path parameter name should contain only alphabetical letters or digits. Symbols like  '_' are NOT allowed.
Last, do not confuse `ctx.Params()` with `ctx.Values()`.
Path parameter's values can be retrieved from `ctx.Params()`,
context's local storage that can be used to communicate between handlers and middleware(s) can be stored to `ctx.Values()`.

# Routing and reverse lookups

As mentioned in the [Handlers](handlers.md) chapter, Iris provides several handler registration methods, each of which returns a [`Route`](https://godoc.org/github.com/kataras/iris/core/router#Route) instance.

## Route naming

Route naming is easy, since we just call the returned `*Route` with a `Name` field to define a name:

```go
package main

import (
    "github.com/kataras/iris"
)

func main() {
    app := iris.New()
    // define a function
    h := func(ctx iris.Context) {
        ctx.HTML("<b>Hi</b1>")
    }

    // handler registration and naming
    home := app.Get("/", h)
    home.Name = "home"
    // or
    app.Get("/about", h).Name = "about"
    app.Get("/page/{id}", h).Name = "page"

    app.Run(iris.Addr(":8080"))
}
```

## Route reversing AKA generating URLs from the route name

When we register the handlers for a specific path, we get the ability to create URLs based on the structured data we pass to Iris. In the example above, we've named three routers, one of which even takes parameters. If we're using the default `html/template` view engine, we can use a simple action to reverse the routes (and generae actual URLs):

```sh
Home: {{ urlpath "home" }}
About: {{ urlpath "about" }}
Page 17: {{ urlpath "page" "17" }}
```

Above code would generate the following output:

```sh
Home: http://localhost:8080/ 
About: http://localhost:8080/about
Page 17: http://localhost:8080/page/17
```

## Using route names in code

We can use the following methods/functions to work with named routes (and their parameters):

* [`GetRoutes`](https://godoc.org/github.com/kataras/iris/core/router#APIBuilder.GetRoutes) function to get all registered routes
* [`GetRoute(routeName string)`](https://godoc.org/github.com/kataras/iris/core/router#APIBuilder.GetRoute) method to retrieve a route by name
* [`URL(routeName string, paramValues ...interface{})`](https://godoc.org/github.com/kataras/iris/core/router#RoutePathReverser.URL) method to generate url string based on supplied parameters
* [`Path(routeName string, paramValues ...interface{}`](https://godoc.org/github.com/kataras/iris/core/router#RoutePathReverser.Path) method to generate just the path (without host and protocol) portion of the URL based on provided values

## Examples

Check out the [https://github.com/kataras/iris/tree/master/_examples/view/template_html_4](https://github.com/kataras/iris/tree/master/_examples/view/template_html_4) example for more details.

# Middleware

When we talk about Middleware in Iris we're talking about running code before and/or after our main handler code in a HTTP request lifecycle. For example, logging middleware might write the incoming request details to a log, then call the handler code, before writing details about the response to the log. One of the cool things about middleware is that these units are extremely flexible and reusable.

A middleware is just a **Handler** form of `func(ctx iris.Context)`, the middleware is being executed when the previous middleware calls the `ctx.Next()`, this can be used for authentication, i.e: if logged in then `ctx.Next()` otherwise fire an error response. 

## Writing a middleware

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // or app.Use(before) and app.Done(after).
    app.Get("/", before, mainHandler, after)
    app.Run(iris.Addr(":8080"))
}

func before(ctx iris.Context) {
    shareInformation := "this is a sharable information between handlers"

    requestPath := ctx.Path()
    println("Before the mainHandler: " + requestPath)

    ctx.Values().Set("info", shareInformation)
    ctx.Next() // execute the next handler, in this case the main one.
}

func after(ctx iris.Context) {
    println("After the mainHandler")
}

func mainHandler(ctx iris.Context) {
    println("Inside mainHandler")

    // take the info from the "before" handler.
    info := ctx.Values().GetString("info")

    // write something to the client as a response.
    ctx.HTML("<h1>Response</h1>")
    ctx.HTML("<br/> Info: " + info)

    ctx.Next() // execute the "after".
}
```

```bash
$ go run main.go # and navigate to the http://localhost:8080
Now listening on: http://localhost:8080
Application started. Press CTRL+C to shut down.
Before the mainHandler: /
Inside mainHandler
After the mainHandler
```

### Globally

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()

    // register our routes.
    app.Get("/", indexHandler)
    app.Get("/contact", contactHandler)

    // Order of those calls doesn't matter, `UseGlobal` and `DoneGlobal`
    // are applied to existing routes and future routes.
    //
    // Remember: the `Use` and `Done` are applied to the current party's and its children,
    // so if we used the `app.Use/Don`e before the routes registration
    // it would work like UseGlobal/DoneGlobal in this case, because the `app` is the root party.
    //
    // See `app.Party/PartyFunc` for more.
    app.UseGlobal(before)
    app.DoneGlobal(after)

    app.Run(iris.Addr(":8080"))
}

func before(ctx iris.Context) {
     // [...]
}

func after(ctx iris.Context) {
    // [...]
}

func indexHandler(ctx iris.Context) {
    // write something to the client as a response.
    ctx.HTML("<h1>Index</h1>")

    ctx.Next() // execute the "after" handler registered via `Done`.
}

func contactHandler(ctx iris.Context) {
    // write something to the client as a response.
    ctx.HTML("<h1>Contact</h1>")

    ctx.Next() // execute the "after" handler registered via `Done`.
}
```

## [Explore](https://github.com/kataras/iris/tree/master/middleware)

# Wrapping the Router

**Very rare**, you may never need that but it's here in any case you need it.

There are times you need to override or decide whether the Router will be executed on an incoming request. If you've any previous experience with the `net/http` and other web frameworks this function will be familiar with you (it has the form of a net/http middleware, but instead of accepting the next handler it accepts the Router as a function to be executed or not).

```go
// WrapperFunc is used as an expected input parameter signature
// for the WrapRouter. It's a "low-level" signature which is compatible
// with the net/http.
// It's being used to run or no run the router based on a custom logic.
type WrapperFunc func(w http.ResponseWriter, r *http.Request, firstNextIsTheRouter http.HandlerFunc)

// WrapRouter adds a wrapper on the top of the main router.
// Usually it's useful for third-party middleware
// when need to wrap the entire application with a middleware like CORS.
//
// Developers can add more than one wrappers,
// those wrappers' execution comes from last to first.
// That means that the second wrapper will wrap the first, and so on.
//
// Before build.
func WrapRouter(wrapperFunc WrapperFunc)
```

Iris' router searches for its routes based on the `HTTP Method` a Router Wrapper can override that behavior and execute custom code.

Example Code:

```go
package main

import (
    "net/http"
    "strings"

    "github.com/kataras/iris"
)

// In this example you'll just see one use case of .WrapRouter.
// You can use the .WrapRouter to add custom logic when or when not the router should
// be executed in order to execute the registered routes' handlers.
//
// To see how you can serve files on root "/" without a custom wrapper
// just navigate to the "file-server/single-page-application" example.
//
// This is just for the proof of concept, you can skip this tutorial if it's too much for you.


func main() {
    app := iris.New()

    app.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
        ctx.HTML("<b>Resource Not found</b>")
    })

    app.Get("/", func(ctx iris.Context) {
        ctx.ServeFile("./public/index.html", false)
    })

    app.Get("/profile/{username}", func(ctx iris.Context) {
        ctx.Writef("Hello %s", ctx.Params().Get("username"))
    })

    // serve files from the root "/", if we used .StaticWeb it could override
    // all the routes because of the underline need of wildcard.
    // Here we will see how you can by-pass this behavior
    // by creating a new file server handler and
    // setting up a wrapper for the router(like a "low-level" middleware)
    // in order to manually check if we want to process with the router as normally
    // or execute the file server handler instead.

    // use of the .StaticHandler
    // which is the same as StaticWeb but it doesn't
    // registers the route, it just returns the handler.
    fileServer := app.StaticHandler("./public", false, false)

    // wrap the router with a native net/http handler.
    // if url does not contain any "." (i.e: .css, .js...)
    // (depends on the app , you may need to add more file-server exceptions),
    // then the handler will execute the router that is responsible for the
    // registered routes (look "/" and "/profile/{username}")
    // if not then it will serve the files based on the root "/" path.
    app.WrapRouter(func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
        path := r.URL.Path
        // Note that if path has suffix of "index.html" it will auto-permant redirect to the "/",
        // so our first handler will be executed instead.

        if !strings.Contains(path, ".") { 
            // if it's not a resource then continue to the router as normally. <-- IMPORTANT
            router(w, r)
            return
        }
        // acquire and release a context in order to use it to execute
        // our file server
        // remember: we use net/http.Handler because here we are in the "low-level", before the router itself.
        ctx := app.ContextPool.Acquire(w, r)
        fileServer(ctx)
        app.ContextPool.Release(ctx)
    })

    // http://localhost:8080
    // http://localhost:8080/index.html
    // http://localhost:8080/app.js
    // http://localhost:8080/css/main.css
    // http://localhost:8080/profile/anyusername
    app.Run(iris.Addr(":8080"))

    // Note: In this example we just saw one use case,
    // you may want to .WrapRouter or .Downgrade in order to bypass the iris' default router, i.e:
    // you can use that method to setup custom proxies too.
    //
    // If you just want to serve static files on other path than root
    // you can just use the StaticWeb, i.e:
    // 					     .StaticWeb("/static", "./public")
    // ________________________________requestPath, systemPath
}
```

There is not much to say here, it's just a function wrapper which accepts the native response writer and request and the next handler which is the Iris' Router itself, it's being or not executed whether is called or not, **it's a middleware for the whole Router**.

# Error handlers

You can define your own handlers when a specific http error code occurs.

> Error codes are the http status codes that are bigger or equal to 400, like 404 not found and 500 internal server.

Example code:

```go
package main

import "github.com/kataras/iris"

func main(){
    app := iris.New()
    app.OnErrorCode(iris.StatusNotFound, notFound)
    app.OnErrorCode(iris.StatusInternalServerError, internalServerError)
    // to register a handler for all "error" status codes(context.StatusCodeNotSuccessful)
    // defaults to < 200 || >= 400:
    // app.OnAnyErrorCode(handler)
    app.Get("/", index)
    app.Run(iris.Addr(":8080"))
}

func notFound(ctx iris.Context) {
    // when 404 then render the template $views_dir/errors/404.html
    ctx.View("errors/404.html")
}

func internalServerError(ctx iris.Context) {
    ctx.WriteString("Oups something went wrong, try again")
}

func index(ctx context.Context) {
    ctx.View("index.html")
}
```

# Context Outline

The `iris.Context` source code can be found [here](https://github.com/kataras/iris/blob/master/context/context.go). Keep note using an IDE/Editors with `auto-complete` feature will help you a lot.

```go
// Context is the midle-man server's "object" for the clients.
//
// A New context is being acquired from a sync.Pool on each connection.
// The Context is the most important thing on the iris's http flow.
//
// Developers send responses to the client's request through a Context.
// Developers get request information from the client's request a Context.
//
// This context is an implementation of the context.Context sub-package.
// context.Context is very extensible and developers can override
// its methods if that is actually needed.
type Context interface {
    // ResponseWriter returns an http.ResponseWriter compatible response writer, as expected.
    ResponseWriter() ResponseWriter
    // ResetResponseWriter should change or upgrade the Context's ResponseWriter.
    ResetResponseWriter(ResponseWriter)

    // Request returns the original *http.Request, as expected.
    Request() *http.Request

    // SetCurrentRouteName sets the route's name internally,
    // in order to be able to find the correct current "read-only" Route when
    // end-developer calls the `GetCurrentRoute()` function.
    // It's being initialized by the Router, if you change that name
    // manually nothing really happens except that you'll get other
    // route via `GetCurrentRoute()`.
    // Instead, to execute a different path
    // from this context you should use the `Exec` function
    // or change the handlers via `SetHandlers/AddHandler` functions.
    SetCurrentRouteName(currentRouteName string)
    // GetCurrentRoute returns the current registered "read-only" route that
    // was being registered to this request's path.
    GetCurrentRoute() RouteReadOnly

    // Do calls the SetHandlers(handlers)
    // and executes the first handler,
    // handlers should not be empty.
    //
    // It's used by the router, developers may use that
    // to replace and execute handlers immediately.
    Do(Handlers)

    // AddHandler can add handler(s)
    // to the current request in serve-time,
    // these handlers are not persistenced to the router.
    //
    // Router is calling this function to add the route's handler.
    // If AddHandler called then the handlers will be inserted
    // to the end of the already-defined route's handler.
    //
    AddHandler(...Handler)
    // SetHandlers replaces all handlers with the new.
    SetHandlers(Handlers)
    // Handlers keeps tracking of the current handlers.
    Handlers() Handlers

    // HandlerIndex sets the current index of the
    // current context's handlers chain.
    // If -1 passed then it just returns the
    // current handler index without change the current index.rns that index, useless return value.
    //
    // Look Handlers(), Next() and StopExecution() too.
    HandlerIndex(n int) (currentIndex int)
    // Proceed is an alternative way to check if a particular handler
    // has been executed and called the `ctx.Next` function inside it.
    // This is useful only when you run a handler inside
    // another handler. It justs checks for before index and the after index.
    //
    // A usecase example is when you want to execute a middleware
    // inside controller's `BeginRequest` that calls the `ctx.Next` inside it.
    // The Controller looks the whole flow (BeginRequest, method handler, EndRequest)
    // as one handler, so `ctx.Next` will not be reflected to the method handler
    // if called from the `BeginRequest`.
    //
    // Although `BeginRequest` should NOT be used to call other handlers,
    // the `BeginRequest` has been introduced to be able to set
    // common data to all method handlers before their execution.
    // Controllers can accept middleware(s) from the MVC's Application's Router as normally.
    //
    // That said let's see an example of `ctx.Proceed`:
    //
    // var authMiddleware = basicauth.New(basicauth.Config{
    // 	Users: map[string]string{
    // 		"admin": "password",
    // 	},
    // })
    //
    // func (c *UsersController) BeginRequest(ctx iris.Context) {
    // 	if !ctx.Proceed(authMiddleware) {
    // 		ctx.StopExecution()
    // 	}
    // }
    // This Get() will be executed in the same handler as `BeginRequest`,
    // internally controller checks for `ctx.StopExecution`.
    // So it will not be fired if BeginRequest called the `StopExecution`.
    // func(c *UsersController) Get() []models.User {
    //	  return c.Service.GetAll()
    //}
    // Alternative way is `!ctx.IsStopped()` if middleware make use of the `ctx.StopExecution()` on failure.
    Proceed(Handler) bool
    // HandlerName returns the current handler's name, helpful for debugging.
    HandlerName() string
    // Next calls all the next handler from the handlers chain,
    // it should be used inside a middleware.
    //
    // Note: Custom context should override this method in order to be able to pass its own context.Context implementation.
    Next()
    // NextOr checks if chain has a next handler, if so then it executes it
    // otherwise it sets a new chain assigned to this Context based on the given handler(s)
    // and executes its first handler.
    //
    // Returns true if next handler exists and executed, otherwise false.
    //
    // Note that if no next handler found and handlers are missing then
    // it sends a Status Not Found (404) to the client and it stops the execution.
    NextOr(handlers ...Handler) bool
    // NextOrNotFound checks if chain has a next handler, if so then it executes it
    // otherwise it sends a Status Not Found (404) to the client and stops the execution.
    //
    // Returns true if next handler exists and executed, otherwise false.
    NextOrNotFound() bool
    // NextHandler returns (it doesn't execute) the next handler from the handlers chain.
    //
    // Use .Skip() to skip this handler if needed to execute the next of this returning handler.
    NextHandler() Handler
    // Skip skips/ignores the next handler from the handlers chain,
    // it should be used inside a middleware.
    Skip()
    // StopExecution if called then the following .Next calls are ignored,
    // as a result the next handlers in the chain will not be fire.
    StopExecution()
    // IsStopped checks and returns true if the current position of the Context is 255,
    // means that the StopExecution() was called.
    IsStopped() bool
    // OnConnectionClose registers the "cb" function which will fire (on its own goroutine, no need to be registered goroutine by the end-dev)
    // when the underlying connection has gone away.
    //
    // This mechanism can be used to cancel long operations on the server
    // if the client has disconnected before the response is ready.
    //
    // It depends on the `http#CloseNotify`.
    // CloseNotify may wait to notify until Request.Body has been
    // fully read.
    //
    // After the main Handler has returned, there is no guarantee
    // that the channel receives a value.
    //
    // Finally, it reports whether the protocol supports pipelines (HTTP/1.1 with pipelines disabled is not supported).
    // The "cb" will not fire for sure if the output value is false.
    //
    // Note that you can register only one callback for the entire request handler chain/per route.
    //
    // Look the `ResponseWriter#CloseNotifier` for more.
    OnConnectionClose(fnGoroutine func()) bool
    // OnClose registers the callback function "cb" to the underline connection closing event using the `Context#OnConnectionClose`
    // and also in the end of the request handler using the `ResponseWriter#SetBeforeFlush`.
    // Note that you can register only one callback for the entire request handler chain/per route.
    //
    // Look the `Context#OnConnectionClose` and `ResponseWriter#SetBeforeFlush` for more.
    OnClose(cb func())

    //  +------------------------------------------------------------+
    //  | Current "user/request" storage                             |
    //  | and share information between the handlers - Values().     |
    //  | Save and get named path parameters - Params()              |
    //  +------------------------------------------------------------+

    // Params returns the current url's named parameters key-value storage.
    // Named path parameters are being saved here.
    // This storage, as the whole Context, is per-request lifetime.
    Params() *RequestParams

    // Values returns the current "user" storage.
    // Named path parameters and any optional data can be saved here.
    // This storage, as the whole Context, is per-request lifetime.
    //
    // You can use this function to Set and Get local values
    // that can be used to share information between handlers and middleware.
    Values() *memstore.Store
    // Translate is the i18n (localization) middleware's function,
    // it calls the Get("translate") to return the translated value.
    //
    // Example: https://github.com/kataras/iris/tree/master/_examples/miscellaneous/i18n
    Translate(format string, args ...interface{}) string

    //  +------------------------------------------------------------+
    //  | Path, Host, Subdomain, IP, Headers etc...                  |
    //  +------------------------------------------------------------+

    // Method returns the request.Method, the client's http method to the server.
    Method() string
    // Path returns the full request path,
    // escaped if EnablePathEscape config field is true.
    Path() string
    // RequestPath returns the full request path,
    // based on the 'escape'.
    RequestPath(escape bool) string

    // Host returns the host part of the current url.
    Host() string
    // Subdomain returns the subdomain of this request, if any.
    // Note that this is a fast method which does not cover all cases.
    Subdomain() (subdomain string)
    // IsWWW returns true if the current subdomain (if any) is www.
    IsWWW() bool
    // RemoteAddr tries to parse and return the real client's request IP.
    //
    // Based on allowed headers names that can be modified from Configuration.RemoteAddrHeaders.
    //
    // If parse based on these headers fail then it will return the Request's `RemoteAddr` field
    // which is filled by the server before the HTTP handler.
    //
    // Look `Configuration.RemoteAddrHeaders`,
    //      `Configuration.WithRemoteAddrHeader(...)`,
    //      `Configuration.WithoutRemoteAddrHeader(...)` for more.
    RemoteAddr() string
    // GetHeader returns the request header's value based on its name.
    GetHeader(name string) string
    // IsAjax returns true if this request is an 'ajax request'( XMLHttpRequest)
    //
    // There is no a 100% way of knowing that a request was made via Ajax.
    // You should never trust data coming from the client, they can be easily overcome by spoofing.
    //
    // Note that "X-Requested-With" Header can be modified by any client(because of "X-"),
    // so don't rely on IsAjax for really serious stuff,
    // try to find another way of detecting the type(i.e, content type),
    // there are many blogs that describe these problems and provide different kind of solutions,
    // it's always depending on the application you're building,
    // this is the reason why this `IsAjax`` is simple enough for general purpose use.
    //
    // Read more at: https://developer.mozilla.org/en-US/docs/AJAX
    // and https://xhr.spec.whatwg.org/
    IsAjax() bool
    // IsMobile checks if client is using a mobile device(phone or tablet) to communicate with this server.
    // If the return value is true that means that the http client using a mobile
    // device to communicate with the server, otherwise false.
    //
    // Keep note that this checks the "User-Agent" request header.
    IsMobile() bool
    // GetReferrer extracts and returns the information from the "Referer" header as specified
    // in https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
    // or by the URL query parameter "referer".
    GetReferrer() Referrer
    //  +------------------------------------------------------------+
    //  | Headers helpers                                            |
    //  +------------------------------------------------------------+

    // Header adds a header to the response writer.
    Header(name string, value string)

    // ContentType sets the response writer's header key "Content-Type" to the 'cType'.
    ContentType(cType string)
    // GetContentType returns the response writer's header value of "Content-Type"
    // which may, setted before with the 'ContentType'.
    GetContentType() string
    // GetContentType returns the request's header value of "Content-Type".
    GetContentTypeRequested() string

    // GetContentLength returns the request's header value of "Content-Length".
    // Returns 0 if header was unable to be found or its value was not a valid number.
    GetContentLength() int64

    // StatusCode sets the status code header to the response.
    // Look .`GetStatusCode` too.
    StatusCode(statusCode int)
    // GetStatusCode returns the current status code of the response.
    // Look `StatusCode` too.
    GetStatusCode() int

    // Redirect sends a redirect response to the client
    // to a specific url or relative path.
    // accepts 2 parameters string and an optional int
    // first parameter is the url to redirect
    // second parameter is the http status should send,
    // default is 302 (StatusFound),
    // you can set it to 301 (Permant redirect)
    // or 303 (StatusSeeOther) if POST method,
    // or StatusTemporaryRedirect(307) if that's nessecery.
    Redirect(urlToRedirect string, statusHeader ...int)

    //  +------------------------------------------------------------+
    //  | Various Request and Post Data                              |
    //  +------------------------------------------------------------+

    // URLParam returns true if the url parameter exists, otherwise false.
    URLParamExists(name string) bool
    // URLParamDefault returns the get parameter from a request,
    // if not found then "def" is returned.
    URLParamDefault(name string, def string) string
    // URLParam returns the get parameter from a request, if any.
    URLParam(name string) string
    // URLParamTrim returns the url query parameter with trailing white spaces removed from a request.
    URLParamTrim(name string) string
    // URLParamTrim returns the escaped url query parameter from a request.
    URLParamEscape(name string) string
    // URLParamInt returns the url query parameter as int value from a request,
    // returns -1 and an error if parse failed.
    URLParamInt(name string) (int, error)
    // URLParamIntDefault returns the url query parameter as int value from a request,
    // if not found or parse failed then "def" is returned.
    URLParamIntDefault(name string, def int) int
    // URLParamInt32Default returns the url query parameter as int32 value from a request,
    // if not found or parse failed then "def" is returned.
    URLParamInt32Default(name string, def int32) int32
    // URLParamInt64 returns the url query parameter as int64 value from a request,
    // returns -1 and an error if parse failed.
    URLParamInt64(name string) (int64, error)
    // URLParamInt64Default returns the url query parameter as int64 value from a request,
    // if not found or parse failed then "def" is returned.
    URLParamInt64Default(name string, def int64) int64
    // URLParamFloat64 returns the url query parameter as float64 value from a request,
    // returns -1 and an error if parse failed.
    URLParamFloat64(name string) (float64, error)
    // URLParamFloat64Default returns the url query parameter as float64 value from a request,
    // if not found or parse failed then "def" is returned.
    URLParamFloat64Default(name string, def float64) float64
    // URLParamBool returns the url query parameter as boolean value from a request,
    // returns an error if parse failed or not found.
    URLParamBool(name string) (bool, error)
    // URLParams returns a map of GET query parameters separated by comma if more than one
    // it returns an empty map if nothing found.
    URLParams() map[string]string

    // FormValueDefault returns a single parsed form value by its "name",
    // including both the URL field's query parameters and the POST or PUT form data.
    //
    // Returns the "def" if not found.
    FormValueDefault(name string, def string) string
    // FormValue returns a single parsed form value by its "name",
    // including both the URL field's query parameters and the POST or PUT form data.
    FormValue(name string) string
    // FormValues returns the parsed form data, including both the URL
    // field's query parameters and the POST or PUT form data.
    //
    // The default form's memory maximum size is 32MB, it can be changed by the
    // `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
    //
    // NOTE: A check for nil is necessary.
    FormValues() map[string][]string

    // PostValueDefault returns the parsed form data from POST, PATCH,
    // or PUT body parameters based on a "name".
    //
    // If not found then "def" is returned instead.
    PostValueDefault(name string, def string) string
    // PostValue returns the parsed form data from POST, PATCH,
    // or PUT body parameters based on a "name"
    PostValue(name string) string
    // PostValueTrim returns the parsed form data from POST, PATCH,
    // or PUT body parameters based on a "name",  without trailing spaces.
    PostValueTrim(name string) string
    // PostValueInt returns the parsed form data from POST, PATCH,
    // or PUT body parameters based on a "name", as int.
    //
    // If not found returns -1 and a non-nil error.
    PostValueInt(name string) (int, error)
    // PostValueIntDefault returns the parsed form data from POST, PATCH,
    // or PUT body parameters based on a "name", as int.
    //
    // If not found returns or parse errors the "def".
    PostValueIntDefault(name string, def int) int
    // PostValueInt64 returns the parsed form data from POST, PATCH,
    // or PUT body parameters based on a "name", as float64.
    //
    // If not found returns -1 and a no-nil error.
    PostValueInt64(name string) (int64, error)
    // PostValueInt64Default returns the parsed form data from POST, PATCH,
    // or PUT body parameters based on a "name", as int64.
    //
    // If not found or parse errors returns the "def".
    PostValueInt64Default(name string, def int64) int64
    // PostValueInt64Default returns the parsed form data from POST, PATCH,
    // or PUT body parameters based on a "name", as float64.
    //
    // If not found returns -1 and a non-nil error.
    PostValueFloat64(name string) (float64, error)
    // PostValueInt64Default returns the parsed form data from POST, PATCH,
    // or PUT body parameters based on a "name", as float64.
    //
    // If not found or parse errors returns the "def".
    PostValueFloat64Default(name string, def float64) float64
    // PostValueInt64Default returns the parsed form data from POST, PATCH,
    // or PUT body parameters based on a "name", as bool.
    //
    // If not found or value is false, then it returns false, otherwise true.
    PostValueBool(name string) (bool, error)
    // PostValues returns all the parsed form data from POST, PATCH,
    // or PUT body parameters based on a "name" as a string slice.
    //
    // The default form's memory maximum size is 32MB, it can be changed by the
    // `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
    PostValues(name string) []string
    // FormFile returns the first uploaded file that received from the client.
    //
    // The default form's memory maximum size is 32MB, it can be changed by the
    //  `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
    //
    // Example: https://github.com/kataras/iris/tree/master/_examples/http_request/upload-file
    FormFile(key string) (multipart.File, *multipart.FileHeader, error)
    // UploadFormFiles uploads any received file(s) from the client
    // to the system physical location "destDirectory".
    //
    // The second optional argument "before" gives caller the chance to
    // modify the *miltipart.FileHeader before saving to the disk,
    // it can be used to change a file's name based on the current request,
    // all FileHeader's options can be changed. You can ignore it if
    // you don't need to use this capability before saving a file to the disk.
    //
    // Note that it doesn't check if request body streamed.
    //
    // Returns the copied length as int64 and
    // a not nil error if at least one new file
    // can't be created due to the operating system's permissions or
    // http.ErrMissingFile if no file received.
    //
    // If you want to receive & accept files and manage them manually you can use the `context#FormFile`
    // instead and create a copy function that suits your needs, the below is for generic usage.
    //
    // The default form's memory maximum size is 32MB, it can be changed by the
    //  `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
    //
    // See `FormFile` to a more controlled to receive a file.
    //
    //
    // Example: https://github.com/kataras/iris/tree/master/_examples/http_request/upload-files
    UploadFormFiles(destDirectory string, before ...func(Context, *multipart.FileHeader)) (n int64, err error)

    //  +------------------------------------------------------------+
    //  | Custom HTTP Errors                                         |
    //  +------------------------------------------------------------+

    // NotFound emits an error 404 to the client, using the specific custom error error handler.
    // Note that you may need to call ctx.StopExecution() if you don't want the next handlers
    // to be executed. Next handlers are being executed on iris because you can alt the
    // error code and change it to a more specific one, i.e
    // users := app.Party("/users")
    // users.Done(func(ctx context.Context){ if ctx.StatusCode() == 400 { /*  custom error code for /users */ }})
    NotFound()

    //  +------------------------------------------------------------+
    //  | Body Readers                                               |
    //  +------------------------------------------------------------+

    // SetMaxRequestBodySize sets a limit to the request body size
    // should be called before reading the request body from the client.
    SetMaxRequestBodySize(limitOverBytes int64)

    // UnmarshalBody reads the request's body and binds it to a value or pointer of any type.
    // Examples of usage: context.ReadJSON, context.ReadXML.
    //
    // Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-custom-via-unmarshaler/main.go
    //
    // UnmarshalBody does not check about gzipped data.
    // Do not rely on compressed data incoming to your server. The main reason is: https://en.wikipedia.org/wiki/Zip_bomb
    // However you are still free to read the `ctx.Request().Body io.Reader` manually.
    UnmarshalBody(outPtr interface{}, unmarshaler Unmarshaler) error
    // ReadJSON reads JSON from request's body and binds it to a pointer of a value of any json-valid type.
    //
    // Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-json/main.go
    ReadJSON(jsonObjectPtr interface{}) error
    // ReadXML reads XML from request's body and binds it to a pointer of a value of any xml-valid type.
    //
    // Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-xml/main.go
    ReadXML(xmlObjectPtr interface{}) error
    // ReadForm binds the formObject  with the form data
    // it supports any kind of type, including custom structs.
    // It will return nothing if request data are empty.
    //
    // Example: https://github.com/kataras/iris/blob/master/_examples/http_request/read-form/main.go
    ReadForm(formObjectPtr interface{}) error

    //  +------------------------------------------------------------+
    //  | Body (raw) Writers                                         |
    //  +------------------------------------------------------------+

    // Write writes the data to the connection as part of an HTTP reply.
    //
    // If WriteHeader has not yet been called, Write calls
    // WriteHeader(http.StatusOK) before writing the data. If the Header
    // does not contain a Content-Type line, Write adds a Content-Type set
    // to the result of passing the initial 512 bytes of written data to
    // DetectContentType.
    //
    // Depending on the HTTP protocol version and the client, calling
    // Write or WriteHeader may prevent future reads on the
    // Request.Body. For HTTP/1.x requests, handlers should read any
    // needed request body data before writing the response. Once the
    // headers have been flushed (due to either an explicit Flusher.Flush
    // call or writing enough data to trigger a flush), the request body
    // may be unavailable. For HTTP/2 requests, the Go HTTP server permits
    // handlers to continue to read the request body while concurrently
    // writing the response. However, such behavior may not be supported
    // by all HTTP/2 clients. Handlers should read before writing if
    // possible to maximize compatibility.
    Write(body []byte) (int, error)
    // Writef formats according to a format specifier and writes to the response.
    //
    // Returns the number of bytes written and any write error encountered.
    Writef(format string, args ...interface{}) (int, error)
    // WriteString writes a simple string to the response.
    //
    // Returns the number of bytes written and any write error encountered.
    WriteString(body string) (int, error)

    // SetLastModified sets the "Last-Modified" based on the "modtime" input.
    // If "modtime" is zero then it does nothing.
    //
    // It's mostly internally on core/router and context packages.
    //
    // Note that modtime.UTC() is being used instead of just modtime, so
    // you don't have to know the internals in order to make that works.
    SetLastModified(modtime time.Time)
    // CheckIfModifiedSince checks if the response is modified since the "modtime".
    // Note that it has nothing to do with server-side caching.
    // It does those checks by checking if the "If-Modified-Since" request header
    // sent by client or a previous server response header
    // (e.g with WriteWithExpiration or StaticEmbedded or Favicon etc.)
    // is a valid one and it's before the "modtime".
    //
    // A check for !modtime && err == nil is necessary to make sure that
    // it's not modified since, because it may return false but without even
    // had the chance to check the client-side (request) header due to some errors,
    // like the HTTP Method is not "GET" or "HEAD" or if the "modtime" is zero
    // or if parsing time from the header failed.
    //
    // It's mostly used internally, e.g. `context#WriteWithExpiration`.
    //
    // Note that modtime.UTC() is being used instead of just modtime, so
    // you don't have to know the internals in order to make that works.
    CheckIfModifiedSince(modtime time.Time) (bool, error)
    // WriteNotModified sends a 304 "Not Modified" status code to the client,
    // it makes sure that the content type, the content length headers
    // and any "ETag" are removed before the response sent.
    //
    // It's mostly used internally on core/router/fs.go and context methods.
    WriteNotModified()
    // WriteWithExpiration like Write but it sends with an expiration datetime
    // which is refreshed every package-level `StaticCacheDuration` field.
    WriteWithExpiration(body []byte, modtime time.Time) (int, error)
    // StreamWriter registers the given stream writer for populating
    // response body.
    //
    // Access to context's and/or its' members is forbidden from writer.
    //
    // This function may be used in the following cases:
    //
    //     * if response body is too big (more than iris.LimitRequestBodySize(if setted)).
    //     * if response body is streamed from slow external sources.
    //     * if response body must be streamed to the client in chunks.
    //     (aka `http server push`).
    //
    // receives a function which receives the response writer
    // and returns false when it should stop writing, otherwise true in order to continue
    StreamWriter(writer func(w io.Writer) bool)

    //  +------------------------------------------------------------+
    //  | Body Writers with compression                              |
    //  +------------------------------------------------------------+
    // ClientSupportsGzip retruns true if the client supports gzip compression.
    ClientSupportsGzip() bool
    // WriteGzip accepts bytes, which are compressed to gzip format and sent to the client.
    // returns the number of bytes written and an error ( if the client doesn' supports gzip compression)
    // You may re-use this function in the same handler
    // to write more data many times without any troubles.
    WriteGzip(b []byte) (int, error)
    // TryWriteGzip accepts bytes, which are compressed to gzip format and sent to the client.
    // If client does not supprots gzip then the contents are written as they are, uncompressed.
    TryWriteGzip(b []byte) (int, error)
    // GzipResponseWriter converts the current response writer into a response writer
    // which when its .Write called it compress the data to gzip and writes them to the client.
    //
    // Can be also disabled with its .Disable and .ResetBody to rollback to the usual response writer.
    GzipResponseWriter() *GzipResponseWriter
    // Gzip enables or disables (if enabled before) the gzip response writer,if the client
    // supports gzip compression, so the following response data will
    // be sent as compressed gzip data to the client.
    Gzip(enable bool)

    //  +------------------------------------------------------------+
    //  | Rich Body Content Writers/Renderers                        |
    //  +------------------------------------------------------------+

    // ViewLayout sets the "layout" option if and when .View
    // is being called afterwards, in the same request.
    // Useful when need to set or/and change a layout based on the previous handlers in the chain.
    //
    // Note that the 'layoutTmplFile' argument can be setted to iris.NoLayout || view.NoLayout
    // to disable the layout for a specific view render action,
    // it disables the engine's configuration's layout property.
    //
    // Look .ViewData and .View too.
    //
    // Example: https://github.com/kataras/iris/tree/master/_examples/view/context-view-data/
    ViewLayout(layoutTmplFile string)
    // ViewData saves one or more key-value pair in order to be passed if and when .View
    // is being called afterwards, in the same request.
    // Useful when need to set or/and change template data from previous hanadlers in the chain.
    //
    // If .View's "binding" argument is not nil and it's not a type of map
    // then these data are being ignored, binding has the priority, so the main route's handler can still decide.
    // If binding is a map or context.Map then these data are being added to the view data
    // and passed to the template.
    //
    // After .View, the data are not destroyed, in order to be re-used if needed (again, in the same request as everything else),
    // to clear the view data, developers can call:
    // ctx.Set(ctx.Application().ConfigurationReadOnly().GetViewDataContextKey(), nil)
    //
    // If 'key' is empty then the value is added as it's (struct or map) and developer is unable to add other value.
    //
    // Look .ViewLayout and .View too.
    //
    // Example: https://github.com/kataras/iris/tree/master/_examples/view/context-view-data/
    ViewData(key string, value interface{})
    // GetViewData returns the values registered by `context#ViewData`.
    // The return value is `map[string]interface{}`, this means that
    // if a custom struct registered to ViewData then this function
    // will try to parse it to map, if failed then the return value is nil
    // A check for nil is always a good practise if different
    // kind of values or no data are registered via `ViewData`.
    //
    // Similarly to `viewData := ctx.Values().Get("iris.viewData")` or
    // `viewData := ctx.Values().Get(ctx.Application().ConfigurationReadOnly().GetViewDataContextKey())`.
    GetViewData() map[string]interface{}
    // View renders a template based on the registered view engine(s).
    // First argument accepts the filename, relative to the view engine's Directory and Extension,
    // i.e: if directory is "./templates" and want to render the "./templates/users/index.html"
    // then you pass the "users/index.html" as the filename argument.
    //
    // The second optional argument can receive a single "view model"
    // that will be binded to the view template if it's not nil,
    // otherwise it will check for previous view data stored by the `ViewData`
    // even if stored at any previous handler(middleware) for the same request.
    //
    // Look .ViewData` and .ViewLayout too.
    //
    // Examples: https://github.com/kataras/iris/tree/master/_examples/view
    View(filename string, optionalViewModel ...interface{}) error

    // Binary writes out the raw bytes as binary data.
    Binary(data []byte) (int, error)
    // Text writes out a string as plain text.
    Text(text string) (int, error)
    // HTML writes out a string as text/html.
    HTML(htmlContents string) (int, error)
    // JSON marshals the given interface object and writes the JSON response.
    JSON(v interface{}, options ...JSON) (int, error)
    // JSONP marshals the given interface object and writes the JSON response.
    JSONP(v interface{}, options ...JSONP) (int, error)
    // XML marshals the given interface object and writes the XML response.
    XML(v interface{}, options ...XML) (int, error)
    // Markdown parses the markdown to html and renders its result to the client.
    Markdown(markdownB []byte, options ...Markdown) (int, error)
    // YAML parses the "v" using the yaml parser and renders its result to the client.
    YAML(v interface{}) (int, error)
    //  +------------------------------------------------------------+
    //  | Serve files                                                |
    //  +------------------------------------------------------------+

    // ServeContent serves content, headers are autoset
    // receives three parameters, it's low-level function, instead you can use .ServeFile(string,bool)/SendFile(string,string)
    //
    //
    // You can define your own "Content-Type" with `context#ContentType`, before this function call.
    //
    // This function doesn't support resuming (by range),
    // use ctx.SendFile or router's `StaticWeb` instead.
    ServeContent(content io.ReadSeeker, filename string, modtime time.Time, gzipCompression bool) error
    // ServeFile serves a file (to send a file, a zip for example to the client you should use the `SendFile` instead)
    // receives two parameters
    // filename/path (string)
    // gzipCompression (bool)
    //
    // You can define your own "Content-Type" with `context#ContentType`, before this function call.
    //
    // This function doesn't support resuming (by range),
    // use ctx.SendFile or router's `StaticWeb` instead.
    //
    // Use it when you want to serve dynamic files to the client.
    ServeFile(filename string, gzipCompression bool) error
    // SendFile sends file for force-download to the client
    //
    // Use this instead of ServeFile to 'force-download' bigger files to the client.
    SendFile(filename string, destinationName string) error

    //  +------------------------------------------------------------+
    //  | Cookies                                                    |
    //  +------------------------------------------------------------+

    // SetCookie adds a cookie.
    // Use of the "options" is not required, they can be used to amend the "cookie".
    //
    // Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
    SetCookie(cookie *http.Cookie, options ...CookieOption)
    // SetCookieKV adds a cookie, requires the name(string) and the value(string).
    //
    // By default it expires at 2 hours and it's added to the root path,
    // use the `CookieExpires` and `CookiePath` to modify them.
    // Alternatively: ctx.SetCookie(&http.Cookie{...})
    //
    // If you want to set custom the path:
    // ctx.SetCookieKV(name, value, iris.CookiePath("/custom/path/cookie/will/be/stored"))
    //
    // If you want to be visible only to current request path:
    // ctx.SetCookieKV(name, value, iris.CookieCleanPath/iris.CookiePath(""))
    // More:
    //                              iris.CookieExpires(time.Duration)
    //                              iris.CookieHTTPOnly(false)
    //
    // Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
    SetCookieKV(name, value string, options ...CookieOption)
    // GetCookie returns cookie's value by it's name
    // returns empty string if nothing was found.
    //
    // If you want more than the value then:
    // cookie, err := ctx.Request().Cookie("name")
    //
    // Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
    GetCookie(name string, options ...CookieOption) string
    // RemoveCookie deletes a cookie by it's name and path = "/".
    // Tip: change the cookie's path to the current one by: RemoveCookie("name", iris.CookieCleanPath)
    //
    // Example: https://github.com/kataras/iris/tree/master/_examples/cookies/basic
    RemoveCookie(name string, options ...CookieOption)
    // VisitAllCookies takes a visitor which loops
    // on each (request's) cookies' name and value.
    VisitAllCookies(visitor func(name string, value string))

    // MaxAge returns the "cache-control" request header's value
    // seconds as int64
    // if header not found or parse failed then it returns -1.
    MaxAge() int64

    //  +------------------------------------------------------------+
    //  | Advanced: Response Recorder and Transactions               |
    //  +------------------------------------------------------------+

    // Record transforms the context's basic and direct responseWriter to a ResponseRecorder
    // which can be used to reset the body, reset headers, get the body,
    // get & set the status code at any time and more.
    Record()
    // Recorder returns the context's ResponseRecorder
    // if not recording then it starts recording and returns the new context's ResponseRecorder
    Recorder() *ResponseRecorder
    // IsRecording returns the response recorder and a true value
    // when the response writer is recording the status code, body, headers and so on,
    // else returns nil and false.
    IsRecording() (*ResponseRecorder, bool)

    // BeginTransaction starts a scoped transaction.
    //
    // You can search third-party articles or books on how Business Transaction works (it's quite simple, especially here).
    //
    // Note that this is unique and new
    // (=I haver never seen any other examples or code in Golang on this subject, so far, as with the most of iris features...)
    // it's not covers all paths,
    // such as databases, this should be managed by the libraries you use to make your database connection,
    // this transaction scope is only for context's response.
    // Transactions have their own middleware ecosystem also, look iris.go:UseTransaction.
    //
    // See https://github.com/kataras/iris/tree/master/_examples/ for more
    BeginTransaction(pipe func(t *Transaction))
    // SkipTransactions if called then skip the rest of the transactions
    // or all of them if called before the first transaction
    SkipTransactions()
    // TransactionsSkipped returns true if the transactions skipped or canceled at all.
    TransactionsSkipped() bool

    // Exec calls the `context/Application#ServeCtx`
    // based on this context but with a changed method and path
    // like it was requested by the user, but it is not.
    //
    // Offline means that the route is registered to the iris and have all features that a normal route has
    // BUT it isn't available by browsing, its handlers executed only when other handler's context call them
    // it can validate paths, has sessions, path parameters and all.
    //
    // You can find the Route by app.GetRoute("theRouteName")
    // you can set a route name as: myRoute := app.Get("/mypath", handler)("theRouteName")
    // that will set a name to the route and returns its RouteInfo instance for further usage.
    //
    // It doesn't changes the global state, if a route was "offline" it remains offline.
    //
    // app.None(...) and app.GetRoutes().Offline(route)/.Online(route, method)
    //
    // Example: https://github.com/kataras/iris/tree/master/_examples/routing/route-state
    //
    // User can get the response by simple using rec := ctx.Recorder(); rec.Body()/rec.StatusCode()/rec.Header().
    //
    // Context's Values and the Session are kept in order to be able to communicate via the result route.
    //
    // It's for extreme use cases, 99% of the times will never be useful for you.
    Exec(method, path string)

    // RouteExists reports whether a particular route exists
    // It will search from the current subdomain of context's host, if not inside the root domain.
    RouteExists(method, path string) bool

    // Application returns the iris app instance which belongs to this context.
    // Worth to notice that this function returns an interface
    // of the Application, which contains methods that are safe
    // to be executed at serve-time. The full app's fields
    // and methods are not available here for the developer's safety.
    Application() Application

    // String returns the string representation of this request.
    // Each context has a unique string representation.
    // It can be used for simple debugging scenarios, i.e print context as string.
    //
    // What it returns? A number which declares the length of the
    // total `String` calls per executable application, followed
    // by the remote IP (the client) and finally the method:url.
    String() string
}
```