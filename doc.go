// Copyright (c) 2016-2017 Gerasimos Maropoulos
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

/*
Package iris is a web framework which provides efficient and well-designed
tools with robust set of features to create your awesome and
high-performance web application powered by unlimited potentials and portability.

Source code and other details for the project are available at GitHub:

   https://github.com/kataras/iris

Looking for further support?

   http://support.iris-go.com


Note: This package is under active development status.
Each month a new version is releasing
to adapt the latest web trends and technologies.

Basic HTTP API

Iris is a very pluggable ecosystem,
router can be customized by adapting a 'RouterBuilderPolicy && RouterReversionPolicy'.

With the power of Iris' router adaptors, developers are able to use any
third-party router's path features without any implications to the rest
of their API.

A Developer is able to select between two out-of-the-box powerful routers:

Httprouter, it's a custom version of https://github.comjulienschmidt/httprouter,
which is edited to support iris' subdomains, reverse routing, custom http errors and a lot features,
it should be a bit faster than the original too because of iris' Context.
It uses `/mypath/:firstParameter/path/:secondParameter` and `/mypath/*wildcardParamName` .

Gorilla Mux, it's the https://github.com/gorilla/mux which supports subdomains,
custom http errors, reverse routing, pattern matching via regex and the rest of the iris' features.
It uses `/mypath/{firstParameter:any-regex-valid-here}/path/{secondParameter}` and `/mypath/{wildcardParamName:.*}`


Example code:


      package main

      import (
      	"gopkg.in/kataras/iris.v6"
      	"gopkg.in/kataras/iris.v6/adaptors/httprouter" // <--- or adaptors/gorillamux
      )

      func main() {
      	app := iris.New()
      	app.Adapt(httprouter.New()) // <--- or gorillamux.New()

      	// HTTP Method: GET
      	// PATH: http://127.0.0.1/
      	// Handler(s): index
      	app.Get("/", index)

      	app.Listen(":80")
      }

      func index(ctx *iris.Context) {
      	ctx.HTML(iris.StatusOK, "<h1> Welcome to my page!</h1>")
      }


Run

  $ go run main.go

  $ iris run main.go ## enables reload on source code changes.


All HTTP methods are supported, users can register handlers for same paths on different methods.
The first parameter is the HTTP Method,
second parameter is the request path of the route,
third variadic parameter should contains one or more iris.Handler/HandlerFunc executed
by the registered order when a user requests for that specific resouce path from the server.

Example code:


        app := iris.New()

        app.Handle("GET", "/about", aboutHandler)

        type aboutHandler struct {}
        func (a aboutHandler) Serve(ctx *iris.Context){
          ctx.HTML("Hello from /about, executed from an iris.Handler")
        }

        app.HandleFunc("GET", "/contact", func(ctx *iris.Context){
          ctx.HTML(iris.StatusOK, "Hello from /contact, executed from an iris.HandlerFunc")
        })


In order to make things easier for the user, Iris provides functions for all HTTP Methods.
The first parameter is the request path of the route,
second variadic parameter should contains one or more iris.HandlerFunc executed
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

      func handler(ctx *iris.Context){
        ctx.Writef("Hello from method: %s and path: %s", ctx.Method(), ctx.Path())
      }


Parameterized Path


Path Parameters' syntax depends on the selected router.
This is the only difference between the routers, the registered path form, the API remains the same for both.

Example `gorillamux` code:


      package main

      import (
      	"gopkg.in/kataras/iris.v6"
      	"gopkg.in/kataras/iris.v6/adaptors/gorillamux"
      )

      func main() {
      	app := iris.New()
      	app.Adapt(iris.DevLogger())
      	app.Adapt(gorillamux.New())

      	app.OnError(iris.StatusNotFound, func(ctx *iris.Context) {
      		ctx.HTML(iris.StatusNotFound, "<h1> custom http error page </h1>")
      	})

      	app.Get("/healthcheck", h)

      	gamesMiddleware := func(ctx *iris.Context) {
      		println(ctx.Method() + ": " + ctx.Path())
      		ctx.Next()
      	}

      	games := app.Party("/games", gamesMiddleware)
      	{ // braces are optional of course, it's just a style of code
      		games.Get("/{gameID:[0-9]+}/clans", h)
      		games.Get("/{gameID:[0-9]+}/clans/clan/{publicID:[0-9]+}", h)
      		games.Get("/{gameID:[0-9]+}/clans/search", h)

      		games.Put("/{gameID:[0-9]+}/players/{publicID:[0-9]+}", h)
      		games.Put("/{gameID:[0-9]+}/clans/clan/{publicID:[0-9]+}", h)

      		games.Post("/{gameID:[0-9]+}/clans", h)
      		games.Post("/{gameID:[0-9]+}/players", h)
      		games.Post("/{gameID:[0-9]+}/clans/{publicID:[0-9]+}/leave", h)
      		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/application", h)
      		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/application/:action", h)
      		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/invitation", h)
      		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/invitation/:action", h)
      		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/delete", h)
      		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/promote", h)
      		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/demote", h)
      	}

      	app.Get("/anything/{anythingparameter:.*}", func(ctx *iris.Context) {
      		s := ctx.Param("anythingparameter")
      		ctx.Writef("The path after /anything is: %s", s)
      	})

      	p := app.Party("mysubdomain.")
      	// http://mysubdomain.myhost.com/
      	p.Get("/", h)

      	app.Listen("myhost.com:80")
      }

      func h(ctx *iris.Context) {
      	ctx.HTML(iris.StatusOK, "<h1>Path<h1/>"+ctx.Path())
      }


Example `httprouter` code:


      package main

      import (
        "gopkg.in/kataras/iris.v6"
        "gopkg.in/kataras/iris.v6/adaptors/httprouter" // <---- NEW
      )

      func main() {
        app := iris.New()
        app.Adapt(iris.DevLogger())
        app.Adapt(httprouter.New()) // <---- NEW


        app.OnError(iris.StatusNotFound, func(ctx *iris.Context){
          ctx.HTML(iris.StatusNotFound, "<h1> custom http error page </h1>")
        })


        app.Get("/healthcheck", h)

        gamesMiddleware := func(ctx *iris.Context) {
          println(ctx.Method() + ": " + ctx.Path())
          ctx.Next()
        }

        games:= app.Party("/games", gamesMiddleware)
        { // braces are optional of course, it's just a style of code
        	 games.Get("/:gameID/clans", h)
        	 games.Get("/:gameID/clans/clan/:publicID", h)
        	 games.Get("/:gameID/clans/search", h)

        	 games.Put("/:gameID/players/:publicID", h)
        	 games.Put("/:gameID/clans/clan/:publicID", h)

        	 games.Post("/:gameID/clans", h)
        	 games.Post("/:gameID/players", h)
        	 games.Post("/:gameID/clans/:publicID/leave", h)
        	 games.Post("/:gameID/clans/:clanPublicID/memberships/application", h)
        	 games.Post("/:gameID/clans/:clanPublicID/memberships/application/:action", h)
        	 games.Post("/:gameID/clans/:clanPublicID/memberships/invitation", h)
        	 games.Post("/:gameID/clans/:clanPublicID/memberships/invitation/:action", h)
        	 games.Post("/:gameID/clans/:clanPublicID/memberships/delete", h)
        	 games.Post("/:gameID/clans/:clanPublicID/memberships/promote", h)
        	 games.Post("/:gameID/clans/:clanPublicID/memberships/demote", h)
        }

        app.Get("/anything/*anythingparameter", func(ctx *iris.Context){
          s := ctx.Param("anythingparameter")
          ctx.Writef("The path after /anything is: %s",s)
        })

        mysubdomain:= app.Party("mysubdomain.")
        // http://mysubdomain.myhost.com/
        mysudomain.Get("/", h)

        app.Listen("myhost.com:80")
      }

      func h(ctx *iris.Context) {
      	ctx.HTML(iris.StatusOK, "<h1>Path<h1/>"+ctx.Path())
      }


Grouping Routes


A set of routes that are being groupped by path prefix can (optionally) share the same middleware handlers and template layout.
A group can have a nested group too.

`.Party` is being used to group routes, developers can declare an unlimited number of (nested) groups.


Example code:


      users:= app.Party("/users", myAuthHandler)

      // http://myhost.com/users/42/profile
      users.Get("/:userid/profile", userProfileHandler) // httprouter path parameters
      // http://myhost.com/users/messages/1
      users.Get("/inbox/:messageid", userMessageHandler)

      app.Listen("myhost.com:80")



Custom HTTP Errors


With iris users are able to register their own handlers for http statuses like 404 not found, 500 internal server error and so on.

Example code:

      // when 404 then render the template $templatedir/errors/404.html
      // *read below for information about the view engine.*
      app.OnError(iris.StatusNotFound, func(ctx *iris.Context){
        ctx.RenderWithstatus(iris.StatusNotFound, "errors/404.html", nil)
      })

      app.OnError(500, func(ctx *iris.Context){
        // ...
      })


Custom http errors can be also be registered to a specific group of routes.

Example code:


      games:= app.Party("/games", gamesMiddleware)
      {
          games.Get("/{gameID:[0-9]+}/clans", h) // gorillamux path parameters
          games.Get("/{gameID:[0-9]+}/clans/clan/{publicID:[0-9]+}", h)
          games.Get("/{gameID:[0-9]+}/clans/search", h)
      }

      games.OnError(iris.StatusNotFound, gamesNotFoundHandler)



Static Files

      // Favicon serves static favicon
      // accepts 2 parameters, second is optional
      // favPath (string), declare the system directory path of the __.ico
      // requestPath (string), it's the route's path, by default this is the "/favicon.ico" because some browsers tries to get this by default first,
      // you can declare your own path if you have more than one favicon (desktop, mobile and so on)
      //
      // this func will add a route for you which will static serve the /yuorpath/yourfile.ico to the /yourfile.ico (nothing special that you can't handle by yourself)
      // Note that you have to call it on every favicon you have to serve automatically (desktop, mobile and so on)
      //
      // panics on error
      Favicon(favPath string, requestPath ...string) RouteInfo

      // StaticContent serves bytes, memory cached, on the reqPath
      // a good example of this is how the websocket server uses that to auto-register the /iris-ws.js
      StaticContent(reqPath string, cType string, content []byte) RouteInfo

      // StaticHandler returns a new Handler which serves static files
      StaticHandler(reqPath string, systemPath string, showList bool, enableGzip bool, exceptRoutes ...RouteInfo) HandlerFunc

      // StaticWeb returns a handler that serves HTTP requests
      // with the contents of the file system rooted at directory.
      //
      // first parameter: the route path
      // second parameter: the system directory
      // third OPTIONAL parameter: the exception routes
      //      (= give priority to these routes instead of the static handler)
      // for more options look iris.StaticHandler.
      //
      //     iris.StaticWeb("/static", "./static")
      //
      // As a special case, the returned file server redirects any request
      // ending in "/index.html" to the same path, without the final
      // "index.html".
      //
      // StaticWeb calls the StaticHandler(reqPath, systemPath, listingDirectories: false, gzip: false ).
      StaticWeb(reqPath string, systemPath string, exceptRoutes ...RouteInfo) RouteInfo

      // StaticEmbedded  used when files are distributed inside the app executable, using go-bindata mostly
      // First parameter is the request path, the path which the files in the vdir will be served to, for example "/static"
      // Second parameter is the (virtual) directory path, for example "./assets"
      // Third parameter is the Asset function
      // Forth parameter is the AssetNames function
      StaticEmbedded(reqPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string) RouteInfo

Example code:


      package main

      import (
      	"gopkg.in/kataras/iris.v6"
      	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
      )

      func main() {

      	app := iris.New()
      	app.Adapt(iris.DevLogger())
      	app.Adapt(httprouter.New())

      	app.Favicon("./static/favicons/iris_favicon_32_32.ico")
      	// This will serve the ./static/favicons/iris_favicon_32_32.ico to: 127.0.0.1:8080/favicon.ico

      	// app.Favicon("./static/favicons/iris_favicon_32_32.ico", "/favicon_32_32.ico")
      	// This will serve the ./static/favicons/iris_favicon_32_32.ico to: 127.0.0.1:8080/favicon_32_32.ico

      	app.Get("/", func(ctx *iris.Context) {
      		ctx.HTML(iris.StatusOK, "You should see the favicon now at the side of your browser.")
      	})

      	app.Listen(":8080")
      }



Middleware Ecosystem

Middleware is just a concept of ordered chain of handlers.
Middleware can be registered globally, per-party, per-subdomain and per-route.


Example code:

      // globally
      // before any routes, appends the middleware to all routes
      app.UseFunc(func(ctx *iris.Context){
         // ... any code here

         ctx.Next() // in order to continue to the next handler,
         // if that is missing then the next in chain handlers will be not executed,
         // useful for authentication middleware
      })

      // globally
      // after or before any routes, prepends the middleware to all routes
      app.UseGlobalFunc(handlerFunc1, handlerFunc2, handlerFunc3)

      // per-route
      app.Post("/login", authenticationHandler, loginPageHandler)

      // per-party(group of routes)
      users := app.Party("/users", usersMiddleware)
      users.Get("/", usersIndex)

      // per-subdomain
      mysubdomain := app.Party("mysubdomain.", firstMiddleware)
      mysubdomain.UseFunc(secondMiddleware)
      mysubdomain.Get("/", mysubdomainIndex)

      // per wildcard, dynamic subdomain
      dynamicSub := app.Party(".*", firstMiddleware, secondMiddleware)
      dynamicSub.Get("/", func(ctx *iris.Context){
        ctx.Writef("Hello from subdomain: "+ ctx.Subdomain())
      })


    `iris.ToHandler` converts(by wrapping) any `http.Handler/HandlerFunc` or
    `func(w http.ResponseWriter,r *http.Request, next http.HandlerFunc)` to an `iris.HandlerFunc`.

    iris.ToHandler(nativeNethttpHandler)

Let's convert the https://github.com/rs/cors net/http external middleware which returns a `next form` handler.


Example code:

      package main

      import (
      	"github.com/rs/cors"
      	"gopkg.in/kataras/iris.v6"
      	"gopkg.in/kataras/iris.v6/adaptors/gorillamux"
      )

      // newCorsMiddleware returns a new cors middleware
      // with the provided options.
      func newCorsMiddleware() iris.HandlerFunc {
      	options := cors.Options{
      		AllowedOrigins: []string{"*"},
      	}
      	handlerWithNext := cors.New(options).ServeHTTP

      	// this is the only func you will have to use if you're going
      	// to make use of any external net/http middleware.
      	// iris.ToHandler converts the net/http middleware to an iris-compatible.
      	return iris.ToHandler(handlerWithNext)
      }

      func main() {
      	app := iris.New()
      	app.Adapt(gorillamux.New())

      	// Any registers a route to all http methods.
      	app.Any("/user", newCorsMiddleware(), func(ctx *iris.Context) {
      		// ....
      	})

      	app.Listen(":8080")
      }



View Engine


Iris supports 5 template engines out-of-the-box, developers can still use any external golang template engine,
as `context.ResponseWriter` is an `io.Writer`.

All of these five template engines have common features with common API,
like Layout, Template Funcs, Party-specific layout, partial rendering and more.

      The standard html, based on github.com/kataras/go-template/tree/master/html
      its template parser is the golang.org/pkg/html/template/.

      Django, based ongithub.com/kataras/go-template/tree/master/django
      its template parser is the github.com/flosch/pongo2

      Pug(Jade), based on github.com/kataras/go-template/tree/master/pug
      its template parser is the github.com/Joker/jade

      Handlebars, based on github.com/kataras/go-template/tree/master/handlebars
      its template parser is the github.com/aymerick/raymond

      Amber, based on github.com/kataras/go-template/tree/master/amber
      its template parser is the github.com/eknkc/amber


Example code:

      package main

      import (
      	"gopkg.in/kataras/iris.v6"
      	"gopkg.in/kataras/iris.v6/adaptors/gorillamux"
      	"gopkg.in/kataras/iris.v6/adaptors/view" // <--- it contains all the template engines
      )

      func main() {
      	app := iris.New(iris.Configuration{Gzip: false, Charset: "UTF-8"}) // defaults to these

      	app.Adapt(iris.DevLogger())
      	app.Adapt(gorillamux.New())

      	// - standard html  | view.HTML(...)
      	// - django         | view.Django(...)
      	// - pug(jade)      | view.Pug(...)
      	// - handlebars     | view.Handlebars(...)
      	// - amber          | view.Amber(...)
      	app.Adapt(view.HTML("./templates", ".html")) // <---- use the standard html

      	// default template funcs:
      	//
      	// - {{ url "mynamedroute" "pathParameter_ifneeded"} }
      	// - {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
      	// - {{ render "header.html" }}
      	// - {{ render_r "header.html" }} // partial relative path to current page
      	// - {{ yield }}
      	// - {{ current }}
      	//
      	// to adapt custom funcs, use:
      	app.Adapt(iris.TemplateFuncsPolicy{"myfunc": func(s string) string {
      		return "hi " + s
      	}}) // usage inside template: {{ hi "kataras"}}

      	app.Get("/hi", func(ctx *iris.Context) {
      		ctx.Render(
      			// the file name of the template relative to the './templates'.
      			"hi.html",
      			iris.Map{"Name": "Iris"},
      			// the .Name inside the ./templates/hi.html,
      			// you can use any custom struct that you want to bind to the requested template.
      			iris.Map{"gzip": false}, // set to true to enable gzip compression.
      		)

      	})

      	// http://127.0.0.1:8080/hi
      	app.Listen(":8080")
      }


View engine supports bundled(https://github.com/jteeuwen/go-bindata) template files too.
go-bindata gives you two functions, asset and assetNames,
these can be setted to each of the template engines using the `.Binary` func.

Example code:

      djangoEngine := view.Django("./templates", ".html")
      djangoEngine.Binary(asset, assetNames)
      app.Adapt(djangoEngine)

A real example can be found here: https://github.com/kataras/iris/tree/v6/_examples/intermediate/view/embedding-templates-into-app.

Enable auto-reloading of templates on each request. Useful while developers are in dev mode
as they no neeed to restart their app on every template edit.

Example code:


    pugEngine := view.Pug("./templates", ".jade")
    pugEngine.Reload(true) // <--- set to true to re-build the templates on each request.
    app.Adapt(pugEngine)


Each one of these template engines has different options located here: https://github.com/kataras/iris/tree/v6/adaptors/view .

That's the basics

But you should have a basic idea of the framework by now, we just scratched the surface.
If you enjoy what you just saw and want to learn more, please follow the below links:

* Examples: https://github.com/iris-contrib/examples

* Adaptors: https://github.com/kataras/iris/tree/v6/adaptors

* Middleware: https://github.com/kataras/iris/tree/v6/middleware and
* https://github.com/iris-contrib/middleware


*/
package iris
