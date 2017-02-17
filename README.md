<p align="center">
 <a href="https://godoc.org/gopkg.in/kataras/iris.v6">
	<img width="500" src="https://raw.githubusercontent.com/kataras/iris/master/logo.jpg"
	alt="Logo created by an Iris community member, https://github.com/OneebMalik"
  title="Logo created by an Iris community member, https://github.com/OneebMalik">
 </a>

<br/>

<a href="https://travis-ci.org/kataras/iris"><img src="https://api.travis-ci.org/kataras/iris.svg?branch=v6&style=flat-square" alt="Build Status"></a>

<a href="http://goreportcard.com/report/kataras/iris"><img src="https://img.shields.io/badge/report%20card%20-a%2B-F44336.svg?style=flat-square" alt="http://goreportcard.com/report/kataras/iris"></a>

<a href="https://golang.org"><img src="https://img.shields.io/badge/platform-any-ec2eb4.svg?style=flat-square" alt="Runs everywhere"></a>

<a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-go-6362c2.svg?style=flat-square" alt="Built with GoLang"></a>


<a href="https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=kataras2006%40hotmail%2ecom&lc=GR&item_name=Iris%20web%20framework&item_number=iriswebframeworkdonationid2016&currency_code=EUR&bn=PP%2dDonationsBF%3abtn_donateCC_LG%2egif%3aNonHosted"><img src="https://img.shields.io/badge/open-%20source-thisismycolor.svg?logo=data:image%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxMDAwIDEwMDAiPjxwYXRoIGZpbGw9InJnYigyMjAsMjIwLDIyMCkiIGQ9Ik04ODYuNiwzMDUuM2MtNDUuNywyMDMuMS0xODcsMzEwLjMtNDA5LjYsMzEwLjNoLTc0LjFsLTUxLjUsMzI2LjloLTYybC0zLjIsMjEuMWMtMi4xLDE0LDguNiwyNi40LDIyLjYsMjYuNGgxNTguNWMxOC44LDAsMzQuNy0xMy42LDM3LjctMzIuMmwxLjUtOGwyOS45LTE4OS4zbDEuOS0xMC4zYzIuOS0xOC42LDE4LjktMzIuMiwzNy43LTMyLjJoMjMuNWMxNTMuNSwwLDI3My43LTYyLjQsMzA4LjktMjQyLjdDOTIxLjYsNDA2LjgsOTE2LjcsMzQ4LjYsODg2LjYsMzA1LjN6Ii8%2BPHBhdGggZmlsbD0icmdiKDIyMCwyMjAsMjIwKSIgZD0iTTc5MS45LDgzLjlDNzQ2LjUsMzIuMiw2NjQuNCwxMCw1NTkuNSwxMEgyNTVjLTIxLjQsMC0zOS44LDE1LjUtNDMuMSwzNi44TDg1LDg1MWMtMi41LDE1LjksOS44LDMwLjIsMjUuOCwzMC4ySDI5OWw0Ny4zLTI5OS42bC0xLjUsOS40YzMuMi0yMS4zLDIxLjQtMzYuOCw0Mi45LTM2LjhINDc3YzE3NS41LDAsMzEzLTcxLjIsMzUzLjItMjc3LjVjMS4yLTYuMSwyLjMtMTIuMSwzLjEtMTcuOEM4NDUuMSwxODIuOCw4MzMuMiwxMzAuOCw3OTEuOSw4My45TDc5MS45LDgzLjl6Ii8%2BPC9zdmc%2B" alt="Donation"></a>

<br/>

Iris is an efficient and well-designed toolbox with robust set of features.<br/>Write <b>your own</b>
<b>perfect high-performance web applications</b> <br/>with unlimited potentials and <b>portability</b>.
<br/>

</p>

<p>
<h2>Hashtag #Not_Just_YAWF<!-- hashtag, no title. --></h2>

<a href="https://github.com/kataras/iris/blob/v6/HISTORY.md"><img src="https://img.shields.io/badge/codename-√Νεxτ%20-blue.svg?style=flat-square" alt="CHANGELOG/HISTORY"></a>

<a href="https://github.com/kataras/iris/issues/606"><img src="https://img.shields.io/badge/examples-%20repository-3362c2.svg?style=flat-square" alt="Examples"></a>

<a href="https://godoc.org/gopkg.in/kataras/iris.v6"><img src="https://img.shields.io/badge/docs-%20reference-5272B4.svg?style=flat-square" alt="Docs"></a>

