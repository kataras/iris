# Gapi
version 0.0.2
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/kataras/gapi?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

## Table of Contents

- [Install](#install)
- [Principles](#principles-of-gapi)
- [Introduction](#introduction)
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
	"net/http"

	"github.com/kataras/gapi"
)

func main() {
	api := gapi.New()

	//Register middlewares
	api.Use("/home", log1Home, log2Home)

	//Register routes
	api.Get("/home", homeHandler).
		Get("/about", aboutHandler)

	api.Post("/register", func(res http.ResponseWriter, req *http.Request) {
		println("Message from [POST] /register")
		res.Write([]byte("<h1>Hello from ROUTER ON /register </h1>"))
	})

	println("Server is running at :80")

	//Listen to
	api.Listen(80)

	//Use gapi as middleware is possible too:
	//log.Fatal(http.ListenAndServe(":80", api))

}

func log1Home(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		println("log1 Home middleware here !!")
		next.ServeHTTP(res, req)
	})
}

func log2Home(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		println("log2 Home middleware here !!")
		next.ServeHTTP(res, req)
	})
}

func homeHandler(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("<h1>Hello from ROUTER ON /home </h1>"))
}

func aboutHandler(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("<h1> Hello from ROUTER ON /about </h1>"))
}

```

## Contributors

Thanks goes to the people who have contributed code to this package, see the
[GitHub Contributors page][].

[GitHub Contributors page]: https://github.com/kataras/gapi/graphs/contributors



## Community

If you'd like to discuss this package, or ask questions about it, please use one
of the following:

* **Chat**: https://gitter.im/kataras/gapi


## Todo
*  Middlewares, default and custom.
*  Provide all kind of servers, not only http.
*  Create examples in this repository

## Licence

This project is licensed under the MIT license.

