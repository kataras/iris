## I'm working hard on the [dev](https://github.com/kataras/iris/tree/dev) branch for the next release of Iris. 

Do you remember, last Christmas? I did publish the version 6 with net/http and HTTP/2 support, and you've embraced Iris with so much love, ultimately it was a successful move.

I tend to make surprises by giving you the most unique and useful features, especially on Christmas period.

This year, I intend to give you more gifts.

Don't worry, it will not contain any breaking changes, except of some MVC concepts that are re-designed.

The new Iris' MVC Ecosystem is ready on the [dev/mvc](https://github.com/kataras/iris/tree/dev/mvc). It contains features that you've never saw before, in any programming language framework. It is also, by far, the fastest MVC implementation ever created, very close to raw handlers - it's Iris, it's superior, we couldn't expect something different after all :) Treat that with respect as it treats you :)

I'm doing my bests to get it ready before Christmas.

Star or watch the repository to stay up to date and get ready for the most amazing features!

Yours faithfully, [Gerasimos Maropoulos](https://twitter.com/MakisMaropoulos).

------

# [![Logo created by @santoshanand](logo_white_35_24.png)](https://iris-go.com) Iris

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris)[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)[![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)[![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples)[![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris)[![CLA assistant](https://cla-assistant.io/readme/badge/kataras/iris?style=flat-square)](https://cla-assistant.io/kataras/iris)

Iris is a fast, simple and efficient web framework for Go.

Iris provides a beautifully expressive and easy to use foundation for your next website, API, or distributed app.

Learn what [others say about Iris](https://www.youtube.com/watch?v=jGx0LkuUs4A) and [star](https://github.com/kataras/iris/stargazers) this github repository to stay [up to date](https://facebook.com/iris.framework).

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new.png)](_benchmarks/README_UNIX.md)

<details>
<summary>Benchmarks from third-party source over the rest web frameworks</summary>

![Comparison with other frameworks](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

_Updated at: [Tuesday, 21 November 2017](_benchmarks/README_UNIX.md)_
</details>

## Built with ♥️

We have no doubt you will able to find other web frameworks written in Go
and even put up a real fight to learn and use them for quite some time but
make no mistake, sooner or later you will be using Iris, not because of the ergonomic, high-performant solution that it provides but its well-documented unique features, as these will transform you to a real rockstar geek.

No matter what you're trying to build, Iris covers 
every type of application, from micro services to large monolithic web applications.
It's actually the best piece of software for back-end web developers
you can find online.

Iris may have reached version 8, but we're not stopping there. We have many feature ideas on our board that we're anxious to add and other innovative web development solutions that we're planning to build into Iris.

Accelerated by [KeyCDN](https://www.keycdn.com/), a simple, fast and reliable CDN.

We are developing this project using the best code editor for Golang; [Visual Studio Code](https://code.visualstudio.com/) supported by [Microsoft](https://www.microsoft.com).

If you're coming from [nodejs](https://nodejs.org) world, Iris is the [expressjs](https://github.com/expressjs/express) equivalent for Gophers.

## Table Of Content

* [Installation](#installation)
* [Latest changes](https://github.com/kataras/iris/blob/master/HISTORY.md#th-09-november-2017--v858)
* [Getting started](#getting-started)
* [Learn](_examples/)
    * [MVC (Model View Controller)](_examples/#mvc) **NEW**
    * [Structuring](_examples/#structuring) **NEW**
    * [HTTP Listening](_examples/#http-listening)
    * [Configuration](_examples/#configuration)
    * [Routing, Grouping, Dynamic Path Parameters, "Macros" and Custom Context](_examples/#routing-grouping-dynamic-path-parameters-macros-and-custom-context)
    * [Subdomains](_examples/#subdomains)
    * [Wrap `http.Handler/HandlerFunc`](_examples/#convert-httphandlerhandlerfunc)
    * [View](_examples/#view)
    * [Authentication](_examples/#authentication)
    * [File Server](_examples/#file-server)
    * [How to Read from `context.Request() *http.Request`](_examples/#how-to-read-from-contextrequest-httprequest)
    * [How to Write to `context.ResponseWriter() http.ResponseWriter`](_examples/#how-to-write-to-contextresponsewriter-httpresponsewriter)
    * [Test](_examples/#testing)	
    * [Cache](_examples/#caching)
    * [Sessions](_examples/#sessions)
    * [Websockets](_examples/#websockets)
    * [Miscellaneous](_examples/#miscellaneous)
    * [POC: Convert the medium-sized project "Parrot" from native to Iris](https://github.com/iris-contrib/parrot)
    * [POC: Isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/kataras/iris-starter-kit)
    * [Typescript Automation Tools](typescript/#table-of-contents)
    * [Tutorial: A URL Shortener Service using Go, Iris and Bolt](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)
    * [Tutorial: Online Visitors](_examples/tutorial/online-visitors)
    * [Tutorial: Caddy](_examples/tutorial/caddy)
    * [Tutorial: DropzoneJS Uploader](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
    * [Tutorial:Iris Go Framework + MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)
* [Middleware](middleware/)
* [Dockerize](https://github.com/iris-contrib/cloud-native-go)
* [Contributing](CONTRIBUTING.md)
* [FAQ](FAQ.md)
* [What's next?](#now-you-are-ready-to-move-to-the-next-step-and-get-closer-to-becoming-a-pro-gopher)
* [People](#people)

## Installation

The only requirement is the [Go Programming Language](https://golang.org/dl/)

```sh
$ go get -u github.com/kataras/iris
```

Iris takes advantage of the [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) feature. You get truly reproducible builds, as this method guards against upstream renames and deletes.

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

![overview screen](https://github.com/kataras/build-a-better-web-together/raw/master/overview_screen_1.png)

> Wanna re-start your app automatically when source code changes happens? Install the [rizla](https://github.com/kataras/rizla) tool and run `rizla main.go` instead of `go run main.go`.

Guidelines for bootstrapping applications can be found at the [_examples/structuring](_examples/#structuring).

## Now you are ready to move to the next step and get closer to becoming a pro gopher

Congratulations, since you've made it so far, we've crafted just for you some next level content to turn you into a real pro gopher 😃

> Don't forget to prepare yourself a cup of coffee, or tea, whatever enjoys you the most!

* [Top 6 web frameworks for Go as of 2017](https://blog.usejournal.com/top-6-web-frameworks-for-go-as-of-2017-23270e059c4b)
* [Iris Go Framework + MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)
* [How to build a file upload form using DropzoneJS and Go](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
* [How to display existing files on server using DropzoneJS and Go](https://hackernoon.com/how-to-display-existing-files-on-server-using-dropzonejs-and-go-53e24b57ba19)
* [Iris, a modular web framework](https://medium.com/@corebreaker/iris-web-cd684b4685c7)
* [Go vs .NET Core in terms of HTTP performance](https://medium.com/@kataras/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8)
* [Iris Go vs .NET Core Kestrel in terms of HTTP performance](https://hackernoon.com/iris-go-vs-net-core-kestrel-in-terms-of-http-performance-806195dc93d5)
* [How to Turn an Android Device into a Web Server](https://twitter.com/ThePracticalDev/status/892022594031017988)
* [Deploying a Iris Golang app in hasura](https://medium.com/@HasuraHQ/deploy-an-iris-golang-app-with-backend-apis-in-minutes-25a559bf530b)
* [A URL Shortener Service using Go, Iris and Bolt](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)

## People

The author of Iris is [@kataras](https://github.com/kataras), you can reach him via;

* [Medium](https://medium.com/@kataras)
* [Twitter](https://twitter.com/makismaropoulos)
* [Dev.to](https://dev.to/@kataras)
* [Facebook](https://facebook.com/iris.framework)
* [Mail](mailto:kataras2006@hotmail.com?subject=Iris%20I%20need%20some%20help%20please)

[List of all Authors](AUTHORS)

[List of all Contributors](https://github.com/kataras/iris/graphs/contributors)

Help this project to continue deliver awesome and unique features with the higher code quality as possible by donating any amount via [PayPal](https://www.paypal.me/kataras) or [BTC](https://iris-go.com/v8/donate).

For more information about contributing to the Iris project please check the [CONTRIBUTING.md file](CONTRIBUTING.md).

### We need your help with translations into your native language

Iris needs your help, please think about contributing to the translation of the [README](README.md) and https://iris-go.com, you will be rewarded.

Instructions can be found at: https://github.com/kataras/iris/issues/796

### 03, October 2017 | Iris User Experience Report

Be part of the **first** Iris User Experience Report by submitting a simple form, it won't take more than **2 minutes**.

The form contains some questions that you may need to answer in order to learn more about you; learning more about you helps us to serve you with the best possible way!

https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link

## Sponsors

Support this project by becoming a sponsor. Your logo will show up here with a link to your website. [Become a sponsor](https://opencollective.com/iris#sponsor)

<a href="https://opencollective.com/iris/sponsor/0/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/0/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/1/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/1/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/2/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/2/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/3/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/3/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/4/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/4/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/5/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/5/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/6/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/6/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/7/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/7/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/8/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/8/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/9/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/9/avatar.svg"></a>

## License

Iris is licensed under the 3-Clause BSD [License](LICENSE). Iris is 100% open-source software.

For any questions regarding the license please [contact us](mailto:kataras2006@hotmail.com?subject=Iris%20License).