<a href="https://kataras.rocket.chat/channel/iris"><img src="https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square" alt="Chat"></a>

</p>


Iris is fully vendored. That means it is independent of any API changes in the used libraries and **will work seamlessly in the future**!

The size of the executable file is a critical part of the Application Deployment Process,

I made two very simple identical applications, the first written with a famous mini web framework named `gin`(=a Router, with logger, recover and pure Context out-of-the-box support) and the second in `iris`
(=every feature that you will need at the first place is bundled when you install Iris. Including sessions, websockets, typescript support, a cloud-editor, the view engine with 5 different template parsers, two Routers to select from, an end-to-end framework to test your API, more than 60 handy helpers via Context, complete rest API implementation, and cors, basicauth, internalization i18n, logger and recover middleware out-of-the-box).

I ran `go build` for both of them,

 - _gin_ had `9.029 KB` overall file size,
 - _iris_ had `8.505 KB` overall file size!
 - _net/http_ had produced an executable file with `5.380 KB` size.


> The app didn't used any third-party library. If you test the same thing I test and adapt other features like sessions and websockets then the size of `gin` and `net/http` could be doubled while `iris`' overall file size will remain almost the same.


**Applications that are written using Iris produce smaller file size even if they use more features** than a simple router library!

> Q: How is that possible?

> A: The Iris' vendor was done manually without any third-party tool. That means that I had the chance to remove any unnecessary code that Iris never uses internally.


Always follows the latest trends and best practices. Iris is the **Secret To Staying One Step Ahead of Your Competition**.


Iris is a high-performance tool, but it doesn't stops there. Performance depends on your application too, **Iris helps you to make the right choices** on every step.

**Familiar** and easy **API**. Sinatra-like REST API.

Contains examples and documentation for all its features.

Iris is a `low-level access` web framework, you always know what you're doing.

You'll **never miss a thing** from `net/http`, but if you do on some point, no problem because Iris is fully compatible with stdlib, you still have access to `http.ResponseWriter` and `http.Request`, you can adapt any third-party middleware of form `func(http.ResponseWriter, *http.Request, next http.HandlerFunc)` as well.

Iris is a community-driven project, **you suggest and I code**.

Unlike other repositories, this one is **very active**. When you post an issue, you get an answer at the next couple of minutes(hours at the worst). If you find a bug, **I am obliged to fix** that on the same day.

> Q: Why this framework is better than alternatives, does the author is, simply, better than other developers?

> A: Probably not, I don't think that I'm better than anyone else, I still learning every single day. The answer is that I have all the world's time to code for Iris the whole day, I don't have any obligations to anybody else, except you. I'd describe my self as a very dedicated FOSS developer.

Click the below animation to see what people say about Iris.

<a href="https://www.youtube.com/watch?v=jGx0LkuUs4A">
<img src="https://github.com/iris-contrib/website/raw/gh-pages/assets/gif_link_to_yt2.gif" alt="What people say" />
</a>


Installation
-----------

