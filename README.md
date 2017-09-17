# ![Logo created by @santoshanand](logo_white_35_24.png) Iris 

<!-- <p align="center">
<img src="iris_the_new_machine.gif">
</p> -->

Iris is a fast, simple and efficient micro web framework for Go. It provides a beautifully expressive and easy to use foundation for your next website, API, or distributed app.

We have no doubt you will able to find other web frameworks written in Go
and even put up a real fight to learn and use them for quite some time but
make no mistake, sooner or later you will be using Iris, not because of the ergonomic, high-performant solution that it provides but its well-documented unique features, as these will transform you to a real rockstar geek. 

No matter what you're trying to build, Iris covers 
every type of application, from micro services to large monolithic web applications.
It's actually the best piece of software for back-end web developers
you can find online.

Iris may have reached version 8, but we're not stopping there. We have many feature ideas on our board that we're anxious to add and other innovative web development solutions that we're planning to build into Iris.

[Star or watch](https://github.com/kataras/iris/stargazers) this repository to stay updated.

**Accelerated by [KeyCDN](https://www.keycdn.com/)**, A Simple, Fast and Reliable CDN.

<!-- [![total used by](https://iris-go.com/graph?style=flat-square)](https://iris-go.com/graph) -->
[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)
[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris)
[![github issues](https://img.shields.io/github/issues/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aopen+is%3Aissue)
[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)
[![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)
[![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples)
[![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris)

<!-- has issues, the img.shields.io's API is problematic for this resource: [![issue stats](https://img.shields.io/issuestats/i/github/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed) -->
<!-- [![godocs](https://img.shields.io/badge/godocs-8.x.x-0366d6.svg?style=flat-square)](https://godoc.org/github.com/kataras/iris) -->
<p>
<img src="https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png" alt="Third-party source for transparency." />
</p>

### 📑 Table of contents

* [Installation](#-installation)
* [Latest changes](https://github.com/kataras/iris/blob/master/HISTORY.md#fr-15-september-2017--v842)
* [Learn](#-learn)
    * [HTTP Listening](_examples/#http-listening)
    * [Configuration](_examples/#configuration)
    * [Routing, Grouping, Dynamic Path Parameters, "Macros" and Custom Context](_examples/#routing-grouping-dynamic-path-parameters-macros-and-custom-context)
    * [MVC (Model View Controller)](_examples/#mvc) **NEW**
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
    * [Typescript Automation Tools](typescript/#table-of-contents)
    * [Tutorial: Online Visitors](_examples/tutorial/online-visitors)
    * [Tutorial: Caddy](_examples/tutorial/caddy)
* [Middleware](middleware/)
* [Dockerize](https://github.com/iris-contrib/cloud-native-go)
* [Community & Support](#-community)
* [Blogs](https://iris-go.com/v8/blogs)
    - [Iris Go vs .NET Core Kestrel in terms of HTTP performance](https://hackernoon.com/iris-go-vs-net-core-kestrel-in-terms-of-http-performance-806195dc93d5)
    - [Go vs .NET Core in terms of HTTP performance](https://medium.com/@kataras/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8)
    - [Iris, a modular web framework](https://medium.com/@corebreaker/iris-web-cd684b4685c7)
    - [Deploying a Iris Golang app in hasura](https://medium.com/@HasuraHQ/deploy-an-iris-golang-app-with-backend-apis-in-minutes-25a559bf530b)
    - [How to Turn an Android Device into a Web Server](https://twitter.com/ThePracticalDev/status/892022594031017988)
    - [A URL Shortener Service using Go, Iris and Bolt](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)
    - [Why I preferred Go over Node.js for simple Web Application](https://medium.com/@tigranbs/why-i-preferred-go-over-node-js-for-simple-web-application-d4a549e979b9)
* [Versioning](#-version)
* [People](#-people)

### 🚀 Installation

The only requirement is the [Go Programming Language](https://golang.org/dl/), at least version 1.8 but **1.9** is highly recommended.

```sh
$ go get -u github.com/kataras/iris
```

> _iris_ takes advantage of the [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) feature. You get truly reproducible builds, as this method guards against upstream renames and deletes.

```go
// file: main.go
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
    app.Get("/user/{id:long}", func(ctx iris.Context) {
        userID, _ := ctx.Params().GetInt64("id")
        ctx.Writef("User ID: %d", userID)
    })

    // Start the server using a network address.
    app.Run(iris.Addr(":8080"))
}
```

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

<details>
<summary>Hello World with Go 1.8</summary>

Iris declares all of its type alias at the same file in order to be easy to be discovered. 

> If you just upgraded to go 1.9 from 1.8 you can always search for a compatible type alias at the [context.go](context.go) file and opposite, if you use go 1.8 and you're new to Iris you can see [that](context.go) file to see the compatible packages. 

If Go 1.8 remains the basic host for your go apps then you should declare and use the `github.com/kataras/iris/context` package on your source file's imports statement.

```go
package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./templates", ".html"))

	app.Get("/", func(ctx context.Context) {
		ctx.ViewData("message", "Hello world!")
		ctx.View("hello.html")
	})

	app.Run(iris.Addr(":8080"))
}
```

</details>

<details>
<summary>Fan of the MVC Architectural Pattern? Click here</summary>

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    app.RegisterView(iris.HTML("./views", ".html"))

    app.Controller("/", new(Controller))

    app.Run(iris.Addr(":8080"))
}

type Controller struct {
    iris.Controller
}

// Method:    GET
// Resource:  http://localhost:8080
func (c *Controller) Get() {
    c.Data["message"] = "Hello world!"
    c.Tmpl = "hello.html"
}

// Method:    GET
// Resource:  http://localhost:8080/user/42
func (c *Controller) GetUserBy(id int64) {
    c.Ctx.Writef("User ID: %d", id)
}
```

</details>

<details>
<summary>Why a new web framework</summary>

### Why

Go is a great technology stack for building scalable, web-based, back-end systems for web 
applications. 

When you think about building web applications and web APIs, or simply building HTTP servers in Go, does your mind go to the standard net/http package?
Then you have to deal with some common situations like dynamic routing (a.k.a parameterized), security and authentication, real-time communication and many other issues that net/http doesn't solve. 

The net/http package is not complete enough to quickly build well-designed back-end web systems. When you realize this, you might be thinking along these lines:

- Ok, the net/http package doesn't suit me, but there are so many frameworks, which one will work for me?!
- Each one of them tells me that it is the best. I don't know what to do!

##### The truth

I did some deep research and benchmarks with 'wrk' and 'ab' in order to choose which framework would suit me and my new project. The results, sadly, were really disappointing to me.

I started wondering if golang wasn't as fast on the web as I had read... but, before I let Golang go and continued to develop with nodejs, I told myself:

> '**Makis, don't lose hope, give at least a chance to Golang. Try to build something totally new without basing it off the "slow" code you saw earlier; learn the secrets of this language and make *others* follow your steps!**'.

These are the words I told myself that day [**13 March 2016**]. 

The same day, later the night, I was reading a book about Greek mythology. I saw an ancient goddess' name and was inspired immediately to give a name to this new web framework (which I had already started writing) - **Iris**.

 I'm still here [because Iris has succeed in being the fastest go web framework](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

[![](http://comments.iris-go.com/screens/comment37.png)](https://twitter.com/ThePracticalDev/status/892022594031017988) 

![](https://docs.iris-go.com/comment36.png)

![](https://docs.iris-go.com/comment35.png)

![](https://docs.iris-go.com/comment34.png)

![](https://docs.iris-go.com/comment2.png)

![](https://docs.iris-go.com/comment30.png)

![](https://docs.iris-go.com/comment31.png)

![](https://docs.iris-go.com/comment32.png)

![](https://docs.iris-go.com/comment33.png)

![](https://docs.iris-go.com/comment26.png)

![](https://docs.iris-go.com/comment1.png)

![](https://docs.iris-go.com/comment3.png)

![](https://docs.iris-go.com/comment8.png)

![](https://docs.iris-go.com/comment12.png)

![](https://docs.iris-go.com/comment13.png)

![](https://docs.iris-go.com/comment14.png)

![](https://docs.iris-go.com/comment17.png)

![](https://docs.iris-go.com/comment21.png)

![](https://docs.iris-go.com/comment22.png)

![](https://docs.iris-go.com/comment24.png)

![](https://docs.iris-go.com/comment27.png)

---- 

_iris_ is easy, it has a familiar API while in the same has far more features than [Gin](https://github.com/gin-gonic/gin) or [Martini](https://github.com/go-martini/martini).

You own your code —it will never generate (unfamiliar) code for you, like [Beego](https://github.com/astaxie/beego), [Revel](https://github.com/revel/revel) and [Buffalo](https://github.com/gobuffalo/buffalo) do.

It's not just-another-router but its overall performance is equivalent with something like [httprouter](https://github.com/julienschmidt/httprouter).

Unlike [fasthttp](https://github.com/valyala/fasthttp), iris provides full HTTP/2 support for free.

Compared to the rest open source projects, this one is very active and you get answers almost immediately.

### 🔥 Hot Features

- Focus on high performance
- Easy Fluent API
- Highly customizable
- Robust routing and middleware ecosystem
    * Build RESTful APIs with iris unique expressionist path interpreter
	* Dynamic path parameterized or wildcard routes are not conflict with static routes 
	* Remove trailing slash from the URL with option to redirect
	* Virtual hosts and subdomains made easy
	* Group API's and static or even dynamic subdomains
	* MVC [**NEW**](_examples/mvc)
	* `net/http` and `negroni-like` handlers are compatible via `iris.FromStd` 
	* Register custom handlers for any HTTP error
	* Transactions and rollback when you need it
	* Cache the response when you need it
	* A single function to serve your embedded assets, always compatible with `go-bindata`
	* HTTP to HTTPS
 	* HTTP to HTTPS WWW
	* [learn the reasons that differ from what you've seen so far](_examples/#routing-grouping-dynamic-path-parameters-macros-and-custom-context)
- Context
	* Highly scalable rich content render (Markdown, JSON, JSONP, XML...)
	* Body binders and handy functions to send HTTP responses
	* Limit request body
	* Serve static resources or embedded assets
	* Localization i18N
	* Compression (Gzip is built'n)
- Authentication
	* Basic Authentication
	* OAuth, OAuth2 supporting 27+ popular websites
	* JWT
- Server
	* Automatically install and serve certificates from https://letsencrypt.org when serving via TLS
	* Gracefully shutdown by-default
	* Register on shutdown, error or interrupt events
	* Attach more than one server, fully compatible with `net/http#Server`
- View system: supporting 5 template engines. Fully compatible with `html/template`
- HTTP Sessions library [you can still use your favorite if you want to]
- Websocket library, its API similar to socket.io [you can still use your favorite if you want to]
- Hot Reload on source code changes[*](https://github.com/kataras/rizla)
- Typescript integration + Web IDE
- And many other things that will surprise you

</details>

### 📖 Learn

<a href="https://github.com/kataras/iris/_examples" alt="documentation and examples">
	<img align="right" src="learn.jpg" width="125" />
</a>

The _iris_ philosophy is to provide robust tooling for HTTP, making it a great solution for single page applications, web sites, hybrids, or public HTTP APIs. Keep note that, today, iris is faster than apache+nginx itself.

_iris_ does not force you to use any specific ORM. With support for the most popular template engines, websocket server and a fast sessions manager you can quickly craft your perfect application.


The awesome _iris_ community is always adding new examples, [_examples](_examples/) is a great place to get started!

Read the [godocs](https://godoc.org/github.com/kataras/iris) for a better understanding.

### 👥 Community

Join the welcoming community of fellow _iris_ developers in [rocket.chat](https://kataras.rocket.chat/channel/iris)

- [Post](http://support.iris-go.com) a feature request or report a bug
- :star: and watch the public [repository](https://github.com/kataras/iris/stargazers), will keep you up to date
- :earth_americas: publish [an article](https://medium.com/search?q=iris) or share a [tweet](https://twitter.com/hashtag/golang) about your personal experience with iris.


The most useful community repository for _iris_ developers is the 
[iris-contrib/middleware](https://github.com/iris-contrib/middleware) which contains some HTTP handlers that can help you finish a lot of your tasks even easier. Feel free to push your own middleware there!

```sh
$ go get -u github.com/iris-contrib/middleware/...
```

#### 📈 One and a half years...

[![total used by](https://iris-go.com/graph?style=flat-square)]() 

Iris exceeded all expectations, started as one-man project.

- 7300 github stars
- 778 github forks
- 1m total views at its documentation
- ~819$ at donations, small amount for the work we put here but it's a good start
- ~557 reported bugs fixed
- ~30 community feature requests have been implemented

### 📌 Version

Current: [VERSION](VERSION)

Each new release is pushed to the master. It stays there until the next version. When a next version is released then the previous version goes to its own branch with `gopkg.in` as its import path (and its own vendor folder), in order to keep it working "for-ever".

Changelog of the current version can be found at the [HISTORY](HISTORY.md) file.

#### Should I upgrade my iris?

Developers are not forced to use the latest _iris_ version, they can use any version in production, they can update at any time they want.

Testers should upgrade immediately, if you're willing to use _iris_ in production you can wait a little more longer, transaction should be as safe as possible.

#### Where can I find older versions?

Previous versions can be found at [releases page](https://github.com/kataras/iris/releases).

<!--
### 😃 Get Hired

Below you'll find a list of open positions that require at least **experience with the Iris web framework**.

| Company | Position | Job Details |
| -----------|--------|-------------|
| Kudo, an Indonesian startup technology company | Application Programming Interface Developer | Navigate to: https://glints.id/opportunities/jobs/5553 |

Employers that are looking for brilliant Software Engineers with good experience on Go Programming Language and Iris can put their startup's or company's name here or, if privacy is the key, [contact with us](mailto:kataras2006@hotmail.com?subject=Employer%20That%20Hires%20Smart%20Devs) to suggest some good and well-tested freelancers that suits your needs.

-->

### 🥇 People

The original author of _Iris_ is [@kataras](https://github.com/kataras), you can reach him via
- [Medium](https://medium.com/@kataras)
- [Twitter](https://twitter.com/makismaropoulos)
- [Dev.to](https://dev.to/@kataras)
- [Facebook](https://facebook.com/kataras.gopher)
- [Mail](mailto:kataras2006@hotmail.com?subject=Iris%20I%20need%20some%20help%20please)

[List of all Authors](AUTHORS)

[List of all Contributors](https://github.com/kataras/iris/graphs/contributors)

Help this project to continue deliver awesome and unique features with the higher code quality as possible by donating any amount

[![](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=kataras2006%40hotmail%2ecom&lc=GR&item_name=Iris%20web%20framework&item_number=iriswebframeworkdonationid2016&currency_code=EUR&bn=PP%2dDonationsBF%3abtn_donateCC_LG%2egif%3aNonHosted)

## License

This software is licensed under the open-source 3-Clause BSD.

You can find the license file [here](LICENSE), for any questions regarding the license please [contact](mailto:kataras2006@hotmail.com?subject=Iris%20License) us.