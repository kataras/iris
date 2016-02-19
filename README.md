# iris
<img align="right" width="248" src="http://nodets.com/iris_logo.gif">
[![Build Status](https://travis-ci.org/kataras/iris.svg)](https://travis-ci.org/kataras/iris)
[![GoDoc](https://godoc.org/github.com/kataras/iris?status.svg)](https://godoc.org/github.com/kataras/iris)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/kataras/iris?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

Iris is a very minimal but flexible web framework written in go, providing a robust set of features for building single & multi-page, web applications.

## Table of Contents

- [Install](#install)
- [Principles](#principles-of-iris)
- [Introduction](#introduction)
- [Features](#features)
- [API](#api)
- [Named Parameters](#named-parameters)
- [Third Party Middleware](#third-party-middleware)
- [Contributors](#contributors)
- [Community](#community)
- [Todo](#todo)

## Install

```sh
$ go get github.com/kataras/iris
```
## Principles of iris

- Easy to use

- Robust

- Simplicity Equals Productivity. The best way to make something seem simple is to have it actually be simple. iris's main functionality has clean, classically beautiful APIs

## Introduction
The word iris means "rainbow" in Greek. Iris was the name of the Greek goddess of the rainbow.
Iris is a very minimal but flexible golang http middleware & standalone web application framework, providing a robust set of features for building single & multi-page, web applications.

```go
package main

import "github.com/kataras/iris"

func main() {
	iris.Get("/hello", func(r *iris.Renderer) {
		r.HTML("<b> Hello </b>")
	})
	iris.Listen(8080)
}

```

## Features

**Only explicit matches:** With other routers, like http.ServeMux, a requested URL path could match multiple patterns. Therefore they have some awkward pattern priority rules, like longest match or first registered, first matched. By design of this router, a request can only match exactly one or no route. As a result, there are also no unintended matches, which makes it great for SEO and improves the user experience.

**Parameters in your routing pattern:** Stop parsing the requested URL path, just give the path segment a name and the router delivers the dynamic value to you. Because of the design of the router, path parameters are very cheap.

**Perfect for APIs:** The router design encourages to build sensible, hierarchical RESTful APIs. Moreover it has builtin native support for OPTIONS requests and 405 Method Not Allowed replies.

**Compatible:** At the end the iris is just a middleware which acts like router and a small simply web framework, this means that you can you use it side-by-side with your favorite big and well-tested web framework. Iris is fully compatible with the **net/http package.**

**Miltiple servers :** Besides the fact that iris has a default main server, which only created only if you call any global function (e.x iris.Get). You can declare a new iris using the iris.New() func. server1:= iris.New(); server1.Get(....); server1.Listen(9999)



## API
**Use of GET,  POST,  PUT,  DELETE, HEAD, PATCH & OPTIONS**

```go
package main

import (
	"github.com/kataras/iris"
	"net/http"
)

func main() {
	iris.Get("/home", testGet)
	iris.Post("/login",testPost)
	iris.Put("/add",testPut)
	iris.Delete("/remove",testDelete)
	iris.Head("/testHead",testHead)
	iris.Patch("/testPatch",testPatch)
	iris.Options("/testOptions",testOptions)

	iris.Listen(8080)
}

//iris is fully compatible with net/http package
func testGet(res http.ResponseWriter, req *http.Request) {
	//...
}

func testPost(c *iris.Context) {
	//...
}

func testPut(r *iris.Renderer) {
	//...
}

func testDelete(c *iris.Context, r *iris.Renderer) {
	//...
}
//and so on....
```

## Named Parameters

Named parameters are just custom paths to your routes, you can access them for each request using context's **c.Param("nameoftheparameter")** or **iris.Param(request,"nameoftheparam")**. Get all, as pair (**map[string]string**) using **c.Params()** or **iris.Params(request)**

By default the :name is matched to any word, you can use custom regex using parenthesis after the parameter example: /user/:name([a-z]+) this will match the route only if the second part of the route after /user/ is a word which it's letters are lowercase only.

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
	// /profile/kataras/friends,  /profile/kataras/friends ,
	// /profile/kataras/friends/string , /profile/anumb3r/friends/1
	iris.Get("/users/:fullname([a-zA-Z]+)/friends/:friendId(int)",
		func(c *iris.Context, r *iris.Renderer){
			name:= c.Param("fullname")
			friendId := c.ParamInt("friendId")
			r.HTML("<b> Hello </b>"+name)
		})

	iris.Listen(8080)
	//or log.Fatal(http.ListenAndServe(":8080", iris))
}

```

**Note:** Since this router has only explicit matches, you can not register static routes and parameters for the same path segment. For example you can not register the patterns /user/new and /user/:user for the same request method at the same time. The routing of different request methods is independent from each other.

## Third Party Middleware
*The iris is re-written in order to support all middlewares that are already exists for [Negroni](https://github.com/codegangsta/negroni) middleware*

Here is a current list of compatible middlware.


| Middleware | Author | Description |
| -----------|--------|-------------|
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
| [sessions](https://github.com/goincremental/negroni-sessions) | [David Bochenski](https://github.com/bochenski) | Session Management |
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

If you'd like to discuss this package, or ask questions about it, please use one
of the following:

* **Chat**: https://gitter.im/kataras/iris


## Todo
*  Complete the documents
*  Query parameters
*  Create examples in this repository

## Licence

This project is licensed under the MIT license.
