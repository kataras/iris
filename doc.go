// Copyright (c) 2017-2019 The Iris Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Iris nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

/*
Package iris implements the highest realistic performance, easy to learn Go web framework.
Iris provides a beautifully expressive and easy to use foundation for your next website, API, or distributed app.
Low-level handlers compatible with `net/http` and high-level fastest MVC implementation and handlers dependency injection.
Easy to learn for new gophers and advanced features for experienced, it goes as far as you dive into it!

Source code and other details for the project are available at GitHub:

   https://github.com/kataras/iris

Current Version

11.2.0

Installation

The only requirement is the Go Programming Language, at least version 1.8 but 1.11.1 and above is highly recommended.

    $ go get -u github.com/kataras/iris


Example code:


    package main

    import "github.com/kataras/iris"

    // User is just a bindable object structure.
    type User struct {
        Username  string `json:"username"`
        Firstname string `json:"firstname"`
        Lastname  string `json:"lastname"`
        City      string `json:"city"`
        Age       int    `json:"age"`
    }

    func main() {
        app := iris.New()

        // Define templates using the std html/template engine.
        // Parse and load all files inside "./views" folder with ".html" file extension.
        // Reload the templates on each request (development mode).
        app.RegisterView(iris.HTML("./views", ".html").Reload(true))

        // Register custom handler for specific http errors.
        app.OnErrorCode(iris.StatusInternalServerError, func(ctx iris.Context) {
            // .Values are used to communicate between handlers, middleware.
            errMessage := ctx.Values().GetString("error")
            if errMessage != "" {
                ctx.Writef("Internal server error: %s", errMessage)
                return
            }

            ctx.Writef("(Unexpected) internal server error")
        })

        app.Use(func(ctx iris.Context) {
            ctx.Application().Logger().Infof("Begin request for path: %s", ctx.Path())
            ctx.Next()
        })

        // app.Done(func(ctx iris.Context) {})

        // Method POST: http://localhost:8080/decode
        app.Post("/decode", func(ctx iris.Context) {
            var user User
            ctx.ReadJSON(&user)
            ctx.Writef("%s %s is %d years old and comes from %s", user.Firstname, user.Lastname, user.Age, user.City)
        })

        // Method GET: http://localhost:8080/encode
        app.Get("/encode", func(ctx iris.Context) {
            doe := User{
                Username:  "Johndoe",
                Firstname: "John",
                Lastname:  "Doe",
                City:      "Neither FBI knows!!!",
                Age:       25,
            }

            ctx.JSON(doe)
        })

        // Method GET: http://localhost:8080/profile/anytypeofstring
        app.Get("/profile/{username:string}", profileByUsername)

        // Want to use a custom regex expression instead?
        // Easy: app.Get("/profile/{username:string regexp(^[a-zA-Z ]+$)}")
        //
        // If parameter type is missing then it's string which accepts anything,
        // i.e: /{paramname} it's exactly the same as /{paramname:string}.

        usersRoutes := app.Party("/users", logThisMiddleware)
        {
            // Method GET: http://localhost:8080/users/42
            usersRoutes.Get("/{id:uint64 min(1)}", getUserByID)
            // Method POST: http://localhost:8080/users/create
            usersRoutes.Post("/create", createUser)
        }

        // Listen for incoming HTTP/1.x & HTTP/2 clients on localhost port 8080.
        app.Run(iris.Addr(":8080"), iris.WithCharset("UTF-8"))
    }

    func logThisMiddleware(ctx iris.Context) {
        ctx.Application().Logger().Infof("Path: %s | IP: %s", ctx.Path(), ctx.RemoteAddr())

        // .Next is required to move forward to the chain of handlers,
        // if missing then it stops the execution at this handler.
        ctx.Next()
    }

    func profileByUsername(ctx iris.Context) {
        // .Params are used to get dynamic path parameters.
        username := ctx.Params().Get("username")
        ctx.ViewData("Username", username)
        // renders "./views/users/profile.html"
        // with {{ .Username }} equals to the username dynamic path parameter.
        ctx.View("users/profile.html")
    }

    func getUserByID(ctx iris.Context) {
        userID := ctx.Params().Get("id") // Or convert directly using: .Values().GetInt/GetUint64/GetInt64 etc...
        // your own db fetch here instead of user :=...
        user := User{Username: "username" + userID}

        ctx.XML(user)
    }

    func createUser(ctx iris.Context) {
        var user User
        err := ctx.ReadForm(&user)
        if err != nil {
            ctx.Values().Set("error", "creating user, read and parse form failed. "+err.Error())
            ctx.StatusCode(iris.StatusInternalServerError)
            return
        }
        // renders "./views/users/create_verification.html"
        // with {{ . }} equals to the User object, i.e {{ .Username }} , {{ .Firstname}} etc...
        ctx.ViewData("", user)
        ctx.View("users/create_verification.html")
    }

Listening and gracefully shutdown

You can start the server(s) listening to any type of `net.Listener` or even `http.Server` instance.
The method for initialization of the server should be passed at the end, via `Run` function.

Below you'll see some useful examples:


    // Listening on tcp with network address 0.0.0.0:8080
    app.Run(iris.Addr(":8080"))


    // Same as before but using a custom http.Server which may be in use somewhere else too
    app.Run(iris.Server(&http.Server{Addr:":8080"}))


    // Using a custom net.Listener
    l, err := net.Listen("tcp4", ":8080")
    if err != nil {
        panic(err)
    }
    app.Run(iris.Listener(l))


    // TLS using files
    app.Run(iris.TLS("127.0.0.1:443", "mycert.cert", "mykey.key"))


    // Automatic TLS
    app.Run(iris.AutoTLS(":443", "example.com", "admin@example.com"))


    // UNIX socket
    if errOs := os.Remove(socketFile); errOs != nil && !os.IsNotExist(errOs) {
        app.Logger().Fatal(errOs)
    }

    l, err := net.Listen("unix", socketFile)

    if err != nil {
        app.Logger().Fatal(err)
    }

    if err = os.Chmod(socketFile, mode); err != nil {
        app.Logger().Fatal(err)
    }

    app.Run(iris.Listener(l))

    // Using any func() error,
    // the responsibility of starting up a listener is up to you with this way,
    // for the sake of simplicity we will use the
    // ListenAndServe function of the `net/http` package.
    app.Run(iris.Raw(&http.Server{Addr:":8080"}).ListenAndServe)

UNIX and BSD hosts can take advantage of the reuse port feature.

Example code:


    package main

    import (
        // Package tcplisten provides customizable TCP net.Listener with various
        // performance-related options:
        //
        //   - SO_REUSEPORT. This option allows linear scaling server performance
        //     on multi-CPU servers.
        //     See https://www.nginx.com/blog/socket-sharding-nginx-release-1-9-1/ for details.
        //
        //   - TCP_DEFER_ACCEPT. This option expects the server reads from the accepted
        //     connection before writing to them.
        //
        //   - TCP_FASTOPEN. See https://lwn.net/Articles/508865/ for details.
        "github.com/valyala/tcplisten"

        "github.com/kataras/iris"
    )

    // $ go get github.com/valyala/tcplisten
    // $ go run main.go

    func main() {
        app := iris.New()

        app.Get("/", func(ctx iris.Context) {
            ctx.HTML("<b>Hello World!</b>")
        })

        listenerCfg := tcplisten.Config{
            ReusePort:   true,
            DeferAccept: true,
            FastOpen:    true,
        }

        l, err := listenerCfg.NewListener("tcp", ":8080")
        if err != nil {
            panic(err)
        }

        app.Run(iris.Listener(l))
    }

That's all with listening, you have the full control when you need it.

Let's continue by learning how to catch CONTROL+C/COMMAND+C or unix kill command and shutdown the server gracefully.

    Gracefully Shutdown on CONTROL+C/COMMAND+C or when kill command sent is ENABLED BY-DEFAULT.

In order to manually manage what to do when app is interrupted,
we have to disable the default behavior with the option `WithoutInterruptHandler`
and register a new interrupt handler (globally, across all possible hosts).


Example code:


    package main

    import (
        stdContext "context"
        "time"

        "github.com/kataras/iris"
    )


    func main() {
        app := iris.New()

        iris.RegisterOnInterrupt(func() {
            timeout := 5 * time.Second
            ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
            defer cancel()
            // close all hosts
            app.Shutdown(ctx)
        })

        app.Get("/", func(ctx iris.Context) {
            ctx.HTML(" <h1>hi, I just exist in order to see if the server is closed</h1>")
        })

        // http://localhost:8080
        app.Run(iris.Addr(":8080"), iris.WithoutInterruptHandler)
    }


Hosts

Access to all hosts that serve your application can be provided by
the `Application#Hosts` field, after the `Run` method.


But the most common scenario is that you may need access to the host before the `Run` method,
there are two ways of gain access to the host supervisor, read below.

First way is to use the `app.NewHost` to create a new host
and use one of its `Serve` or `Listen` functions
to start the application via the `iris#Raw` Runner.
Note that this way needs an extra import of the `net/http` package.

Example Code:


    h := app.NewHost(&http.Server{Addr:":8080"})
    h.RegisterOnShutdown(func(){
        println("terminate")
    })

    app.Run(iris.Raw(h.ListenAndServe))

Second, and probably easier way is to use the `host.Configurator`.

Note that this method requires an extra import statement of
"github.com/kataras/iris/core/host" when using go < 1.9,
if you're targeting on go1.9 then you can use the `iris#Supervisor`
and omit the extra host import.

All common `Runners` we saw earlier (`iris#Addr, iris#Listener, iris#Server, iris#TLS, iris#AutoTLS`)
accept a variadic argument of `host.Configurator`, there are just `func(*host.Supervisor)`.
Therefore the `Application` gives you the rights to modify the auto-created host supervisor through these.


Example Code:


    package main

    import (
        stdContext "context"
        "time"

        "github.com/kataras/iris"
        "github.com/kataras/iris/core/host"
    )

    func main() {
        app := iris.New()

        app.Get("/", func(ctx iris.Context) {
            ctx.HTML("<h1>Hello, try to refresh the page after ~10 secs</h1>")
        })

        app.Logger().Info("Wait 10 seconds and check your terminal again")
        // simulate a shutdown action here...
        go func() {
            <-time.After(10 * time.Second)
            timeout := 5 * time.Second
            ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
            defer cancel()
            // close all hosts, this will notify the callback we had register
            // inside the `configureHost` func.
            app.Shutdown(ctx)
        }()

        // start the server as usual, the only difference is that
        // we're adding a second (optional) function
        // to configure the just-created host supervisor.
        //
        // http://localhost:8080
        // wait 10 seconds and check your terminal.
        app.Run(iris.Addr(":8080", configureHost), iris.WithoutServerError(iris.ErrServerClosed))

    }

    func configureHost(su *host.Supervisor) {
        // here we have full access to the host that will be created
        // inside the `Run` function.
        //
        // we register a shutdown "event" callback
        su.RegisterOnShutdown(func() {
            println("terminate")
        })
        // su.RegisterOnError
        // su.RegisterOnServe
    }


Read more about listening and gracefully shutdown by navigating to:

    https://github.com/kataras/iris/tree/master/_examples/#http-listening


Routing

All HTTP methods are supported, developers can also register handlers for same paths for different methods.
The first parameter is the HTTP Method,
second parameter is the request path of the route,
third variadic parameter should contains one or more iris.Handler executed
by the registered order when a user requests for that specific resouce path from the server.

Example code:


    app := iris.New()

    app.Handle("GET", "/contact", func(ctx iris.Context) {
        ctx.HTML("<h1> Hello from /contact </h1>")
    })


In order to make things easier for the user, iris provides functions for all HTTP Methods.
The first parameter is the request path of the route,
second variadic parameter should contains one or more iris.Handler executed
by the registered order when a user requests for that specific resouce path from the server.

Example code:


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



Grouping Routes


A set of routes that are being groupped by path prefix can (optionally) share the same middleware handlers and template layout.
A group can have a nested group too.

`.Party` is being used to group routes, developers can declare an unlimited number of (nested) groups.


Example code:


    users := app.Party("/users", myAuthMiddlewareHandler)

    // http://myhost.com/users/42/profile
    users.Get("/{id:uint64}/profile", userProfileHandler)
    // http://myhost.com/users/messages/1
    users.Get("/inbox/{id:int}", userMessageHandler)


Custom HTTP Errors


iris developers are able to register their own handlers for http statuses like 404 not found, 500 internal server error and so on.

Example code:


    // when 404 then render the template $templatedir/errors/404.html
    app.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context){
        ctx.View("errors/404.html")
    })

    app.OnErrorCode(500, func(ctx iris.Context){
        // ...
    })

Basic HTTP API

With the help of iris's expressionist router you can build any form of API you desire, with
safety.

Example code:


    package main

    import "github.com/kataras/iris"

    func main() {
        app := iris.New()

        // registers a custom handler for 404 not found http (error) status code,
        // fires when route not found or manually by ctx.StatusCode(iris.StatusNotFound).
        app.OnErrorCode(iris.StatusNotFound, notFoundHandler)

        // GET -> HTTP Method
        // / -> Path
        // func(ctx iris.Context) -> The route's handler.
        //
        // Third receiver should contains the route's handler(s), they are executed by order.
        app.Handle("GET", "/", func(ctx iris.Context) {
            // navigate to the middle of $GOPATH/src/github.com/kataras/iris/context/context.go
            // to overview all context's method (there a lot of them, read that and you will learn how iris works too)
            ctx.HTML("Hello from " + ctx.Path()) // Hello from /
        })

        app.Get("/home", func(ctx iris.Context) {
            ctx.Writef(`Same as app.Handle("GET", "/", [...])`)
        })

        app.Get("/donate", donateHandler, donateFinishHandler)

        // Pssst, don't forget dynamic-path example for more "magic"!
        app.Get("/api/users/{userid:uint64 min(1)}", func(ctx iris.Context) {
            userID, err := ctx.Params().GetUint64("userid")

            if err != nil {
                ctx.Writef("error while trying to parse userid parameter," +
                    "this will never happen if :int is being used because if it's not integer it will fire Not Found automatically.")
                ctx.StatusCode(iris.StatusBadRequest)
                return
            }

            ctx.JSON(map[string]interface{}{
                // you can pass any custom structured go value of course.
                "user_id": userID,
            })
        })
        // app.Post("/", func(ctx iris.Context){}) -> for POST http method.
        // app.Put("/", func(ctx iris.Context){})-> for "PUT" http method.
        // app.Delete("/", func(ctx iris.Context){})-> for "DELETE" http method.
        // app.Options("/", func(ctx iris.Context){})-> for "OPTIONS" http method.
        // app.Trace("/", func(ctx iris.Context){})-> for "TRACE" http method.
        // app.Head("/", func(ctx iris.Context){})-> for "HEAD" http method.
        // app.Connect("/", func(ctx iris.Context){})-> for "CONNECT" http method.
        // app.Patch("/", func(ctx iris.Context){})-> for "PATCH" http method.
        // app.Any("/", func(ctx iris.Context){}) for all http methods.

        // More than one route can contain the same path with a different http mapped method.
        // You can catch any route creation errors with:
        // route, err := app.Get(...)
        // set a name to a route: route.Name = "myroute"

        // You can also group routes by path prefix, sharing middleware(s) and done handlers.

        adminRoutes := app.Party("/admin", adminMiddleware)

        adminRoutes.Done(func(ctx iris.Context) { // executes always last if ctx.Next()
            ctx.Application().Logger().Infof("response sent to " + ctx.Path())
        })
        // adminRoutes.Layout("/views/layouts/admin.html") // set a view layout for these routes, see more at view examples.

        // GET: http://localhost:8080/admin
        adminRoutes.Get("/", func(ctx iris.Context) {
            // [...]
            ctx.StatusCode(iris.StatusOK) // default is 200 == iris.StatusOK
            ctx.HTML("<h1>Hello from admin/</h1>")

            ctx.Next() // in order to execute the party's "Done" Handler(s)
        })

        // GET: http://localhost:8080/admin/login
        adminRoutes.Get("/login", func(ctx iris.Context) {
            // [...]
        })
        // POST: http://localhost:8080/admin/login
        adminRoutes.Post("/login", func(ctx iris.Context) {
            // [...]
        })

        // subdomains, easier than ever, should add localhost or 127.0.0.1 into your hosts file,
        // etc/hosts on unix or C:/windows/system32/drivers/etc/hosts on windows.
        v1 := app.Party("v1.")
        { // braces are optional, it's just type of style, to group the routes visually.

            // http://v1.localhost:8080
            v1.Get("/", func(ctx iris.Context) {
                ctx.HTML("Version 1 API. go to <a href='" + ctx.Path() + "/api" + "'>/api/users</a>")
            })

            usersAPI := v1.Party("/api/users")
            {
                // http://v1.localhost:8080/api/users
                usersAPI.Get("/", func(ctx iris.Context) {
                    ctx.Writef("All users")
                })
                // http://v1.localhost:8080/api/users/42
                usersAPI.Get("/{userid:uint64}", func(ctx iris.Context) {
                    ctx.Writef("user with id: %s", ctx.Params().GetUint64("userid"))
                })
            }
        }

        // wildcard subdomains.
        wildcardSubdomain := app.Party("*.")
        {
            wildcardSubdomain.Get("/", func(ctx iris.Context) {
                ctx.Writef("Subdomain can be anything, now you're here from: %s", ctx.Subdomain())
            })
        }

        // http://localhost:8080
        // http://localhost:8080/home
        // http://localhost:8080/donate
        // http://localhost:8080/api/users/42
        // http://localhost:8080/admin
        // http://localhost:8080/admin/login
        //
        // http://localhost:8080/api/users/0
        // http://localhost:8080/api/users/blabla
        // http://localhost:8080/wontfound
        //
        // if hosts edited:
        //  http://v1.localhost:8080
        //  http://v1.localhost:8080/api/users
        //  http://v1.localhost:8080/api/users/42
        //  http://anything.localhost:8080
        app.Run(iris.Addr(":8080"))
    }

    func adminMiddleware(ctx iris.Context) {
        // [...]
        ctx.Next() // to move to the next handler, or don't that if you have any auth logic.
    }

    func donateHandler(ctx iris.Context) {
        ctx.Writef("Just like an inline handler, but it can be " +
            "used by other package, anywhere in your project.")

        // let's pass a value to the next handler
        // Values is the way handlers(or middleware) are communicating between each other.
        ctx.Values().Set("donate_url", "https://github.com/kataras/iris#-people")
        ctx.Next() // in order to execute the next handler in the chain, look donate route.
    }

    func donateFinishHandler(ctx iris.Context) {
        // values can be any type of object so we could cast the value to a string
        // but iris provides an easy to do that, if donate_url is not defined, then it returns an empty string instead.
        donateURL := ctx.Values().GetString("donate_url")
        ctx.Application().Logger().Infof("donate_url value was: " + donateURL)
        ctx.Writef("\n\nDonate sent(?).")
    }

    func notFoundHandler(ctx iris.Context) {
        ctx.HTML("Custom route for 404 not found http code, here you can render a view, html, json <b>any valid response</b>.")
    }


Parameterized Path

At the previous example,
we've seen static routes, group of routes, subdomains, wildcard subdomains, a small example of parameterized path
with a single known parameter and custom http errors, now it's time to see wildcard parameters and macros.

iris, like net/http std package registers route's handlers
by a Handler, the iris' type of handler is just a func(ctx iris.Context)
where context comes from github.com/kataras/iris/context.

Iris has the easiest and the most powerful routing process you have ever meet.

At the same time,
iris has its own interpeter(yes like a programming language)
for route's path syntax and their dynamic path parameters parsing and evaluation,
We call them "macros" for shortcut.
How? It calculates its needs and if not any special regexp needed then it just
registers the route with the low-level path syntax,
otherwise it pre-compiles the regexp and adds the necessary middleware(s).

Standard macro types for parameters:

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
    i.e: "/test/{param:path}" and request: "/test/path1/path2/path3" , ctx.Params().Get("param") == "path1/path2/path3"

if type is missing then parameter's type is defaulted to string, so
{param} == {param:string}.

If a function not found on that type then the "string"'s types functions are being used.
i.e:


    {param:int min(3)}


Besides the fact that iris provides the basic types and some default "macro funcs"
you are able to register your own too!.

Register a named path parameter function:


    app.Macros().Get("int").RegisterFunc("min", func(argument int) func(paramValue int) bool {
        return func(paramValue int) bool {
            [...]
            return true/false -> true means valid.
        }
    })

at the func(argument ...) you can have any standard type, it will be validated before the server starts
so don't care about performance here, the only thing it runs at serve time is the returning func(paramValue string) bool.

    {param:string equal(iris)} , "iris" will be the argument here:
    app.Macros().Get("string").RegisterFunc("equal", func(argument string) func(paramValue string) bool {
        return func(paramValue string){ return argument == paramValue }
    })


Example Code:


	// you can use the "string" type which is valid for a single path parameter that can be anything.
	app.Get("/username/{name}", func(ctx iris.Context) {
		ctx.Writef("Hello %s", ctx.Params().Get("name"))
	}) // type is missing = {name:string}

	// Let's register our first macro attached to int macro type.
	// "min" = the function
	// "minValue" = the argument of the function
	// func(<T>) bool = the macro's path parameter evaluator, this executes in serve time when
	// a user requests a path which contains the int macro type with the min(...) macro parameter function.
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


Last, do not confuse ctx.Values() with ctx.Params().
Path parameter's values goes to ctx.Params() and context's local storage
that can be used to communicate between handlers and middleware(s) goes to
ctx.Values(), path parameters and the rest of any custom values are separated for your own good.

Run

  $ go run main.go



Static Files


    // StaticServe serves a directory as web resource
    // it's the simpliest form of the Static* functions
    // Almost same usage as StaticWeb
    // accepts only one required parameter which is the systemPath,
    // the same path will be used to register the GET and HEAD method routes.
    // If second parameter is empty, otherwise the requestPath is the second parameter
    // it uses gzip compression (compression on each request, no file cache).
    //
    // Returns the GET *Route.
    StaticServe(systemPath string, requestPath ...string) (*Route, error)

    // StaticContent registers a GET and HEAD method routes to the requestPath
    // that are ready to serve raw static bytes, memory cached.
    //
    // Returns the GET *Route.
    StaticContent(reqPath string, cType string, content []byte) (*Route, error)

    // StaticEmbedded  used when files are distributed inside the app executable, using go-bindata mostly
    // First parameter is the request path, the path which the files in the vdir will be served to, for example "/static"
    // Second parameter is the (virtual) directory path, for example "./assets"
    // Third parameter is the Asset function
    // Forth parameter is the AssetNames function.
    //
    // Returns the GET *Route.
    //
    // Example: https://github.com/kataras/iris/tree/master/_examples/file-server/embedding-files-into-app
    StaticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) (*Route, error)

    // Favicon serves static favicon
    // accepts 2 parameters, second is optional
    // favPath (string), declare the system directory path of the __.ico
    // requestPath (string), it's the route's path, by default this is the "/favicon.ico" because some browsers tries to get this by default first,
    // you can declare your own path if you have more than one favicon (desktop, mobile and so on)
    //
    // this func will add a route for you which will static serve the /yuorpath/yourfile.ico to the /yourfile.ico
    // (nothing special that you can't handle by yourself).
    // Note that you have to call it on every favicon you have to serve automatically (desktop, mobile and so on).
    //
    // Returns the GET *Route.
    Favicon(favPath string, requestPath ...string) (*Route, error)

    // StaticWeb returns a handler that serves HTTP requests
    // with the contents of the file system rooted at directory.
    //
    // first parameter: the route path
    // second parameter: the system directory
    // third OPTIONAL parameter: the exception routes
    //      (= give priority to these routes instead of the static handler)
    // for more options look app.StaticHandler.
    //
    //     app.StaticWeb("/static", "./static")
    //
    // As a special case, the returned file server redirects any request
    // ending in "/index.html" to the same path, without the final
    // "index.html".
    //
    // StaticWeb calls the StaticHandler(systemPath, listingDirectories: false, gzip: false ).
    //
    // Returns the GET *Route.
    StaticWeb(requestPath string, systemPath string, exceptRoutes ...*Route) (*Route, error)


Example code:


    package main

    import "github.com/kataras/iris"

    func main() {
        app := iris.New()

        // This will serve the ./static/favicons/ion_32_32.ico to: localhost:8080/favicon.ico
        app.Favicon("./static/favicons/ion_32_32.ico")

        // app.Favicon("./static/favicons/ion_32_32.ico", "/favicon_48_48.ico")
        // This will serve the ./static/favicons/ion_32_32.ico to: localhost:8080/favicon_48_48.ico

        app.Get("/", func(ctx iris.Context) {
            ctx.HTML(`<a href="/favicon.ico"> press here to see the favicon.ico</a>.
            At some browsers like chrome, it should be visible at the top-left side of the browser's window,
            because some browsers make requests to the /favicon.ico automatically,
            so iris serves your favicon in that path too (you can change it).`)
        }) // if favicon doesn't show to you, try to clear your browser's cache.

        app.Run(iris.Addr(":8080"))
    }

More examples can be found here: https://github.com/kataras/iris/tree/master/_examples/beginner/file-server


Middleware Ecosystem

Middleware is just a concept of ordered chain of handlers.
Middleware can be registered globally, per-party, per-subdomain and per-route.


Example code:

      // globally
      // before any routes, appends the middleware to all routes
      app.Use(func(ctx iris.Context){
         // ... any code here

         ctx.Next() // in order to continue to the next handler,
         // if that is missing then the next in chain handlers will be not executed,
         // useful for authentication middleware
      })

      // globally
      // after or before any routes, prepends the middleware to all routes
      app.UseGlobal(handler1, handler2, handler3)

      // per-route
      app.Post("/login", authenticationHandler, loginPageHandler)

      // per-party(group of routes)
      users := app.Party("/users", usersMiddleware)
      users.Get("/", usersIndex)

      // per-subdomain
      mysubdomain := app.Party("mysubdomain.", firstMiddleware)
      mysubdomain.Use(secondMiddleware)
      mysubdomain.Get("/", mysubdomainIndex)

      // per wildcard, dynamic subdomain
      dynamicSub := app.Party(".*", firstMiddleware, secondMiddleware)
      dynamicSub.Get("/", func(ctx iris.Context){
        ctx.Writef("Hello from subdomain: "+ ctx.Subdomain())
      })


iris is able to wrap and convert any external, third-party Handler you used to use to your web application.
Let's convert the https://github.com/rs/cors net/http external middleware which returns a `next form` handler.


Example code:

    package main

    import (
        "github.com/rs/cors"

        "github.com/kataras/iris"
    )

    func main() {

        app := iris.New()
        corsOptions := cors.Options{
            AllowedOrigins:   []string{"*"},
            AllowCredentials: true,
        }

        corsWrapper := cors.New(corsOptions).ServeHTTP

        app.WrapRouter(corsWrapper)

        v1 := app.Party("/api/v1")
        {
            v1.Get("/", h)
            v1.Put("/put", h)
            v1.Post("/post", h)
        }

        app.Run(iris.Addr(":8080"))
    }

    func h(ctx iris.Context) {
        ctx.Application().Logger().Infof(ctx.Path())
        ctx.Writef("Hello from %s", ctx.Path())
    }


View Engine


Iris supports 5 template engines out-of-the-box, developers can still use any external golang template engine,
as `context/context#ResponseWriter()` is an `io.Writer`.

All of these five template engines have common features with common API,
like Layout, Template Funcs, Party-specific layout, partial rendering and more.

      The standard html,
      its template parser is the golang.org/pkg/html/template/

      Django,
      its template parser is the github.com/flosch/pongo2

      Pug(Jade),
      its template parser is the github.com/Joker/jade

      Handlebars,
      its template parser is the github.com/aymerick/raymond

      Amber,
      its template parser is the github.com/eknkc/amber


Example code:

    package main

    import "github.com/kataras/iris"

    func main() {
        app := iris.New()

        // - standard html  | iris.HTML(...)
        // - django         | iris.Django(...)
        // - pug(jade)      | iris.Pug(...)
        // - handlebars     | iris.Handlebars(...)
        // - amber          | iris.Amber(...)

        tmpl := iris.HTML("./templates", ".html")
        tmpl.Reload(true) // reload templates on each request (development mode)
        // default template funcs are:
        //
        // - {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
        // - {{ render "header.html" }}
        // - {{ render_r "header.html" }} // partial relative path to current page
        // - {{ yield }}
        // - {{ current }}

        // register a custom template func.
        tmpl.AddFunc("greet", func(s string) string {
            return "Greetings " + s + "!"
        })

        // register the view engine to the views, this will load the templates.
        app.RegisterView(tmpl)

        app.Get("/", hi)

        // http://localhost:8080
        app.Run(iris.Addr(":8080"), iris.WithCharset("UTF-8")) // defaults to that but you can change it.
    }

    func hi(ctx iris.Context) {
        ctx.ViewData("Title", "Hi Page")
        ctx.ViewData("Name", "iris") // {{.Name}} will render: iris
        // ctx.ViewData("", myCcustomStruct{})
        ctx.View("hi.html")
    }



View engine supports bundled(https://github.com/shuLhan/go-bindata) template files too.
go-bindata gives you two functions, asset and assetNames,
these can be setted to each of the template engines using the `.Binary` func.

Example code:

    package main

    import "github.com/kataras/iris"

    func main() {
        app := iris.New()
        // $ go get -u github.com/shuLhan/go-bindata/...
        // $ go-bindata ./templates/...
        // $ go build
        // $ ./embedding-templates-into-app
        // html files are not used, you can delete the folder and run the example
        app.RegisterView(iris.HTML("./templates", ".html").Binary(Asset, AssetNames))
        app.Get("/", hi)

        // http://localhost:8080
        app.Run(iris.Addr(":8080"))
    }

    type page struct {
        Title, Name string
    }

    func hi(ctx iris.Context) {
        ctx.ViewData("", page{Title: "Hi Page", Name: "iris"})
        ctx.View("hi.html")
    }


A real example can be found here: https://github.com/kataras/iris/tree/master/_examples/view/embedding-templates-into-app.

Enable auto-reloading of templates on each request. Useful while developers are in dev mode
as they no neeed to restart their app on every template edit.

Example code:


    pugEngine := iris.Pug("./templates", ".jade")
    pugEngine.Reload(true) // <--- set to true to re-build the templates on each request.
    app.RegisterView(pugEngine)

Note:

In case you're wondering, the code behind the view engines derives from the "github.com/kataras/iris/view" package,
access to the engines' variables can be granded by "github.com/kataras/iris" package too.

    iris.HTML(...) is a shortcut of view.HTML(...)
    iris.Django(...)     >> >>      view.Django(...)
    iris.Pug(...)        >> >>      view.Pug(...)
    iris.Handlebars(...) >> >>      view.Handlebars(...)
    iris.Amber(...)      >> >>      view.Amber(...)

Each one of these template engines has different options located here: https://github.com/kataras/iris/tree/master/view .


Sessions


This example will show how to store and access data from a session.

You don’t need any third-party library,
but If you want you can use any session manager compatible or not.

In this example we will only allow authenticated users to view our secret message on the /secret page.
To get access to it, the will first have to visit /login to get a valid session cookie,
which logs him in. Additionally he can visit /logout to revoke his access to our secret message.


Example code:


    // main.go
    package main

    import (
        "github.com/kataras/iris"

        "github.com/kataras/iris/sessions"
    )

    var (
        cookieNameForSessionID = "mycookiesessionnameid"
        sess                   = sessions.New(sessions.Config{Cookie: cookieNameForSessionID})
    )

    func secret(ctx iris.Context) {

        // Check if user is authenticated
        if auth, _ := sess.Start(ctx).GetBoolean("authenticated"); !auth {
            ctx.StatusCode(iris.StatusForbidden)
            return
        }

        // Print secret message
        ctx.WriteString("The cake is a lie!")
    }

    func login(ctx iris.Context) {
        session := sess.Start(ctx)

        // Authentication goes here
        // ...

        // Set user as authenticated
        session.Set("authenticated", true)
    }

    func logout(ctx iris.Context) {
        session := sess.Start(ctx)

        // Revoke users authentication
        session.Set("authenticated", false)
    }

    func main() {
        app := iris.New()

        app.Get("/secret", secret)
        app.Get("/login", login)
        app.Get("/logout", logout)

        app.Run(iris.Addr(":8080"))
    }


Running the example:


    $ go get github.com/kataras/iris/sessions
    $ go run main.go

    $ curl -s http://localhost:8080/secret
    Forbidden

    $ curl -s -I http://localhost:8080/login
    Set-Cookie: mycookiesessionnameid=MTQ4NzE5Mz...

    $ curl -s --cookie "mycookiesessionnameid=MTQ4NzE5Mz..." http://localhost:8080/secret
    The cake is a lie!


Sessions persistence can be achieved using one (or more) `sessiondb`.

Example Code:

    package main

    import (
        "time"

        "github.com/kataras/iris"

        "github.com/kataras/iris/sessions"
        "github.com/kataras/iris/sessions/sessiondb/redis"
        "github.com/kataras/iris/sessions/sessiondb/redis/service"
    )

    // tested with redis version 3.0.503.
    // for windows see: https://github.com/ServiceStack/redis-windows
    func main() {
        // replace with your running redis' server settings:
        db := redis.New(service.Config{
            Network:     service.DefaultRedisNetwork,
            Addr:        service.DefaultRedisAddr,
            Password:    "",
            Database:    "",
            MaxIdle:     0,
            MaxActive:   0,
            IdleTimeout: service.DefaultRedisIdleTimeout,
            Prefix:      ""}) // optionally configure the bridge between your redis server

        // close connection when control+C/cmd+C
        iris.RegisterOnInterrupt(func() {
            db.Close()
        })

        defer db.Close() // close the database connection if application errored.

        sess := sessions.New(sessions.Config{
            Cookie:  "sessionscookieid",
            Expires: 45 * time.Minute}, // <=0 means unlimited life. Defaults to 0.
        )

        //
        // IMPORTANT:
        //
        sess.UseDatabase(db)

        // the rest of the code stays the same.
        app := iris.New()

        app.Get("/", func(ctx iris.Context) {
            ctx.Writef("You should navigate to the /set, /get, /delete, /clear,/destroy instead")
        })
        app.Get("/set", func(ctx iris.Context) {
            s := sess.Start(ctx)
            //set session values
            s.Set("name", "iris")

            //test if setted here
            ctx.Writef("All ok session value of the 'name' is: %s", s.GetString("name"))
        })

        app.Get("/set/{key}/{value}", func(ctx iris.Context) {
            key, value := ctx.Params().Get("key"), ctx.Params().Get("value")
            s := sess.Start(ctx)
            // set session values
            s.Set(key, value)

            // test if setted here
            ctx.Writef("All ok session value of the '%s' is: %s", key, s.GetString(key))
        })

        app.Get("/get", func(ctx iris.Context) {
            // get a specific key, as string, if no found returns just an empty string
            name := sess.Start(ctx).GetString("name")

            ctx.Writef("The 'name' on the /set was: %s", name)
        })

        app.Get("/get/{key}", func(ctx iris.Context) {
            // get a specific key, as string, if no found returns just an empty string
            name := sess.Start(ctx).GetString(ctx.Params().Get("key"))

            ctx.Writef("The name on the /set was: %s", name)
        })

        app.Get("/delete", func(ctx iris.Context) {
            // delete a specific key
            sess.Start(ctx).Delete("name")
        })

        app.Get("/clear", func(ctx iris.Context) {
            // removes all entries
            sess.Start(ctx).Clear()
        })

        app.Get("/destroy", func(ctx iris.Context) {
            //destroy, removes the entire session data and cookie
            sess.Destroy(ctx)
        })

        app.Get("/update", func(ctx iris.Context) {
            // updates expire date with a new date
            sess.ShiftExpiration(ctx)
        })

        app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
    }


More examples:

    https://github.com/kataras/iris/tree/master/_examples/sessions


Websockets

In this example we will create a small chat between web sockets via browser.

Example Server Code:

    // main.go
    package main

    import (
        "fmt"

        "github.com/kataras/iris"

        "github.com/kataras/iris/websocket"
    )

    func main() {
        app := iris.New()

        app.Get("/", func(ctx iris.Context) {
            ctx.ServeFile("websockets.html", false) // second parameter: enable gzip?
        })

        setupWebsocket(app)

        // x2
        // http://localhost:8080
        // http://localhost:8080
        // write something, press submit, see the result.
        app.Run(iris.Addr(":8080"))
    }

    func setupWebsocket(app *iris.Application) {
        // create our echo websocket server
        ws := websocket.New(websocket.Config{
            ReadBufferSize:  1024,
            WriteBufferSize: 1024,
        })
        ws.OnConnection(handleConnection)

        // register the server on an endpoint.
        // see the inline javascript code i the websockets.html, this endpoint is used to connect to the server.
        app.Get("/echo", ws.Handler())

        // serve the javascript builtin client-side library,
        // see websockets.html script tags, this path is used.
        app.Any("/iris-ws.js", func(ctx iris.Context) {
            ctx.Write(websocket.ClientSource)
        })
    }

    func handleConnection(c websocket.Connection) {
        // Read events from browser
        c.On("chat", func(msg string) {
            // Print the message to the console, c.Context() is the iris's http context.
            fmt.Printf("%s sent: %s\n", c.Context().RemoteAddr(), msg)
            // Write message back to the client message owner:
            // c.Emit("chat", msg)
            c.To(websocket.Broadcast).Emit("chat", msg)
        })
    }

Example Client(javascript) Code:

    <!-- websockets.html -->
    <input id="input" type="text" />
    <button onclick="send()">Send</button>
    <pre id="output"></pre>
    <script src="/iris-ws.js"></script>
    <script>
        var input = document.getElementById("input");
        var output = document.getElementById("output");

        // Ws comes from the auto-served '/iris-ws.js'
        var socket = new Ws("ws://localhost:8080/echo");
        socket.OnConnect(function () {
            output.innerHTML += "Status: Connected\n";
        });

        socket.OnDisconnect(function () {
            output.innerHTML += "Status: Disconnected\n";
        });

        // read events from the server
        socket.On("chat", function (msg) {
            addMessage(msg)
        });

        function send() {
            addMessage("Me: "+input.value) // write ourselves
            socket.Emit("chat", input.value);// send chat event data to the websocket server
            input.value = ""; // clear the input
        }

        function addMessage(msg) {
            output.innerHTML += msg + "\n";
        }
    </script>


Running the example:


    $ go get github.com/kataras/iris/websocket
    $ go run main.go
    $ start http://localhost:8080


MVC - Model View Controller

Iris has first-class support for the MVC pattern, you'll not find
these stuff anywhere else in the Go world.

Example Code:

    package main

    import (
        "github.com/kataras/iris"
        "github.com/kataras/iris/mvc"

        "github.com/kataras/iris/middleware/logger"
        "github.com/kataras/iris/middleware/recover"
    )

    func main() {
        app := iris.New()
        // Optionally, add two builtin handlers
        // that can recover from any http-relative panics
        // and log the requests to the terminal.
        app.Use(recover.New())
        app.Use(logger.New())

        // Serve a controller based on the root Router, "/".
        mvc.New(app).Handle(new(ExampleController))

        // http://localhost:8080
        // http://localhost:8080/ping
        // http://localhost:8080/hello
        // http://localhost:8080/custom_path
        app.Run(iris.Addr(":8080"))
    }

    // ExampleController serves the "/", "/ping" and "/hello".
    type ExampleController struct{}

    // Get serves
    // Method:   GET
    // Resource: http://localhost:8080
    func (c *ExampleController) Get() mvc.Result {
        return mvc.Response{
            ContentType: "text/html",
            Text:        "<h1>Welcome</h1>",
        }
    }

    // GetPing serves
    // Method:   GET
    // Resource: http://localhost:8080/ping
    func (c *ExampleController) GetPing() string {
        return "pong"
    }

    // GetHello serves
    // Method:   GET
    // Resource: http://localhost:8080/hello
    func (c *ExampleController) GetHello() interface{} {
        return map[string]string{"message": "Hello Iris!"}
    }


// GetUserBy serves
// Method:   GET
// Resource: http://localhost:8080/user/{username:string}
// By is a reserved "keyword" to tell the framework that you're going to
// bind path parameters in the function's input arguments, and it also
// helps to have "Get" and "GetBy" in the same controller.
//
// func (c *ExampleController) GetUserBy(username string) mvc.Result {
// 	return mvc.View{
// 		Name: "user/username.html",
// 		Data: username,
// 	}
// }

Can use more than one, the factory will make sure
that the correct http methods are being registered for each route
for this controller, uncomment these if you want:

    func (c *ExampleController) Post() {}
    func (c *ExampleController) Put() {}
    func (c *ExampleController) Delete() {}
    func (c *ExampleController) Connect() {}
    func (c *ExampleController) Head() {}
    func (c *ExampleController) Patch() {}
    func (c *ExampleController) Options() {}
    func (c *ExampleController) Trace() {}
*/
//
/*
    func (c *ExampleController) All() {}
    //        OR
    func (c *ExampleController) Any() {}

    func (c *ExampleController) BeforeActivation(b mvc.BeforeActivation) {
        // 1 -> the HTTP Method
        // 2 -> the route's path
        // 3 -> this controller's method name that should be handler for that route.
        b.Handle("GET", "/mypath/{param}", "DoIt", optionalMiddlewareHere...)
    }

    // After activation, all dependencies are set-ed - so read only access on them
    // but still possible to add custom controller or simple standard handlers.
    func (c *ExampleController) AfterActivation(a mvc.AfterActivation) {}


Iris web framework supports Request data, Models, Persistence Data and Binding
with the fastest possible execution.

Characteristics:

All HTTP Methods are supported, for example if want to serve `GET`
then the controller should have a function named `Get()`,
you can define more than one method function to serve in the same Controller.

Register custom controller's struct's methods as handlers with custom paths(even with regex parametermized path)
via the `BeforeActivation` custom event callback, per-controller. Example:

    package main

    import (
        "github.com/kataras/iris"
        "github.com/kataras/iris/mvc"
    )

    func main() {
        app := iris.New()
        mvc.Configure(app.Party("/root"), myMVC)
        app.Run(iris.Addr(":8080"))
    }

    func myMVC(app *mvc.Application) {
        // app.Register(...)
        // app.Router.Use/UseGlobal/Done(...)
        app.Handle(new(MyController))
    }

    type MyController struct {}

    func (m *MyController) BeforeActivation(b mvc.BeforeActivation) {
        // b.Dependencies().Add/Remove
        // b.Router().Use/UseGlobal/Done // and any standard API call you already know

        // 1-> Method
        // 2-> Path
        // 3-> The controller's function name to be parsed as handler
        // 4-> Any handlers that should run before the MyCustomHandler
        b.Handle("GET", "/something/{id:long}", "MyCustomHandler", anyMiddleware...)
    }

    // GET: http://localhost:8080/root
    func (m *MyController) Get() string { return "Hey" }

    // GET: http://localhost:8080/root/something/{id:long}
    func (m *MyController) MyCustomHandler(id int64) string { return "MyCustomHandler says Hey" }


Persistence data inside your Controller struct (share data between requests)
by defining services to the Dependencies or have a `Singleton` controller scope.

Share the dependencies between controllers or register them on a parent MVC Application, and ability
to modify dependencies per-controller on the `BeforeActivation` optional event callback inside a Controller,
i.e

    func(c *MyController) BeforeActivation(b mvc.BeforeActivation) { b.Dependencies().Add/Remove(...) }

Access to the `Context` as a controller's field(no manual binding is neede) i.e `Ctx iris.Context` or via a method's input argument,
i.e
    func(ctx iris.Context, otherArguments...)

Models inside your Controller struct (set-ed at the Method function and rendered by the View).
You can return models from a controller's method or set a field in the request lifecycle
and return that field to another method, in the same request lifecycle.

Flow as you used to, mvc application has its own `Router` which is a type of `iris/router.Party`, the standard iris api.
`Controllers` can be registered to any `Party`, including Subdomains, the Party's begin and done handlers work as expected.

Optional `BeginRequest(ctx)` function to perform any initialization before the method execution,
useful to call middlewares or when many methods use the same collection of data.

Optional `EndRequest(ctx)` function to perform any finalization after any method executed.

Session dynamic dependency via manager's `Start` to the MVC Application, i.e

    mvcApp.Register(sessions.New(sessions.Config{Cookie: "iris_session_id"}).Start)

Inheritance, recursively.

Access to the dynamic path parameters via the controller's methods' input arguments, no binding is needed.
When you use the Iris' default syntax to parse handlers from a controller, you need to suffix the methods
with the `By` word, uppercase is a new sub path. Example:

Register one or more relative paths and able to get path parameters, i.e

    If `mvc.New(app.Party("/user")).Handle(new(user.Controller))`

    - `func(*Controller) Get()` - `GET:/user` , as usual.
    - `func(*Controller) Post()` - `POST:/user`, as usual.
    - `func(*Controller) GetLogin()` - `GET:/user/login`
    - `func(*Controller) PostLogin()` - `POST:/user/login`
    - `func(*Controller) GetProfileFollowers()` - `GET:/user/profile/followers`
    - `func(*Controller) PostProfileFollowers()` - `POST:/user/profile/followers`
    - `func(*Controller) GetBy(id int64)` - `GET:/user/{param:long}`
    - `func(*Controller) PostBy(id int64)` - `POST:/user/{param:long}`

    If `mvc.New(app.Party("/profile")).Handle(new(profile.Controller))`

    - `func(*Controller) GetBy(username string)` - `GET:/profile/{param:string}`

    If `mvc.New(app.Party("/assets")).Handle(new(file.Controller))`

    - `func(*Controller) GetByWildard(path string)` - `GET:/assets/{param:path}`

    If `mvc.New(app.Party("/equality")).Handle(new(profile.Equality))`

    - `func(*Controller) GetBy(is bool)` - `GET:/equality/{param:boolean}`
    - `func(*Controller) GetByOtherBy(is bool, otherID int64)` - `GET:/equality/{paramfirst:boolean}/other/{paramsecond:long}`

    Supported types for method functions receivers: int, int64, bool and string.

Response via output arguments, optionally, i.e

    func(c *ExampleController) Get() string |
    (string, string) |
    (string, int) |
    (string, error) |
    int |
    (int, string) |
    (any, int) |
    error |
    (int, error) |
    (customStruct, error) |
    (any, error) |
    bool |
    (any, bool)
    customStruct |
    (customStruct, int) |
    (customStruct, string) |
    `Result` or (`Result`, error)

Where `any` means everything, from custom structs to standard language's types-.
`Result` is an interface which contains only that function: Dispatch(ctx iris.Context)
and Get where HTTP Method function(Post, Put, Delete...).


Iris MVC Method Result

Iris has a very powerful and blazing fast MVC support, you can return any value of any type from a method function
and it will be sent to the client as expected.

* if `string` then it's the body.
* if `string` is the second output argument then it's the content type.
* if `int` then it's the status code.
* if `bool` is false then it throws 404 not found http error by skipping everything else.
* if `error` and not nil then (any type) response will be omitted and error's text with a 400 bad request will be rendered instead.
* if `(int, error)` and error is not nil then the response result will be the error's text with the status code as `int`.
* if  `custom struct` or `interface{}` or `slice` or `map` then it will be rendered as json, unless a `string` content type is following.
* if `mvc.Result` then it executes its `Dispatch` function, so good design patters can be used to split the model's logic where needed.

Examples with good patterns to follow but not intend to be used in production of course can be found at:
https://github.com/kataras/iris/tree/master/_examples/#mvc.


Using Iris MVC for code reuse

By creating components that are independent of one another,
developers are able to reuse components quickly and easily in other applications.
The same (or similar) view for one application can be refactored for another application with
different data because the view is simply handling how the data is being displayed to the user.

If you're new to back-end web development read about the MVC architectural pattern first,
a good start is that wikipedia article: https://en.wikipedia.org/wiki/Model%E2%80%93view%E2%80%93controller.



That's the basics

But you should have a basic idea of the framework by now, we just scratched the surface.
If you enjoy what you just saw and want to learn more, please follow the below links:

Examples:

    https://github.com/kataras/iris/tree/master/_examples

Middleware:

    https://github.com/kataras/iris/tree/master/middleware
    https://github.com/iris-contrib/middleware

Home Page:

    https://iris-go.com

Book (in-progress):

    https://docs.iris-go.com

*/
package iris