The only requirement is the [Go Programming Language](https://golang.org/dl/), at least 1.8

```bash
$ go get gopkg.in/kataras/iris.v6
```


Overview
-----------

```go
package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/view"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
	app := iris.New()
	app.Adapt(iris.Devlogger()) // adapt a logger which prints all errors to the os.Stdout
	app.Adapt(httprouter.New()) // adapt the adaptors/httprouter or adaptors/gorillamux

	// 5 template engines are supported out-of-the-box:
	//
	// - standard html/template
	// - amber
	// - django
	// - handlebars
	// - pug(jade)
	//
	// Use the html standard engine for all files inside "./views" folder with extension ".html"
	templates := view.HTML("./views", ".html")
	app.Adapt(templates)

	// http://localhost:6200
	// Method: "GET"
	// Render ./views/index.html
	app.Get("/", func(ctx *iris.Context) {
		ctx.Render("index.html", nil)
	})

	// Group routes, optionally: share middleware, template layout and custom http errors.
	userAPI := app.Party("/users", userAPIMiddleware).
		Layout("layouts/userLayout.html")
	{
		// Fire userNotFoundHandler when Not Found
		// inside http://localhost:6200/users/*anything
		userAPI.OnError(404, userNotFoundHandler)

		// http://localhost:6200/users
		// Method: "GET"
		userAPI.Get("/", getAllHandler)

		// http://localhost:6200/users/42
		// Method: "GET"
		userAPI.Get("/:id", getByIDHandler)

		// http://localhost:6200/users
		// Method: "POST"
		userAPI.Post("/", saveUserHandler)
	}

	// Start the server at 127.0.0.1:6200
	app.Listen(":6200")
}

func getByIDHandler(ctx *iris.Context) {
	// take the :id from the path, parse to integer
	// and set it to the new userID local variable.
	userID, _ := ctx.ParamInt("id")

	// userRepo, imaginary database service <- your only job.
	user := userRepo.GetByID(userID)

	// send back a response to the client,
	// .JSON: content type as application/json; charset="utf-8"
	// iris.StatusOK: with 200 http status code.
	//
	// send user as it is or make use of any json valid golang type,
	// like the iris.Map{"username" : user.Username}.
	ctx.JSON(iris.StatusOK, user)
}
```
> TIP: Execute `iris run main.go` to enable hot-reload on .go source code changes.

> TIP: Add `templates.Reload(true)` to monitor the template changes.

Documentation
-----------

 <a href="https://godoc.org/gopkg.in/kataras/iris.v6"><img align="right" width="125" src="https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/book/cover_4.jpg"></a>


 - The most important is to know where to find the [details](https://godoc.org/gopkg.in/kataras/iris.v6)

 - [./adaptors](https://github.com/kataras/iris/tree/v6/adaptors) and [./middleware](https://github.com/kataras/iris/tree/v6/middleware) contains examples for their usage.

 - [HISTORY.md](https://github.com//kataras/iris/tree/v6/HISTORY.md) is your best friend, version migrations are released there.


Testing
------------

You can find End-To-End test examples by navigating to the source code.

A simple test is located to [./httptest/_example/main_test.go](https://github.com/kataras/iris/blob/v6/httptest/_example/main_test.go)


Read more about [gavv's httpexpect](https://github.com/gavv/httpexpect).


FAQ
-----------

Explore [these questions](https://github.com/kataras/iris/issues?q=label%3Aquestion) and join to our [community chat][Chat]!


Philosophy
------------

The Iris philosophy is to provide robust tooling for HTTP, making it a great solution for single page applications, web sites, hybrids, or public HTTP APIs. Keep note that, today, iris is faster than nginx itself.

Iris does not force you to use any specific ORM or template engine. With support for the most used template engines (6+), you can quickly craft the perfect application.


People & Support
------------

The author of Iris is [@kataras](https://github.com/kataras).

The Success of Iris belongs to YOU with your bug reports and feature requests that made this Framework so Unique.

#### Who is kataras?

Hi, my name is Gerasimos Maropoulos and I'm the author of this project, let me put a few words about me.

I started to design Iris the night of the 13 March 2016, some weeks later, iris started to became famous and I have to fix many issues and implement new features, but I didn't have time to work on Iris because I had a part time job and the (software engineering) colleague which I studied.

I wanted to make iris' users proud of the framework they're using, so I decided to interrupt my studies and colleague, two days later I left from my part time job also.

Today I spend all my days and nights coding for Iris, and I'm happy about this, therefore I have zero incoming value.

- Star the project, will help you to follow the upcoming features.
- [Donate](https://github.com/kataras/iris/blob/master/DONATIONS.md), if you can afford any cost.
- Write an article about Iris or even post a Tweet.
- Do Pull Requests on the [iris-contrib](https://github.com/iris-contrib) organisation's repositories, like book and examples.

If you are interested in contributing to the Iris project, please see the document [CONTRIBUTING](https://github.com/kataras/iris/blob/master/.github/CONTRIBUTING.md).


Contact
------------

Besides the fact that we have a [community chat][Chat] for questions or reports and ideas, [stackoverflow](http://stackoverflow.com/) section for generic go+iris questions and the [github issues](https://github.com/kataras/iris/issues) for bug reports and feature requests, you can also contact with me, as a person who is always open to help you:

- [Twitter](https://twitter.com/MakisMaropoulos)
- [Facebook](https://facebook.com/kataras.gopher)
- [Linkedin](https://www.linkedin.com/in/gerasimos-maropoulos)


Versioning
------------

Current: **v6**, code-named as "√Νεxτ"

v5: https://github.com/kataras/iris/tree/5.0.0


License
------------

Unless otherwise noted, the source files are distributed
under the MIT License found in the [LICENSE file](LICENSE).

Note that some optional components that you may use with Iris requires
different license agreements.

[Chat]: https://kataras.rocket.chat/channel/iris
