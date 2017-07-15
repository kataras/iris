# ![Logo created by @santoshanand](logo_white_35_24.png) Iris 

Iris is a fast, simple and efficient micro web framework for Go. It provides a beautifully expressive and easy to use foundation for your next website, API, or distributed app.

Web applications powered by Iris run everywhere, even [from an android device](https://medium.com/@kataras/how-to-turn-an-android-device-into-a-web-server-9816b28ab199).

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)
[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris)
[![godocs](https://img.shields.io/badge/godocs-8.x.x-0366d6.svg?style=flat-square)](https://godoc.org/github.com/kataras/iris)
[![get support](https://img.shields.io/badge/get-support-cccc00.svg?style=flat-square)](http://support.iris-go.com)
[![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples)
[![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris)

<!--
# Mo, 10 July 2017 | v8.0.0

### 📈 One and a half years with Iris and You...

- 7070 github stars
- 749 github forks
- 1m total views at its documentation
- ~800$ at donations (there're a lot for a golang open-source project, thanks to you)
- ~550 reported bugs fixed
- ~30 community feature requests have been implemented

### 🔥 Reborn

As you may have heard I have huge responsibilities on my new position at Dubai nowdays, therefore I don't have the needed time to work on this project anymore.

After almost a month of negotiations and searching I succeed to find a decent software engineer to continue my work on the open source community.

The leadership of this, open-source, repository was transfered to [hiveminded](https://github.com/hiveminded).

These types of projects need heart and sacrifices to continue offer the best developer experience like a paid software, please do support him as you did with me!

> Please [contact](https://kataras.rocket.chat/channel/iris) with the project team if you want to help at the development process!

### 📑 Table of contents

<a href="https://github.com/kataras/iris/_examples" alt="documentation and examples">
	<img align="right" src="learn.jpg" width="125" />
</a>

* [Installation](#-installation)
* [Latest changes](https://github.com/kataras/iris/blob/master/HISTORY.md#su-15-july-2017--v802)
* [Learn](#-learn)
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
	* [Cache](cache/#table-of-contents)
	* [Sessions](sessions/#table-of-contents)
	* [Websockets](websocket/#table-of-contents)
	* [Miscellaneous](_examples/#miscellaneous)
	* [Typescript Automation Tools](typescript/#table-of-contents)
	* [Tutorial: Online Visitors](_examples/tutorial/online-visitors)
	* [Tutorial: URL Shortener using BoltDB](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)
	* [Tutorial: How to turn your Android Device into a fully featured Web Server](https://medium.com/@kataras/how-to-turn-an-android-device-into-a-web-server-9816b28ab199)
* [Middleware](middleware/)
* [Dockerize](https://github.com/iris-contrib/cloud-native-go)
* [Philosophy](#-philosophy)
* [Support](#-support)
* [Versioning](#-version)
    * [When should I upgrade?](#should-i-upgrade-my-iris)
    * [Where can I find older versions?](#where-can-i-find-older-versions)
* [People](#-people)

### 🚀 Installation

The only requirement is the [Go Programming Language](https://golang.org/dl/), at least version 1.8

```sh
$ go get -u github.com/kataras/iris
```

> _iris_ takes advantage of the [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) feature. You get truly reproducible builds, as this method guards against upstream renames and deletes.

```go
// file: main.go
package main
import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/context"
)
func main() {
    app := iris.New()
    // Load all templates from the "./templates" folder
    // where extension is ".html" and parse them
    // using the standard `html/template` package.
    app.RegisterView(iris.HTML("./templates", ".html"))

    // Method:    GET
    // Resource:  http://localhost:8080
    app.Get("/", func(ctx context.Context) {
        // Bind: {{.message}} with "Hello world!"
        ctx.ViewData("message", "Hello world!")
        // Render template file: ./templates/hello.html
        ctx.View("hello.html")
    })

    // Start the server using a network address and block.
    app.Run(iris.Addr(":8080"))
}
```
```html
<!-- file: ./templates/hello.html -->
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
<summary>Hello World with Go 1.9</summary>

If you've installed Go 1.9 then you can omit the `github.com/kataras/iris/context` package from the imports statement.

```go
// +build go1.9

package main

import "github.com/kataras/iris"

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./templates", ".html"))
	
	app.Get("/", func(ctx iris.Context) {
		ctx.ViewData("message", "Hello world!")
		ctx.View("hello.html")
	})

	app.Run(iris.Addr(":8080"))
}
```

We expect Go version 1.9 to be released in August, however you can install Go 1.9 beta today.

### Installing Go 1.9beta2
 
1. Go to https://golang.org/dl/#go1.9beta2
2. Download a compatible, with your OS, archieve, i.e `go1.9beta2.windows-amd64.zip`
3. Unzip the contents of `go1.9beta2.windows-amd64.zip/go` folder to your $GOROOT, i.e `C:\Go`
4. Open a terminal and execute `go version`, it should output the go1.9beta2 version, i.e:
```sh
C:\Users\hiveminded>go version
go version go1.9beta2 windows/amd64
```

</details>

<details>
<summary>Why another new web framework?</summary>

_iris_ is easy, it has a familiar API while in the same has far more features than [Gin](https://github.com/gin-gonic/gin) or [Martini](https://github.com/go-martini/martini).

You own your code —it will never generate (unfamiliar) code for you, like [Beego](https://github.com/astaxie/beego), [Revel](https://github.com/revel/revel) and [Buffalo](https://github.com/gobuffalo/buffalo) do.

It's not just-another-router but its overall performance is equivalent with something like [httprouter](https://github.com/julienschmidt/httprouter).

Unlike [fasthttp](https://github.com/valyala/fasthttp), iris provides full HTTP/2 support for free.

Compared to the rest open source projects, this one is very active and you get answers almost immediately.

</details>

### 👥 Community

The most useful community repository for _iris_ developers is the 
[iris-contrib/middleware](https://github.com/iris-contrib/middleware) which contains some HTTP handlers that can help you finish a lot of your tasks even easier.

```sh
$ go get -u github.com/iris-contrib/middleware/...
```

> Feel free to put your own middleware there!

Join the welcoming community of fellow _iris_ developers in [rocket.chat](https://kataras.rocket.chat/channel/iris).

### 📖 Learn

The awesome _iris_ community is always adding new examples, [_examples](_examples/) is a great place to get started!

Read the [godocs](https://godoc.org/github.com/kataras/iris) for a better understanding.

### 🤔 Philosophy

The _iris_ philosophy is to provide robust tooling for HTTP, making it a great solution for single page applications, web sites, hybrids, or public HTTP APIs. Keep note that, today, iris is faster than apache+nginx itself.

_iris_ does not force you to use any specific ORM. With support for the most popular template engines, websocket server and a fast sessions manager you can quickly craft your perfect application.

### 💙 Support

- [Post](http://support.iris-go.com) a feature request or report a bug
- :star: and watch the public [repository](https://github.com/kataras/iris/stargazers), will keep you up to date
- :earth_americas: publish [an article](https://medium.com/search?q=iris) or share a [tweet](https://twitter.com/hashtag/golang) about your personal experience with iris

### 📌 Version

Current: **8.0.2**

Each new release is pushed to the master. It stays there until the next version. When a next version is released then the previous version goes to its own branch with `gopkg.in` as its import path (and its own vendor folder), in order to keep it working "for-ever".

Changelog of the current version can be found at the [HISTORY](HISTORY.md) file.

#### Should I upgrade my iris?

Developers are not forced to use the latest _iris_ version, they can use any version in production, they can update at any time they want.

Testers should upgrade immediately, if you're willing to use _iris_ in production you can wait a little more longer, transaction should be as safe as possible.

#### Where can I find older versions?

Previous versions can be found at [releases page](https://github.com/kataras/iris/releases).

### 🥇 People

The original author of _iris_ is [Gerasimos Maropoulos](https://github.com/kataras)

The current lead maintainer is [Bill Qeras, Jr.](https://github.com/hiveminded)

[List of all contributors](https://github.com/kataras/iris/graphs/contributors)

[![](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=kataras2006%40hotmail%2ecom&lc=GR&item_name=Iris%20web%20framework&item_number=iriswebframeworkdonationid2016&currency_code=EUR&bn=PP%2dDonationsBF%3abtn_donateCC_LG%2egif%3aNonHosted)