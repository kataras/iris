# Iris Web Framework
<img align="right" width="132" src="http://kataras.github.io/iris/assets/56e4b048f1ee49764ddd78fe_iris_favicon.ico">
[![Build Status](https://travis-ci.org/kataras/iris.svg?branch=development&style=flat-square)](https://travis-ci.org/kataras/iris)
[![Go Report Card](https://goreportcard.com/badge/github.com/kataras/iris?style=flat-square)](https://goreportcard.com/report/github.com/kataras/iris)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/kataras/iris?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![GoDoc](https://godoc.org/github.com/kataras/iris?status.svg)](https://godoc.org/github.com/kataras/iris)
[![License](https://img.shields.io/badge/license-BSD3-blue.svg?style=flat-square)](LICENSE)

The [fastest](#benchmarks)  go web framework which provides robust set of features for building modern & shiny web applications.

![Hi Iris GIF](http://kataras.github.io/iris/assets/hi_iris_march.gif)



## News: 12 April 2016 
V1.1


Iris now has [fasthttp](https://github.com/valyala/fasthttp) as it's base, standar API didn't changed but Context has some additional things and other things works different than before.[*](#context)


[Examples](https://github.com/kataras/iris/tree/examples) are not yet updated with the new version, stay tunned!

----			
				

## Features

* **FastHTTP**: Iris uses [fasthttp](https://github.com/valyala/fasthttp) package as it's default base [*](#declaration)

* **Context**: Iris uses [Context](#context) for storing route params, sharing variables between middlewares and render rich content to the client.

* **Plugins**: You can build your own plugins to  inject the Iris framework[*](#plugins).

* **Full API**: All http methods are supported, you can group routes and sharing resources together[*](#api).

* **Zero allocations**: Iris generates zero garbage.

* **Multi server instances**: Besides the fact that Iris has a default main server. You can declare as many as you need[*](#declaration).

* **Middlewares**: Create and use global or per route middlewares with the Iris' simplicity[*](#middlewares). 

### Q: What makes iris significantly [faster](#benchmarks)?
##### A: These are the QNIQUE features that Iris brings

*    First of all Iris uses [fasthttp](https://github.com/valyala/fasthttp) as it's  base 

*    Iris uses the same algorithm as the BSD's kernel does for routing (call it Trie)

*    Iris can detect what features are used and what don't and optimized itself before server run.

*    Middlewares and Plugins are 'light' and that is, a principle.


## Table of Contents

- [Install](#install)
- [Introduction](#introduction)
- [TLS](#tls)
- [Handlers](#handlers)
 - [Using Handlers](#using-handlers)
 - [Using HandlerFuncs](#using-handlerfuncs)
 - [Using Annotated](#using-annotated)
 - [Using native http.Handler](#using-native-httphandler)
	    - [Using native http.Handler via iris.ToHandlerFunc()](#Using-native-http.Handler-via-ToHandlerFunc())
- [Middlewares](#middlewares)
- [API](#api)
- [Declaration & Options](#declaration)
- [Party](#party)
- [Named Parameters](#named-parameters)
- [Catch all and Static serving](#match-anything-and-the-static-serve-handler)
- [Custom HTTP Errors](#custom-http-errors)
- [Context](#context)
- [Plugins](#plugins)
- [Internationalization and Localization](https://github.com/kataras/iris/tree/examples/middleware_internationalization_i18n)
- [Examples](https://github.com/kataras/iris/tree/examples)
- [Benchmarks](#benchmarks)
- [Third Party Middlewares](#third-party-middlewares)
- [Contributors](#contributors)
- [Community](#community)
- [Todo](#todo)
- [External source articles](#articles)
- [License](#license)


### Install
In order to have the latest version update the package one per week
```sh
$ go get -u github.com/kataras/iris
```


## Introduction
The name of this framework came from **Greek mythology**, **Iris** was the name of the Greek goddess of the **rainbow**.
Iris is a very minimal but flexible golang http middleware & standalone web application framework, providing a robust set of features for building single & multi-page, web applications.

```go
package main

import "github.com/kataras/iris"

func main() {
	iris.Get("/hello", func(c *iris.Context) {
		c.HTML("<b> Hello </b>")
	})
	iris.Listen(":8080")
}

```
>Note: for macOS, If you are having problems on .Listen then pass only the port "8080" without ':'

#### Listening

```go
//1  defaults to tcp 8080
iris.Listen()
//2
log.Fatal(iris.Listen(":8080"))

```

## TLS
TLS for https:// and http2:

```go
ListenTLS(fulladdr string, certFile, keyFile string) error
```
```go
//1
iris.ListenTLS(":8080", "myCERTfile.cert", "myKEYfile.key")
//2
log.Fatal(iris.ListenTLS(":8080", "myCERTfile.cert", "myKEYfile.key"))

```


## Handlers

Handlers should implement the Handler interface:

```go
type Handler interface {
	Serve(*Context)
}
```

### Using Handlers

```go

type myHandlerGet struct {
}

func (m myHandlerGet) Serve(c *iris.Context) {
    c.Write("From %s", c.PathString())
}

//and so on


iris.Handle("GET", "/get", myHandlerGet)
iris.Handle("POST", "/post", post)
iris.Handle("PUT", "/put", put)
iris.Handle("DELETE", "/delete", del)
```

### Using HandlerFuncs
HandlerFuncs should implement the Serve(*Context) func.
HandlerFunc is most simple method to register a route or a middleware, but under the hoods it's acts like a Handler. It's implements the Handler interface as well:

```go
type HandlerFunc func(*Context)

func (h HandlerFunc) Serve(c *Context) {
	h(c)
}

```
HandlerFuncs shoud have this function signature:
```go
func handlerFunc(c *iris.Context)  {
	c.Write("Hello")
}


iris.HandleFunc("GET","/letsgetit",handlerFunc)
//OR
iris.Get("/get", handlerFunc)
iris.Post("/post", handlerFunc)
iris.Put("/put", handlerFunc)
iris.Delete("/delete", handlerFunc)
```


### Using Annotated
Implements the Handler interface

```go
///file: userhandler.go
import "github.com/kataras/iris"

type UserHandler struct {
	iris.Handler `get:"/profile/user/:userId"`
}

func (u *UserHandler) Serve(c *iris.Context) {
	userId := c.Param("userId")
	c.RenderFile("user.html", struct{ Message string }{Message: "Hello User with ID: " + userId})
}

```
```go
///file: main.go
//...cache the html files
iris.Templates("src/iristests/templates/**/*.html")
//...register the handler
iris.HandleAnnotated(&UserHandler{})
//...continue writing your wonderful API

```



### Using native http.Handler
> Note that using native http handler you cannot access url params.



```go

type nativehandler struct {}

func (_ nativehandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {

}

func main() {
	iris.Handle("", "/path", iris.ToHandler(nativehandler{}))
	//"" means ANY(GET,POST,PUT,DELETE and so on)
}


```

#### Using native http.Handler via *iris.ToHandlerFunc()*

```go
iris.Get("/letsget", iris.ToHandlerFunc(nativehandler{}))
iris.Post("/letspost", iris.ToHandlerFunc(nativehandler{}))
iris.Put("/letsput", iris.ToHandlerFunc(nativehandler{}))
iris.Delete("/letsdelete", iris.ToHandlerFunc(nativehandler{}))

```

## Middlewares

Middlewares in Iris are not complicated, imagine them as simple Handlers.
They should implement the Handler interface as well:

```go
type Handler interface {
	Serve(*Context)
}
type Middleware []Handler
```

Handler middleware example:

```go

type myMiddleware struct {}

func (m *myMiddleware) Serve(c *iris.Context){
	shouldContinueToTheNextHandler := true

	if shouldContinueToTheNextHandler {
		c.Next()
	}else{
	    c.WriteText(403,"Forbidden !!")
	}

}

iris.Use(&myMiddleware{})

iris.Get("/home", func (c *iris.Context){
	c.HTML("<h1>Hello from /home </h1>")
})

iris.Listen()
```

HandlerFunc middleware example:

```go

func myMiddleware(c *iris.Context){
	c.Next()
}

iris.UseFunc(myMiddleware)

```

HandlerFunc middleware for a specific route:

```go

func mySecondMiddleware(c *iris.Context){
	c.Next()
}

iris.Get("/dashboard", func(c *iris.Context) {
    loggedIn := true
    if loggedIn {
        c.Next()
    }
}, mySecondMiddleware, func (c *iris.Context){
    c.Write("The last HandlerFunc is the main handler, all before that are the middlewares for this route /dashboard")
})

iris.Listen(":8080")

```

Uses one of build'n Iris [middlewares](https://github.com/kataras/iris/tree/development/middleware), view practical [examples here](https://github.com/kataras/iris/tree/examples)

```go
package main

import (
 "github.com/kataras/iris"
 "github.com/kataras/iris/middleware/logger"
)

type Page struct {
	Title string
}

iris.Templates("./yourpath/templates/*")

iris.Use(logger.Logger())

iris.Get("/", func(c *iris.Context) {
		c.RenderFile("index.html", Page{"My Index Title"})
})

iris.Listen(":8080") // .Listen() listens to TCP port 8080 by default
```

## API
**Use of GET,  POST,  PUT,  DELETE, HEAD, PATCH & OPTIONS**

```go
package main

import "github.com/kataras/iris"

func main() {
	iris.Get("/home", testGet)
	iris.Post("/login",testPost)
	iris.Put("/add",testPut)
	iris.Delete("/remove",testDelete)
	iris.Head("/testHead",testHead)
	iris.Patch("/testPatch",testPatch)
	iris.Options("/testOptions",testOptions)

	iris.Listen(":8080")
}

func testGet(c *iris.Context) {
	//...
}
func testPost(c *iris.Context) {
	//...
}

//and so on....
```

## Declaration

Let's make a pause,

- Q: Other frameworks needs more lines to start a server, why Iris is different?
- A: Iris gives you the freedom to choose between three methods/ways to use Iris

 1. global **iris.**
 2. set a new iris with variable  = iris**.New()**
 3. set a new iris with custom options with variable = iris**.Custom(options)**


```go
import "github.com/kataras/iris"

// 1.
func methodFirst() {

	iris.Get("/home",func(c *iris.Context){})
	iris.Listen(":8080")
	//iris.ListenTLS(":8080","yourcertfile.cert","yourkeyfile.key"
}
// 2.
func methodSecond() {

	api := iris.New()
	api.Get("/home",func(c *iris.Context){})
	api.Listen(":8080")
}
// 3.
func methodThree() {
	//these are the default options' values
	options := iris.StationOptions{
		Profile:            false,
		ProfilePath:        iris.DefaultProfilePath,
		PathCorrection: 	true, //explanation at the end of this chapter
	}//these are the default values that you can change
	//DefaultProfilePath = "/debug/pprof"

	api := iris.Custom(options)
	api.Get("/home",func(c *iris.Context){})
	api.Listen(":8080")
}

```

> Note that with 2. & 3. you **can define and use more than one Iris container** in the
> same app, when it's necessary.

As you can see there are some options that you can chage at your iris declaration, you cannot change them after.
**If an option value not passed then it considers to be false if bool or the default if string.**

For example if we do that...
```go
import "github.com/kataras/iris"
func main() {
	options := iris.StationOptions{
		Profile:            true,
		ProfilePath:        "/mypath/debug",
	}

	api := iris.Custom(options)
	api.Listen(":8080")
}
```
run it, then you can open your browser, type '**localhost:8080/mypath/debug/profile**' at the location input field and you should see a webpage  shows you informations about CPU.

For profiling & debug there are seven (7) generated pages ('/debug/pprof/' is the default profile path, which on previous example we changed it to '/mypath/debug'):

 1. /debug/pprof/cmdline
 2. /debug/pprof/profile
 3. /debug/pprof/symbol
 4. /debug/pprof/goroutine
 5. /debug/pprof/heap
 6. /debug/pprof/threadcreate
 7. /debug/pprof/pprof/block


**PathCorrection**
corrects and redirects the requested path to the registed path
for example, if /home/ path is requested but no handler for this Route found,
then the Router checks if /home handler exists, if yes, redirects the client to the correct path /home
and VICE - VERSA if /home/ is registed but /home is requested then it redirects to /home/ (Default is true)

## Party

Let's party with Iris web framework!

```go
func main() {

    //log everything middleware

    iris.UseFunc(func(c *iris.Context) {
		println("[Global log] the requested url path is: ", c.PathString())
		c.Next()
	})

    // manage all /users
    users := iris.Party("/users")
    {
  	    // provide a  middleware
		users.UseFunc(func(c *iris.Context) {
			println("LOG [/users...] This is the middleware for: ", c.PathString())
			c.Next()
		})
		users.Post("/login", loginHandler)
        users.Get("/:userId", singleUserHandler)
        users.Delete("/:userId", userAccountRemoveUserHandler)
    }



    // Party inside an existing Party example:

    beta:= iris.Party("/beta")

    admin := beta.Party("/admin")
    {
		/// GET: /beta/admin/
		admin.Get("/", func(c *iris.Context){})
		/// POST: /beta/admin/signin
        admin.Post("/signin", func(c *iris.Context){})
		/// GET: /beta/admin/dashboard
        admin.Get("/dashboard", func(c *iris.Context){})
		/// PUT: /beta/admin/users/add
        admin.Put("/users/add", func(c *iris.Context){})
    }



    iris.Listen(":8080")
}
```


## Named Parameters

Named parameters are just custom paths to your routes, you can access them for each request using context's **c.Param("nameoftheparameter")**. Get all, as array (**{Key,Value}**) using **c.Params** property.

No limit on how long a path can be.

Usage:


```go
package main

import "github.com/kataras/iris"

func main() {
	// MATCH to /hello/anywordhere  (if PathCorrection:true match also /hello/anywordhere/)
	// NOT match to /hello or /hello/ or /hello/anywordhere/something
	iris.Get("/hello/:name", func(c *iris.Context) {
		name := c.Param("name")
		c.Write("Hello %s", name)
	})

	// MATCH to /profile/iris/friends/42  (if PathCorrection:true matches also /profile/iris/friends/42/ ,otherwise not match)
	// NOT match to /profile/ , /profile/something ,
	// NOT match to /profile/something/friends,  /profile/something/friends ,
	// NOT match to /profile/anything/friends/42/something
	iris.Get("/profile/:fullname/friends/:friendId",
		func(c *iris.Context){
			name:= c.Param("fullname")
			//friendId := c.ParamInt("friendId")
			c.HTML("<b> Hello </b>"+name)
		})

	iris.Listen(":8080")
}

```

## Match anything and the Static serve handler

####Catch all
```go
// Will match any request which url's preffix is "/anything/" and has content after that
iris.Get("/anything/*randomName", func(c *iris.Context) { } )
// Match: /anything/whateverhere/whateveragain , /anything/blablabla
// c.Params("randomName") will be /whateverhere/whateveragain, blablabla
// Not Match: /anything , /anything/ , /something
```
#### Static handler using *iris.Static(""/public",./path/to/the/resources/directory/", 1)*
```go
iris.Static("/public", "./static/resources/",1))
//-> /public/assets/favicon.ico
```

## Custom HTTP Errors

You can define your own handlers for http errors, which can render an html file for example. e.g for for 404 not found:

```go
package main

import "github.com/kataras/iris"
func main() {
	iris.OnError(404,func (c *iris.Context){
		c.HTML("<h1> The page you looking doesn't exists </h1>")
		c.SetStatusCode(404)
	})
	//or OnNotFound(func (c *iris.Context){})... for 404 only.
	//or OnPanic(func (c *iris.Context){})... for 500 only.

	//...
}

```

We saw how to declare a custom error for a http status code, now let's look for how to send/emit an error to the client manually  , for example let's emit the 404 we defined before, simple:

```go

iris.Get("/thenotfound",func (c *iris.Context) {
	c.EmitError(404)
	//or c.NotFound() for 404 only.
	//and c.Panic() for 500 only.
})

```

## Context
![Iris Context Outline view](http://kataras.github.io/iris/assets/context_view.png)



Inside the [examples](https://github.com/kataras/iris/tree/examples) branch you will find practical examples



## Plugins
Plugins are modules that you can build to inject the Iris' flow. Think it like a middleware for the Iris framework itself, not only the requests. Middleware starts it's actions after the server listen, Plugin on the other hand starts working when you registed them, from the begin, to the end. Look how it's interface looks:

```go
type (
	// IPlugin is the interface which all Plugins must implement.
	//
	// A Plugin can register other plugins also from it's Activate state
	IPlugin interface {
		// GetName has to returns the name of the plugin, a name is unique
		// name has to be not dependent from other methods of the plugin,
		// because it is being called even before the Activate
		GetName() string
		// GetDescription has to returns the description of what the plugins is used for
		GetDescription() string

		// Activate called BEFORE the plugin being added to the plugins list,
		// if Activate returns none nil error then the plugin is not being added to the list
		// it is being called only one time
		//
		// PluginContainer parameter used to add other plugins if that's necessary by the plugin
		Activate(IPluginContainer) error
	}

	IPluginPreHandle interface {
		// PreHandle it's being called every time BEFORE a Route is registed to the Router
		//
		//  parameter is the Route
		PreHandle(IRoute)
	}

	IPluginPostHandle interface {
		// PostHandle it's being called every time AFTER a Route successfully registed to the Router
		//
		// parameter is the Route
		PostHandle(IRoute)
	}

	IPluginPreListen interface {
		// PreListen it's being called only one time, BEFORE the Server is started (if .Listen called)
		// is used to do work at the time all other things are ready to go
		//  parameter is the station
		PreListen(*Station)
	}

	IPluginPostListen interface {
		// PostListen it's being called only one time, AFTER the Server is started (if .Listen called)
		// parameter is the station
		PostListen(*Station)
	}

	IPluginPreClose interface {
		// PreClose it's being called only one time, BEFORE the Iris .Close method
		// any plugin cleanup/clear memory happens here
		//
		// The plugin is deactivated after this state
		PreClose(*Station)
	}
)

```

A small example, imagine that you want to get all routes registered to your server (OR modify them at runtime),  with their time registed, methods, (sub)domain  and the path, what whould you do on other frameworks when you want something from the framework which it doesn't supports out of the box? and what you can do with Iris:

```go
//file myplugin.go
package main

import (
	"time"

	"github.com/kataras/iris"
)

type RouteInfo struct {
	Method       string
	Domain       string
	Path         string
	TimeRegisted time.Time
}

type myPlugin struct {
	container iris.IPluginContainer
	routes    []RouteInfo
}

func NewMyPlugin() *myPlugin {
	return &myPlugin{routes: make([]RouteInfo, 0)}
}

// All plugins must at least implements these 3 functions

func (i *myPlugin) Activate(container iris.IPluginContainer) error {
	// use the container if you want to register other plugins to the server, yes it's possible a plugin can registers other plugins too.
	// here we set the container in order to use it's printf later at the PostListen.
	i.container = container
	return nil
}

func (i myPlugin) GetName() string {
	return "MyPlugin"
}

func (i myPlugin) GetDescription() string {
	return "My Plugin is just a simple Iris plugin.\n"
}

//
// Implement our plugin, you can view your inject points - listeners on the /kataras/iris/plugin.go too.
//
// Implement the PostHandle, because this is what we need now, we need to add a listener after a route is registed to our server so we do:
func (i *myPlugin) PostHandle(route iris.IRoute) {
	myRouteInfo := &RouteInfo{}
	myRouteInfo.Method = route.GetMethod()
	myRouteInfo.Domain = route.GetDomain()
	myRouteInfo.Path = route.GetPath()

	myRouteInfo.TimeRegisted = time.Now()

	i.routes = append(i.routes, myRouteInfo)
}

// PostListen called after the server is started, here you can do a lot of staff
// you have the right to access the whole iris' Station also, here you can add more routes and do anything you want, for example start a second server too, an admin web interface!
// for example let's print to the server's stdout the routes we collected...
func (i *myPlugin) PostListen(s *iris.Station) {
	i.container.Printf("From MyPlugin: You have registed %d routes ", len(i.routes))
	//do what ever you want, if you have imagination you can do a lot
}

//

```
Let's register our plugin:
```go

//file main.go
package main

import "github.com/kataras/iris"

func main() {
   iris.Plugin(NewMyPlugin())
   //the plugin is running and caching all these routes
   iris.Get("/", func(c *iris.Context){})
   iris.Post("/login", func(c *iris.Context){})
   iris.Get("/login", func(c *iris.Context){})
   iris.Get("/something", func(c *iris.Context){})

   iris.Listen()
}


```
Output:

>From MyPlugin: You have registed 4 routes

An example of one plugin which is under development is the Iris control, a web interface that gives you control to your server remotely. You can find it's code [here](https://github.com/kataras/iris/tree/development/plugins/iriscontrol)

## Benchmarks

With Intel(R) Core(TM) i7-4710HQ CPU @ 2.50GHz 2.50 HGz and 8GB Ram:

![Benchmark Wizzard 24 March 2016 - comparing the fasters](http://kataras.github.io/iris/assets/benchmarks_all.png)

[See all benchmarks](https://github.com/iris-contrib/go-http-routing-benchmark)

1. Total Operations (Executions)
2. Nanoseconds per operation ns/op
3. Heap Memory B/op
4. Allocations per operation allocs/op



Benchmark name 					| Total Operations 		| ns/op 		| B/op 		| allocs/op
--------------------------------|----------:|----------:|----------:|------:
BenchmarkAce_GithubAll 			| 10000 	| 121206 	| 13792 	| 167
BenchmarkBear_GithubAll 		| 10000 	| 348919 	| 86448 	| 943
BenchmarkBeego_GithubAll 		| 5000 		| 296816 	| 16608 	| 524
BenchmarkBone_GithubAll 		| 500 		| 2502143 	| 548736 	| 7241
BenchmarkDenco_GithubAll 		| 20000 	| 99705 	| 20224 	| 167
BenchmarkEcho_GithubAll 		| 30000 	| 45469 	| 0 		| 0
BenchmarkGin_GithubAll 		| 50000     | 39402     | 0 	    | 0
BenchmarkGocraftWeb_GithubAll 	| 5000 		| 446025 	| 131656 	| 1686
BenchmarkGoji_GithubAll 		| 2000 		| 547698 	| 56112 	| 334
BenchmarkGojiv2_GithubAll 		| 2000 		| 763043 	| 118864 	| 3103
BenchmarkGoJsonRest_GithubAll 	| 5000 		| 538030 	| 134371 	| 2737
BenchmarkGoRestful_GithubAll 	| 100 		| 14870850 	| 837832 	| 6913
BenchmarkGorillaMux_GithubAll 	| 200 		| 6690383 	| 144464 	| 1588
BenchmarkHttpRouter_GithubAll 	| 20000 	| 65653 	| 13792 	| 167
BenchmarkHttpTreeMux_GithubAll 	| 10000 	| 215312 	| 65856 	| 671
**BenchmarkIris_GithubAll** 	| **100000** 	| **20731** 	| **0** 	| **0**
BenchmarkKocha_GithubAll 		| 10000 	| 167209 	| 23304 	| 843
BenchmarkLARS_GithubAll 		| 30000 		| 41069 	| 0 	| 0
BenchmarkMacaron_GithubAll 		| 2000 	| 665038 	| 201138 	| 1803
BenchmarkMartini_GithubAll 		| 100 		| 5433644 	| 228213 	| 2483
BenchmarkPat_GithubAll 			| 300 		| 4210240 	| 1499569 	| 27435
BenchmarkPossum_GithubAll 		| 10000 	| 255114 	| 84448 	| 609
BenchmarkR2router_GithubAll 	| 10000 	| 237113 	| 77328 	| 979
BenchmarkRevel_GithubAll 		| 2000 		| 1150565 	| 337424 	| 5512
BenchmarkRivet_GithubAll 		| 20000 	| 96555 	| 16272 	| 167
BenchmarkTango_GithubAll 		| 5000 		| 417423 	| 87075 	| 2267
BenchmarkTigerTonic_GithubAll 	| 2000 		| 994556 	| 233680 	| 5035
BenchmarkTraffic_GithubAll 		| 200 		| 7770444 	| 2659331 	| 21848
BenchmarkVulcan_GithubAll 		| 5000 		| 292216 	| 19894 	| 609

## Third Party Middlewares


>Note: After v1.1 most of the third party middlewares are incompatible, I'm putting a big effort to convert all of these to work with Iris,
if you want to help please do so (pr).


| Middleware | Author | Description | Tested |
| -----------|--------|-------------|--------|
| [sessions](https://github.com/kataras/iris/tree/development/sessions) | [Ported to Iris](https://github.com/kataras/iris/tree/development/sessions) | Session Management | [Yes](https://github.com/kataras/iris/tree/development/sessions) |
| [Graceful](https://github.com/tylerb/graceful) | [Tyler Bunnell](https://github.com/tylerb) | Graceful HTTP Shutdown | [Yes](https://github.com/kataras/iris/tree/examples/thirdparty_graceful) |
| [gzip](https://github.com/kataras/iris/tree/development/middleware/gzip/) | [Iris](https://github.com/kataras/iris) | GZIP response compression | [Yes](https://github.com/kataras/iris/tree/development/middleware/gzip/README.md) |
| [RestGate](https://github.com/pjebs/restgate) | [Prasanga Siripala](https://github.com/pjebs) | Secure authentication for REST API endpoints | No |
| [secure](https://github.com/unrolled/secure) | [Cory Jacobsen](https://github.com/unrolled) | Middleware that implements a few quick security wins | [Yes](https://github.com/kataras/iris/tree/examples/thirdparty_secure) |
| [JWT Middleware](https://github.com/auth0/go-jwt-middleware) | [Auth0](https://github.com/auth0) | Middleware checks for a JWT on the `Authorization` header on incoming requests and decodes it| No |
| [binding](https://github.com/mholt/binding) | [Matt Holt](https://github.com/mholt) | Data binding from HTTP requests into structs | No |
| [i18n](https://github.com/kataras/iris/tree/development/middleware/i18n) | [Iris](https://github.com/kataras/iris) | Internationalization and Localization | [Yes](https://github.com/kataras/iris/tree/examples/middleware_internationalization_i18n) |
| [logrus](https://github.com/meatballhat/negroni-logrus) | [Dan Buch](https://github.com/meatballhat) | Logrus-based logger | No |
| [render](https://github.com/unrolled/render) | [Cory Jacobsen](https://github.com/unrolled) | Render JSON, XML and HTML templates | No |
| [gorelic](https://github.com/jingweno/negroni-gorelic) | [Jingwen Owen Ou](https://github.com/jingweno) | New Relic agent for Go runtime | No |
| [pongo2](https://github.com/kataras/iris/middleware/pongo2) | [Iris](https://github.com/kataras/iris) | Middleware for [pongo2 templates](https://github.com/flosch/pongo2)| [Yes](https://github.com/kataras/iris/middleware/pongo2/README.md) |
| [oauth2](https://github.com/goincremental/negroni-oauth2) | [David Bochenski](https://github.com/bochenski) | oAuth2 middleware | No |
| [permissions2](https://github.com/xyproto/permissions2) | [Alexander Rødseth](https://github.com/xyproto) | Cookies, users and permissions | No |
| [onthefly](https://github.com/xyproto/onthefly) | [Alexander Rødseth](https://github.com/xyproto) | Generate TinySVG, HTML and CSS on the fly | No |
| [cors](https://github.com/kataras/iris/tree/development/middleware/cors) | [Keuller Magalhaes](https://github.com/keuller) | [Cross Origin Resource Sharing](http://www.w3.org/TR/cors/) (CORS) support | [Yes](https://github.com/kataras/iris/tree/development/middleware/cors) |
| [xrequestid](https://github.com/pilu/xrequestid) | [Andrea Franz](https://github.com/pilu) | Middleware that assigns a random X-Request-Id header to each request | No |
| [VanGoH](https://github.com/auroratechnologies/vangoh) | [Taylor Wrobel](https://github.com/twrobel3) | Configurable [AWS-Style](http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html) HMAC authentication middleware | No |
| [stats](https://github.com/thoas/stats) | [Florent Messa](https://github.com/thoas) | Store information about your web application (response time, etc.) | No |

## Contributors

Thanks goes to the people who have contributed code to this package, see the
[GitHub Contributors page][].

[GitHub Contributors page]: https://github.com/kataras/iris/graphs/contributors



## Community

If you'd like to discuss this package, or ask questions about it, feel free to

* **Chat**: https://gitter.im/kataras/iris

## Guidelines
- Never stop writing the docs.
- Provide full README for examples branch and thirdparty middleware examples.
- Before any commit run -count 50 -benchtime 30s , if performance stays on top then commit else find other way to do the same thing.
- Notice the author of any thirdparty package before I try to port into iris myself, maybe they can do it better.
- Notice author's of middleware, which I'm writing examples for,to take a look, if they don't want to exists in the Iris community, I have to respect them.

## Todo
- [x] [Provide a lighter, with less using bytes,  to save middleware for a route.](https://github.com/kataras/iris/tree/development/handler.go)
- [x] [Create examples.](https://github.com/kataras/iris/tree/examples)
- [x] [Subdomains supports with the same syntax as iris.Get, iris.Post ...](https://github.com/kataras/iris/tree/examples/subdomains_simple)
- [x] [Provide a more detailed benchmark table to the README with all go web frameworks that I know, no just the 6 most famous](https://github.com/kataras/iris/tree/benchmark)
- [x] Convert useful middlewares out there into Iris middlewares, or contact with their authors to do so.
- [ ] Provide automatic HTTPS using https://letsencrypt.org/how-it-works/.
- [ ] Create administration web interface as plugin.
- [ ] Create an easy websocket api.
- [ ] Create a mechanism that scan for Typescript files, compile them on server startup and serve them.

## Articles

* [Ultra-wide framework Go Http routing performance comparison](https://translate.google.com/translate?sl=auto&tl=en&js=y&prev=_t&hl=el&ie=UTF-8&u=http%3A%2F%2Fcolobu.com%2F2016%2F03%2F23%2FGo-HTTP-request-router-and-web-framework-benchmark%2F&edit-text=&act=url)
> According to my  article ( comparative ultra wide frame Go Http routing performance ) on a variety of relatively Go http routing framework, Iris clear winner, its performance far exceeds other Golang http routing framework.

* +1 pending article waiting, after writer's holidays

## License

This project is licensed under the [BSD 3-Clause License](https://opensource.org/licenses/BSD-3-Clause).
License can be found [here](https://github.com/kataras/iris/blob/master/LICENSE).
