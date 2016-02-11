# Gapi  (beta)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/kataras/gapi?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

## Table of Contents

- [Install](#install)
- [Principles](#principles-of-gapi)
- [Introduction](#introduction)
- [Third Party Middleware](#third-party-middleware)
- [Contributors](#contributors)
- [Community](#community)
- [Todo](#todo)

## Install

```sh
$ go get github.com/kataras/gapi
```
## Principles of Gapi

- Easy to use

- Robust

- Simplicity Equals Productivity. The best way to make something seem simple is to have it actually be simple. Gapi's main functionality has clean, classically beautiful APIs

## Introduction

A very minimal but flexible golang web application framework, providing a robust set of features for building single & multi-page, web applications.

```go
package main

import (
    "github.com/kataras/gapi"
	"log"
	"net/http"
)

var api = gapi.New()

func main() {
	//register global middleware
	api.UseFunc(globalLog)

	api.Post("/register", func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("<h1>Hello from ROUTER ON Post request /register </h1>"))
	})

	api.Get("/profile/user/{name}/details/{something}", profileHandler) // Parameters
	//or if you want a route to listen to more than one method than one you can do that:
	api.Route("/api/user/{userId(int)}", func(c *gapi.Context) {
		c.Write("<h1> TEST CONTEXT userId =  " + c.Param("userId") + " </h1>")
	}).Methods(gapi.HTTPMethods.GET, gapi.HTTPMethods.POST) // or .ALL if you want all (get,post,head,put,options,delete,patch...)

	//register route, it's 'controller' homeHandler and its middleware log1,
	//middleware will run first and if next fn is exists and executed
	//or no next fn exists in middleware then will continue to homeHandler
	api.Get("/home", homeHandler).UseFunc(log1)

	println("Server is running at :80")

	//Listen to (runs on top of the http.NewServeMux())
	log.Fatal(api.Listen(80))
	//Use gapi as middleware is possible too (runs independed):
	//log.Fatal(http.ListenAndServe(":80", api))

}

func globalLog(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	println("GLOBAL LOG  middleware here !!")
	next.ServeHTTP(res, req)
}

func log1(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	println("log1  middleware here !!")
	next.ServeHTTP(res, req)

}

func log2() http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		println("log2  middleware here !!")
		//next.ServeHTTP(res, req)

	})
}

func homeHandler(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("<h1>Hello from ROUTER ON /home </h1>"))
}

func profileHandler(res http.ResponseWriter, req *http.Request) {

	params := api.Params(req)
	name := params.Get("name") // or params["name"]
	//or name := api.Param(req,"name")

	res.Write([]byte("<h1> Hello from ROUTER ON /profile/" + name + " </h1>"))
}

```


## Third Party Middleware
*The gapi is re-written in order to support all middlewares that are already exists for [Negroni](https://github.com/codegangsta/negroni) middleware*
 
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

[GitHub Contributors page]: https://github.com/kataras/gapi/graphs/contributors



## Community

If you'd like to discuss this package, or ask questions about it, please use one
of the following:

* **Chat**: https://gitter.im/kataras/gapi


## Todo
*  Query parameters
*  Provide a way to define a content Renderer in the Context
*  Provide all kind of servers, not only http.
*  Create examples in this repository

## Licence

This project is licensed under the MIT license.

