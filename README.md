# ![Logo created by @santoshanand](logo_white_35_24.png) Iris

Iris is a fast, simple and efficient micro web framework for Go. It provides a beautifully expressive and easy to use foundation for your next website, API, or distributed app.

### About our User Experience Report

Three days ago, _at 03 October_, we announced the first [Iris User Experience form-based Report](https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link) to let us learn more about you and any issues that troubles you with Iris (if any).

At overall, the results (so far) are very promising, high number of participations and the answers to the questions are near to the green feedback we were receiving over the past months from Gophers worldwide via our [rocket chat](https://chat.iris-go.com) and [author's twitter](https://twitter.com/makismaropoulos). **If you didn't complete the form yet, [please do so](https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link) as soon as possible!**

However, as everything in life; nothing goes as expected, people are strange, we programmers even more. The last part of the form has a text area which participiations can add any "questions or comments", there we saw one comment that surprised me, in the bad sense. We respect all individual singularities the same, we do not discriminate between people. The data are anonymous, so the only place to answer to that person is, _surprisingly_, here!

<details>
<summary>"I admire your dedication to iris and I am in love with its speed..."</summary>

The comment was "I admire your dedication to iris and I am in love with its speed but.. I've read some things on that blog and blablabla..." you get the point, at the first we were happy and suddenly we saw that "but... I've" and we broke xD.

My answer to this is clear in simple words so that anyone can understand; Did you really believed those unsubstantial things even if you could take some time off to read the source code?🤔

Iris was the one of the top github trending projects written in Go Programming Language for the 2016 and the most trending web framework in the globe. We couldn't even imagine that we will be the receiver of countless "[thank you for iris, finally a web framework I can work on](https://twitter.com/_mgale/status/818591490305761280)" comments from hundreds strangers around the globe!

Please do research before reading and assimilate everything, those blog spots are not always telling the whole truth, they are not so innocent :)

Especially those from that kid which do not correspond to reality;

```go
/* start */
```

First of all, that article **is reffering 1.5 years ago**, to pretend that this article speaks for the present is hilariously ridiculous! Iris is on version 8 now and it's not a router any more, it's a fully featured web framework with its own ecosystem.

1. Iris does NOT use any third-party code inside it, like "httprouter" or "fasthttp". Just navigate to the source code. If you care about historical things you can search the project but it doesn't matter because the internal implementation of Iris changed a lot of times, a lot more than its public API changes:P.
2. Iris makes use of its own routing mechanisms with a unique **language interpreter** in order to serve even the most demanding of us `/user/{id:int min(2)}`, `/alphabetical/{param:string regexp(^[a-zA-Z ]+$)}` et cetera.
3. Iris has its own unique MVC architectural parser with hurt-breaking performance.
4. Was it possible to do all those things and [much more](_examples) before Iris? Exaclty. Iris offers you all these for free, plus the unmatched performance.
5. Iris is the result of hundreds(or thousands(?)) of hours of **FREE and UNPAID** work. There are people who actually found a decent job because of Iris. Thousands of Gophers are watching or/and helping to make Iris even better, the silent majority loves Iris even more.

That 23 years old, inhibited boy, who published that post had played you with the most immoral way! Reading the Iris' source code doesn't cost you a thing! Iris is free to use for everyone, Iris is an open-source software, no hidden spots. **Don't stuck on the past, get over that, Iris has succeed, move on now.**

```go
/* end */
```
</details>

