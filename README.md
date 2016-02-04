# Gapi  (beta)
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
	"github.com/kataras/gapi"
	"log"
	"net/http"
)

var api = gapi.New()

func main() {

	//Register middlewares
	api.Use(log1, log2)

	//Register routes
	api.Get("/home", homeHandler).
		Post("/register", func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("<h1>Hello from ROUTER ON Post request /register </h1>"))
	})

	api.Get("/profile/{name}", profileHandler) // Parameters
    //api.Get("/profile/{name}/friends", userFriendsHandler) // A parameter can followed by other /path too
    //api.Get("/profile/{name}/friends/{friendId}",userProfileFriendHandler) // Two parameters with path parts also possible 
	println("Server is running at :80")

	//Listen to
	//api.Listen(80)

	//Use gapi as middleware is possible too:
	log.Fatal(http.ListenAndServe(":80", api))

}

func log1(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		println("log1  middleware here !!")
		next.ServeHTTP(res, req)
	})
}

func log2(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		println("log2  middleware here !!")
		next.ServeHTTP(res, req)

	})
}

func homeHandler(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte("<h1>Hello from ROUTER ON /home </h1>"))
}

func profileHandler(res http.ResponseWriter, req *http.Request) {
	//two ways to get the parameters from:
	params := api.Params(req)
	name := params.Get("name") // or params["name"]
   //OR
   //name := api.Param(req,"name")
   

	res.Write([]byte("<h1> Hello from ROUTER ON /profile/"+name+" </h1>"))
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

