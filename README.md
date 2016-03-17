# Iris Web Framework
     This project is under heavy development
<img align="right" width="132" src="http://kataras.github.io/iris/assets/56e4b048f1ee49764ddd78fe_iris_favicon.ico">
[![Build Status](https://travis-ci.org/kataras/iris.svg)](https://travis-ci.org/kataras/iris)
[![GoDoc](https://godoc.org/github.com/kataras/iris?status.svg)](https://godoc.org/github.com/kataras/iris)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/kataras/iris?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

Iris is a very minimal but flexible web framework written in go, providing a robust set of features for building single & multi-page, web applications.

## Table of Contents

- [Install](#install)
- [Principles](#principles-of-iris)
- [Introduction](#introduction)
- [Benchmarks](#benchmarks)
- [Features](#features)
- [Startup & Options](#startup)
- [API](#api)
- [Party](#party)
- [Named Parameters](#named-parameters)
- [Match anything and the Static serve handler](#match-anything-and-the-static-serve-handler)
- [Declaring routes](#declaring-routes)
- [Context](#context)
- [Third Party Middleware](#third-party-middleware)
- [Contributors](#contributors)
- [Community](#community)
- [Todo](#todo)

## Install

```sh
$ go get github.com/kataras/iris
```

### Update
Iris is still in development status, in order to have the latest version update the package every 2-3 days
```sh
$ go get -u github.com/kataras/iris
```

## Principles of iris

- Easy to use

- Robust

- Simplicity Equals Productivity. The best way to make something seem simple is to have it actually be simple. iris's main functionality has clean, classically beautiful APIs

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
	//or for https and http2
	//iris.ListenTLS(":8080","localhost.cert","localhost.key")
	//the cert and key must be in the same path of the executable main server file
}

```
>Note: for macOS, If you are having problems on .Listen then pass only the port "8080" without ':'
## Benchmarks
Benchmark tests were written by 'the standar' way of benchmarking and comparing performance of other routers and frameworks, see [go-http-routing-benchmark](https://github.com/julienschmidt/go-http-routing-benchmark/) .

In order to have safe results, this table was taken from [different source](https://raw.githubusercontent.com/gin-gonic/gin/develop/BENCHMARKS.md) than Iris.

 1. Total Operations
 2. Nanoseconds per Operation (ns/op)
 3. Heap Memory (B/op)
 4. Average Allocations per Operation (allocs/op)

Benchmark name 					| 1 		| 2 		| 3 		| 4
--------------------------------|----------:|----------:|----------:|------:
BenchmarkAce_GithubAll 			| 10000 	| 109482 	| 13792 	| 167
BenchmarkBear_GithubAll 		| 10000 	| 287490 	| 79952 	| 943
BenchmarkBeego_GithubAll 		| 3000 		| 562184 	| 146272 	| 2092
BenchmarkBone_GithubAll 		| 500 		| 2578716 	| 648016 	| 8119
BenchmarkDenco_GithubAll 		| 20000 	| 94955 	| 20224 	| 167
BenchmarkEcho_GithubAll 		| 30000 	| 58705 	| 0 		| 0
BenchmarkGin_GithubAll 		| 30000 | 50991| 0 	| 0
BenchmarkGocraftWeb_GithubAll 	| 5000 		| 449648 	| 133280 	| 1889
BenchmarkGoji_GithubAll 		| 2000 		| 689748 	| 56113 	| 334
BenchmarkGoJsonRest_GithubAll 	| 5000 		| 537769 	| 135995 	| 2940
BenchmarkGoRestful_GithubAll 	| 100 		| 18410628 	| 797236 	| 7725
BenchmarkGorillaMux_GithubAll 	| 200 		| 8036360 	| 153137 	| 1791
BenchmarkHttpRouter_GithubAll 	| 20000 	| 63506 	| 13792 	| 167
BenchmarkHttpTreeMux_GithubAll 	| 10000 	| 165927 	| 56112 	| 334
**BenchmarkIris_GithubAll** 		| **100000** | **19691** | **0** 	| **0**
**BenchmarkIrisNoCache_GithubAll** 		| **30000** | **37833** | **0** 	| **0**
BenchmarkKocha_GithubAll 		| 10000 	| 171362 	| 23304 	| 843
BenchmarkMacaron_GithubAll 		| 2000 		| 817008 	| 224960 	| 2315
BenchmarkMartini_GithubAll 		| 100 		| 12609209 	| 237952 	| 2686
BenchmarkPat_GithubAll 			| 300 		| 4830398 	| 1504101 	| 32222
BenchmarkPossum_GithubAll 		| 10000 	| 301716 	| 97440 	| 812
BenchmarkR2router_GithubAll 	| 10000 	| 270691 	| 77328 	| 1182
BenchmarkRevel_GithubAll 		| 1000 		| 1491919 	| 345553 	| 5918
BenchmarkRivet_GithubAll 		| 10000 	| 283860 	| 84272 	| 1079
BenchmarkTango_GithubAll 		| 5000 		| 473821 	| 87078 	| 2470
BenchmarkTigerTonic_GithubAll 	| 2000 		| 1120131 	| 241088 	| 6052
BenchmarkTraffic_GithubAll 		| 200 		| 8708979 	| 2664762 	| 22390
BenchmarkVulcan_GithubAll 		| 5000 		| 353392 	| 19894 	| 609
BenchmarkZeus_GithubAll 		| 2000 		| 944234 	| 300688 	| 2648

With Intel(R) Core(TM) i7-4710HQ CPU @ 2.50GHz 2.50 HGz and 8GB Ram:

![Benchmark Wizzard Iris vs gin vs martini](http://kataras.github.io/iris/assets/benchmarks_all.png)

## Features

**Parameters in your routing pattern:** Stop parsing the requested URL path, just give the path segment a name and the iris' router delivers the dynamic value to you. Really, path parameters are very cheap.

**Only explicit matches:** With other routers, like http.ServeMux, a requested URL path could match multiple patterns. Therefore they have some awkward pattern priority rules, like longest match or first registered, first matched. By design of this router, a request can only match exactly one or no route. As a result, there are also no unintended matches, which makes it great for SEO and improves the user experience.

**Party of routes:** Combine routes where have same prefix, provide a middleware to this Party, a Party can have other Party too.

**Compatible:** At the end the Iris is just a middleware which acts like router and a small simply web framework, this means that you can you use it side-by-side with your favorite big and well-tested web framework. Iris is fully compatible with the **net/http package.**

**Multi server instances:** Besides the fact that iris has a default main server. You can declare a new iris using the iris.New() func. example: server1:= iris.New(); server1.Get(....); server1.Listen(":9999")

## Startup

As a developer you have three (3) methods to start with Iris.

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
	
	options := iris.StationOptions{
		Profile:            false,
		ProfilePath:        iris.DefaultProfilePath,
		Cache:              true,
		CacheMaxItems:      0,
		CacheResetDuration: 5 * time.Minute,
	}//these are the default values that you can change
	//DefaultProfilePath = "/debug/pprof"
	
	api := iris.Custom(options)
	api.Get("/home",func(c *iris.Context){})
	api.Listen(":8080")
}

```

> Note that with 2. & 3. you **can define and use more than one Iris container** in the
> same app, when it's necessary.

As you can see there are some options that you can chage at your iris declaration, you cannot change them after. If a value not setted then the default used instead.

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

## API
**Use of GET,  POST,  PUT,  DELETE, HEAD, PATCH & OPTIONS**

```go
package main

import (
	"github.com/kataras/iris"
	"net/http"
)

func main() {

	iris.Get("/home", iris.ToHandlerFunc(testGet))
	iris.Post("/login",testPost)
	iris.Put("/add",testPut)
	iris.Delete("/remove",testDelete)
	iris.Head("/testHead",testHead)
	iris.Patch("/testPatch",testPatch)
	iris.Options("/testOptions",testOptions)



	iris.Listen(":8080")
}

func testGet(res http.ResponseWriter, req *http.Request) {
	//...
}

//iris.Context gives more information and control of the route, named parameters, redirect, error handling and render.
func testPost(c *iris.Context) {
	//...
}

//and so on....
```
> Iris is compatible with net/http package over iris.ToHandlerFunc(...) or iris.ToHandler(...) if you wanna use a whole iris.Handler interface. You can use any method you like but, believe me it's easier to pass just a func(c *Context).

## Party

Let's party with Iris web framework!

```go
func main() {
    
    //log everything middleware
    
    iris.UseFunc(func(c *iris.Context) {
		println("[Global log] the requested url path is: ", c.Request.URL.Path)
		c.Next()
	})
    
    // manage all /users
    users := iris.Party("/users")
    {
  	    // provide a  middleware
		users.UseFunc(func(c *iris.Context) {
			println("LOG [/users...] This is the middleware for: ", c.Request.URL.Path)
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
	// MATCH to /hello/anywordhere
	// NOT match to /hello or /hello/ or /hello/anywordhere/something
	iris.Get("/hello/:name", func(c *iris.Context) {
		name := c.Param("name")
		c.Write("Hello " + name)
	})

	// MATCH to /profile/kataras/friends/1
	// NOT match to /profile/ , /profile/kataras ,
	// NOT match to /profile/kataras/friends,  /profile/kataras/friends ,
	// NOT match to /profile/kataras/friends/2/something
	iris.Get("/profile/:fullname/friends/:friendId",
		func(c *iris.Context){
			name:= c.Param("fullname")
			//friendId := c.ParamInt("friendId")
			c.HTML("<b> Hello </b>"+name)
		})

	iris.Listen(":8080")
	//or log.Fatal(http.ListenAndServe(":8080", iris))
}

```

## Match anything and the Static serve handler

Match everything/anything (symbol *withAKeyLikeParameters)
```go
// Will match any request which url's preffix is "/anything/" and has content after that
iris.Get("/anything/*randomName", func(c *iris.Context) { } )
// Match: /anything/whateverhere/whateveragain , /anything/blablabla
// c.Params("randomName") will be /whateverhere/whateveragain, blablabla
// Not Match: /anything , /anything/ , /something
```
Pure http static  file server as handler using **iris.Static("./path/to/the/resources/directory/","path_to_strip_or_nothing")**
```go
// Will match any request which url's preffix is "/public/"
/* and continues with a file whith it's extension which exists inside the os.Gwd()(dot means working directory)+ /static/resources/
*/
iris.Get("/public/*assets", iris.Static("./static/resources/","/public/"))
//Note: strip of the /public/ is handled  by passing the last argument to "/public/"
//you can pass only the first two arguments for no strip path.
```
## Declaring routes
Iris framework has three (3) different forms of functions in order to declare a route's handler and one(1) annotated struct to declare a complete route.


 1. Typical classic handler function, compatible with net/http and other frameworks using iris.ToHandlerFunc
	 *  **func(res http.ResponseWriter, req *http.Request)**
```go
	iris.Get("/user/add", iris.ToHandlerFunc(func(res http.ResponseWriter, req *http.Request)) {

	})
```
 2. Context parameter in function-declaration
	 * **func(c *iris.Context)**

```go
	iris.Get("/user/:userId", func(c *iris.Context) {

	})
```

 3. http.Handler again it can be converted by ToHandlerFunc
	 * **http.Handler**

```go
	iris.Get("/about", iris.ToHandlerFunc(http.HandlerFunc(func(res http.Response, req *req.Request)) {

	}))
```
> Note that all .Get,.Post takes a func(c *Context) as parameter, to pass an iris.Handler use the iris.Handle("/path",handler,"GET")
 4. **'External' annotated struct** which directly implements the Iris Handler interface



```go
///file: userhandler.go
import "github.com/kataras/iris"

type UserRoute struct {
	iris.Handler `get:"/profile/user/:userId"`
}

func (u *UserRoute) Serve(c *iris.Context) {
	defer c.Close()
	userId := c.Param("userId")
	c.RenderFile("user.html", struct{ Message string }{Message: "Hello User with ID: " + userId})
}


///file: main.go

	//...cache the html files
	iris.Templates("src/iristests/templates/**/*.html")
	//...register the handler
	iris.HandleAnnotated(&UserRoute{})
	//...continue writing your wonderful API

```
Personally I use the external struct and the **func(c *iris.Context)** form .
 At the next chapter you will learn what are the benefits of having the  **Context** as parameter to the handler.


## Context

> Variables

 1. **ResponseWriter**
	 - The ResponseWriter is the exactly the same as you used to use with the standar http library.
 2. **Request**
	 - The Request is the pointer of the *Request, is the exactly the same as you used to use with the standar http library.
 3. **Params**
	 - Contains the Named path Parameters, imagine it as a map[string]string which contains all parameters of a request.

>Functions

 0. **Clone()**
	 - Returns a clone of the Context, useful when you want to use the context outscoped for example in goroutines.
 1. **Write(contents string)**
	 - Writes a pure string to the ResponseWriter and sends to the client.
 2. **Param(key string)** returns string
	 - Returns the string representation of the key's  named parameter's value. Registed path= /profile/:name) Requested url is /profile/something where the key argument is the named parameter's key, returns the value  which is 'something' here.
 3. **ParamInt(key string)** returns integer, error
	 - Returns the int representation of the key's  named parameter's value, if something goes wrong the second return value, the error is not nil.
 4. **URLParam(key string)** returns string
	 - Returns the string representation of a requested url parameter (?key=something) where the key argument is the name of, something is the returned value.
 5. **URLParamInt(key string)** returns integer, error
	 - Returns the int representation of  a requested url parameter
 6. **SetCookie(name string, value string)**
	 - Adds a cookie to the request.
 7. **GetCookie(name string)** returns string
	 - Get the cookie value, as string, of a cookie.
 8. **ServeFile(path string)**
	 - This just calls the http.ServeFile, which serves a file given by the path argument  to the client.
 9. **NotFound()**
	 - Sends a http.StatusNotFound with a custom template you defined (if any otherwise the default template is there) to the client.
	 --- *Note: We will learn all about Custom Error Handlers later*.
 10. **Close()**
	 - Calls the Request.Body.Close().

 11. **WriteHTML(status int, contents string) & HTML(contents string)**
	 - WriteHTML: Writes html string with a given http status to the client, it sets the Header with the correct content-type.
	 - HTML: Same as WriteHTML but you don't have to pass a status, it's defaulted to http.StatusOK (200).
 12. **WriteData(status int, binaryData []byte) & Data(binaryData []byte)**
	 - WriteData: Writes binary data with a given http status to the client, it sets the Header with the correct content-type.
	 - Data : Same as WriteData but you don't have to pass a status, it's defaulted to http.StatusOK (200).
 13. **WriteText(status int, contents string) & Text(contents string)**
	 - WriteText: Writes plain text with a given http status to the client, it sets the Header with the correct content-type.
	 - Text: Same as WriteTextbut you don't have to pass a status, it's defaulted to http.StatusOK (200).
 14. **WriteJSON(status int, jsonObject interface{}) & JSON(jsonObject interface{}) returns error**
	 - WriteJSON: Writes json which is converted from structed object(s) with a given http status to the client, it sets the Header with the correct content-type. If something goes wrong then it's returned value which is an error type is not nil. No indent.
 15.  **RenderJSON(jsonObjects ...interface{}) returns error**
	 - RenderJSON: Same as WriteJSON & JSON but with Indent (formated json)	 
	 - JSON: Same as WriteJSON but you don't have to pass a status, it's defaulted to http.StatusOK (200).
 16. **WriteXML(status int, xmlStructs ...interface{}) & XML(xmlStructs ...interface{}) returns error**
	 - WriteXML: Writes writes xml which is converted from struct(s) with a given http status to the client, it sets the Header with the correct content-type. If something goes wrong then it's returned value which is an error type is not nil.
	 - XML: Same as WriteXML but you don't have to pass a status, it's defaulted to http.StatusOK (200).
 17. **RenderFile(file string, pageContext interface{}) returns error**
	 - RenderFile: Renders a file by its name (which a file is saved to the template cache) and a page context passed to the function, default http status is http.StatusOK(200) if the template was found, otherwise http.StatusNotFound(404). If something goes wrong then it's returned value which is an error type is not nil.
 18. **Render(pageContext interface{})  returns error**
	 - Render: Renders the root file template and a context passed to the function, default http status is http.StatusOK(200) if the template was found, otherwise http.StatusNotFound(404). If something goes wrong then it's returned value which is an error type is not nil.
	--- *Note:  We will learn how to add templates at the next chapters*.


**The next chapters are being written this time, they will be published soon, check the docs later [[TODO chapters: Register custom error handlers, Add templates to the route, Declare middlewares]]**


## Third Party Middleware

*The iris tries to supports a lot of middleware out there, you can use them by parsing their handlers, for example: *
```go
iris.UseFunc(func(c *iris.Context) {
		//run the middleware here
		c.Next()
	})
```

>Note: Some of these, may not be work, a lot of them are especially for Negroni and nothing more.

Iris has a middleware system to create it's own middleware and is at a state which tries to find person who are be willing to convert them to Iris middleware or create new. Contact or open an issue if you are interesting.


| Middleware | Author | Description |
| -----------|--------|-------------|
| [sessions](https://github.com/kataras/iris/tree/master/sessions) | [Ported to Iris](https://github.com/kataras/iris/tree/master/sessions) | Session Management |
| [RestGate](https://github.com/pjebs/restgate) | [Prasanga Siripala](https://github.com/pjebs) | Secure authentication for REST API endpoints |
| [Graceful](https://github.com/stretchr/graceful) | [Tyler Bunnell](https://github.com/tylerb) | Graceful HTTP Shutdown |
| [secure](https://github.com/unrolled/secure) | [Cory Jacobsen](https://github.com/unrolled) | Middleware that implements a few quick security wins |
| [JWT Middleware](https://github.com/auth0/go-jwt-middleware) | [Auth0](https://github.com/auth0) | Middleware checks for a JWT on the `Authorization` header on incoming requests and decodes it|
| [binding](https://github.com/mholt/binding) | [Matt Holt](https://github.com/mholt) | Data binding from HTTP requests into structs |
| [logrus](https://github.com/meatballhat/negroni-logrus) | [Dan Buch](https://github.com/meatballhat) | Logrus-based logger |
| [render](https://github.com/unrolled/render) | [Cory Jacobsen](https://github.com/unrolled) | Render JSON, XML and HTML templates |
| [gorelic](https://github.com/jingweno/negroni-gorelic) | [Jingwen Owen Ou](https://github.com/jingweno) | New Relic agent for Go runtime |
| [gzip](https://github.com/phyber/negroni-gzip) | [phyber](https://github.com/phyber) | GZIP response compression |
| [oauth2](https://github.com/goincremental/negroni-oauth2) | [David Bochenski](https://github.com/bochenski) | oAuth2 middleware |
| [permissions2](https://github.com/xyproto/permissions2) | [Alexander Rødseth](https://github.com/xyproto) | Cookies, users and permissions |
| [onthefly](https://github.com/xyproto/onthefly) | [Alexander Rødseth](https://github.com/xyproto) | Generate TinySVG, HTML and CSS on the fly |
| [cors](https://github.com/rs/cors) | [Olivier Poitrey](https://github.com/rs) | [Cross Origin Resource Sharing](http://www.w3.org/TR/cors/) (CORS) support |
| [xrequestid](https://github.com/pilu/xrequestid) | [Andrea Franz](https://github.com/pilu) | Middleware that assigns a random X-Request-Id header to each request |
| [VanGoH](https://github.com/auroratechnologies/vangoh) | [Taylor Wrobel](https://github.com/twrobel3) | Configurable [AWS-Style](http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html) HMAC authentication middleware |
| [stats](https://github.com/thoas/stats) | [Florent Messa](https://github.com/thoas) | Store information about your web application (response time, etc.) |

## Contributors

Thanks goes to the people who have contributed code to this package, see the
[GitHub Contributors page][].

[GitHub Contributors page]: https://github.com/kataras/iris/graphs/contributors



## Community

If you'd like to discuss this package, or ask questions about it, feel free to

* **Chat**: https://gitter.im/kataras/iris


## Todo
- [ ] Never stop writing the docs.
- [x] Provide a lighter, with less using bytes,  to save middleware for a route.
- [x] Create examples in this repository.
- [ ] Convert useful middlewares out there into Iris middlewares, or contact with their authors to do so.
- [ ] Create an easy websocket api also.
- [ ] Create a mechanism that scan for Typescript files, compile them on server startup and serve them.

## Licence

This project is licensed under the MIT license.