_Psst_, we've produced a small video about your feelings regrating to Iris! You can watch the whole video at https://www.youtube.com/watch?v=jGx0LkuUs4A.

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)
[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris)
[![github issues](https://img.shields.io/github/issues/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aopen+is%3Aissue)
[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)
[![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)
[![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples)
[![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris)

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new-gray.png)](_benchmarks)

</p>

## Installation

The only requirement is the [Go Programming Language](https://golang.org/dl/), at least version 1.9.

```sh
$ go get -u github.com/kataras/iris
```

* Iris takes advantage of the [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) feature. You get truly reproducible builds, as this method guards against upstream renames and deletes.

* [Latest changes](https://github.com/kataras/iris/blob/master/HISTORY.md#fr-06-october-2017--v845)

## Getting Started

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // Load all templates from the "./views" folder
    // where extension is ".html" and parse them
    // using the standard `html/template` package.
    app.RegisterView(iris.HTML("./views", ".html"))

    // Method:    GET
    // Resource:  http://localhost:8080
    app.Get("/", func(ctx iris.Context) {
        // Bind: {{.message}} with "Hello world!"
        ctx.ViewData("message", "Hello world!")
        // Render template file: ./views/hello.html
        ctx.View("hello.html")
    })

    // Method:    GET
    // Resource:  http://localhost:8080/user/42
    //
    // Need to use a custom regexp instead?
    // Easy;
    // Just mark the parameter's type to 'string'
    // which accepts anything and make use of
    // its `regexp` macro function, i.e:
    // app.Get("/user/{id:string regexp(^[0-9]+$)}")
    app.Get("/user/{id:long}", func(ctx iris.Context) {
        userID, _ := ctx.Params().GetInt64("id")
        ctx.Writef("User ID: %d", userID)
    })

    // Start the server using a network address.
    app.Run(iris.Addr(":8080"))
}
```

> Learn more about path parameter's types by clicking [here](https://github.com/kataras/iris/blob/master/_examples/routing/dynamic-path/main.go#L31).

```html
<!-- file: ./views/hello.html -->
<html>
<head>
    <title>Hello Page</title>
</head>
<body>
    <h1>{{.message}}</h1>
</body>
</html>
```

```sh
$ go run main.go
> Now listening on: http://localhost:8080
> Application started. Press CTRL+C to shut down.
```

### Quick MVC Tutorial

```go
package main

import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/mvc"
)

func main() {
    app := iris.New()

    app.Controller("/helloworld", new(HelloWorldController))

    app.Run(iris.Addr("localhost:8080"))
}

type HelloWorldController struct {
    mvc.Controller

    // [ your fields here ]
    // Request lifecycle data
    // Models
    // Database
    // Global properties
}

//
// GET: /helloworld

func (c *HelloWorldController) Get() {
    c.Ctx.Text("This is my default action...")
}

//
// GET: /helloworld/welcome

func (c *HelloWorldController) GetWelcome() {
    c.Ctx.HTML("This is the <b>GetWelcome</b> action func...")
}

//
// GET: /helloworld/welcome/{name:string}/{numTimes:int}

func (c *HelloWorldController) GetWelcomeBy(name string, numTimes int) {
    c.Ctx.Writef("Hello %s, NumTimes is: %d", name, numTimes)
}
```

> The [_examples/mvc](_examples/mvc) and [mvc/controller_test.go](https://github.com/kataras/iris/blob/master/mvc/controller_test.go) files explain each feature with simple paradigms, they show how you can take advandage of the Iris MVC Binder, Iris MVC Models and many more...

Every `exported` func prefixed with an HTTP Method(`Get`, `Post`, `Put`, `Delete`...) in a controller is callable as an HTTP endpoint. In the sample above, all funcs writes a string to the response. Note the comments preceding each method.

An HTTP endpoint is a targetable URL in the web application, such as `http://localhost:8080/helloworld`, and combines the protocol used: HTTP, the network location of the web server (including the TCP port): `localhost:8080` and the target URI `/helloworld`.

The first comment states this is an [HTTP GET](https://www.w3schools.com/tags/ref_httpmethods.asp) method that is invoked by appending "/helloworld" to the base URL. The second comment specifies an [HTTP GET](https://www.w3schools.com/tags/ref_httpmethods.asp) method that is invoked by appending "/helloworld/welcome/" to the URL.

Controller knows how to handle the "name" and "numTimes" at `GetWelcomeBy`, because of the `By` keyword, and builds the dynamic route without boilerplate; the third comment specifies an [HTTP GET](https://www.w3schools.com/tags/ref_httpmethods.asp) dynamic method that is invoked by any URL that starts with "/helloworld/welcome" and followed by two more path parts, the first one can accept any value and the second can accept only numbers, i,e: "http://localhost:8080/helloworld/welcome/golang/32719", otherwise a [404 Not Found HTTP Error](https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html#sec10.4.5) will be sent to the client instead.

## 😃 Do you like what you see so far?

> Prepare yourself a cup of coffee, or tea, whatever enjoys you the most!

- [How to build a file upload form using DropzoneJS and Go](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
- [How to display existing files on server using DropzoneJS and Go](https://hackernoon.com/how-to-display-existing-files-on-server-using-dropzonejs-and-go-53e24b57ba19)
- [Iris Go vs .NET Core Kestrel in terms of HTTP performance](https://hackernoon.com/iris-go-vs-net-core-kestrel-in-terms-of-http-performance-806195dc93d5)
- [Go vs .NET Core in terms of HTTP performance](https://medium.com/@kataras/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8)
- [Iris, a modular web framework](https://medium.com/@corebreaker/iris-web-cd684b4685c7)
- [Deploying a Iris Golang app in hasura](https://medium.com/@HasuraHQ/deploy-an-iris-golang-app-with-backend-apis-in-minutes-25a559bf530b)
- [How to Turn an Android Device into a Web Server](https://twitter.com/ThePracticalDev/status/892022594031017988)
- [A URL Shortener Service using Go, Iris and Bolt](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)
- [Why I preferred Go over Node.js for simple Web Application](https://medium.com/@tigranbs/why-i-preferred-go-over-node-js-for-simple-web-application-d4a549e979b9)


Take some time, `don't say we didn't warn you`,  and continue your journey by [navigating to the bigger README page](README_BIG.md).

## License

Iris is licensed under the 3-Clause BSD [License](LICENSE). Iris is 100% open-source software.