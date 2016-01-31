# Gapi
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/kataras/gapi?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

## Table of Contents

- [Install](#install)
- [Principles](#principles-of-mysql-live)
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
	"net/http"
)

func main() {
	server, router := gapi.New()

	router.
		If("/home").Then(homeHandler()).
		If("/about").Then(aboutHandler()).
		If("/contact").Then(return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("<h1>Chat with the gapi at https://gitter.im/kataras/gapi  | /contact </h1>"))
		}))

	server.Listen(8080) //or server.Listen(":8080")

}

func homeHandler() http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("<h1>Hello world, from gapi | /home </h1>"))
	})
}

func aboutHandler() http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("<h1> About gapi | /about </h1>"))
	})
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


## This is my first golang package, so ... Todo
*  Middlewares, default and custom.
*  Provide all kind of servers, not only http.
*  Create examples in this repository

## Licence

This project is licensed under the MIT license.

