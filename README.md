<p align="center">
 <a href="https://www.gitbook.com/book/kataras/iris/details">
	<img width="500" src="https://raw.githubusercontent.com/kataras/iris/master/logo.jpg"
	alt="Logo created by an Iris community member, https://github.com/OneebMalik"
  title="Logo created by an Iris community member, https://github.com/OneebMalik">
 </a>

<br/>


<a href="https://travis-ci.org/kataras/iris"><img src="https://api.travis-ci.org/kataras/iris.svg?branch=v6&style=flat-square" alt="Build Status"></a>

<a href="http://goreportcard.com/report/kataras/iris"><img src="https://img.shields.io/badge/report%20card%20-a%2B-F44336.svg?style=flat-square" alt="http://goreportcard.com/report/kataras/iris"></a>

<a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-go-6362c2.svg?style=flat-square" alt="Built with GoLang"></a>

<a href="https://golang.org"><img src="https://img.shields.io/badge/platform-any-ec2eb4.svg?style=flat-square" alt="Cross framework"></a>

<a href="https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=kataras2006%40hotmail%2ecom&lc=GR&item_name=Iris%20web%20framework&item_number=iriswebframeworkdonationid2016&currency_code=EUR&bn=PP%2dDonationsBF%3abtn_donateCC_LG%2egif%3aNonHosted"><img src="https://img.shields.io/badge/open-%20source-thisismycolor.svg?logo=data:image%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxMDAwIDEwMDAiPjxwYXRoIGZpbGw9InJnYigyMjAsMjIwLDIyMCkiIGQ9Ik04ODYuNiwzMDUuM2MtNDUuNywyMDMuMS0xODcsMzEwLjMtNDA5LjYsMzEwLjNoLTc0LjFsLTUxLjUsMzI2LjloLTYybC0zLjIsMjEuMWMtMi4xLDE0LDguNiwyNi40LDIyLjYsMjYuNGgxNTguNWMxOC44LDAsMzQuNy0xMy42LDM3LjctMzIuMmwxLjUtOGwyOS45LTE4OS4zbDEuOS0xMC4zYzIuOS0xOC42LDE4LjktMzIuMiwzNy43LTMyLjJoMjMuNWMxNTMuNSwwLDI3My43LTYyLjQsMzA4LjktMjQyLjdDOTIxLjYsNDA2LjgsOTE2LjcsMzQ4LjYsODg2LjYsMzA1LjN6Ii8%2BPHBhdGggZmlsbD0icmdiKDIyMCwyMjAsMjIwKSIgZD0iTTc5MS45LDgzLjlDNzQ2LjUsMzIuMiw2NjQuNCwxMCw1NTkuNSwxMEgyNTVjLTIxLjQsMC0zOS44LDE1LjUtNDMuMSwzNi44TDg1LDg1MWMtMi41LDE1LjksOS44LDMwLjIsMjUuOCwzMC4ySDI5OWw0Ny4zLTI5OS42bC0xLjUsOS40YzMuMi0yMS4zLDIxLjQtMzYuOCw0Mi45LTM2LjhINDc3YzE3NS41LDAsMzEzLTcxLjIsMzUzLjItMjc3LjVjMS4yLTYuMSwyLjMtMTIuMSwzLjEtMTcuOEM4NDUuMSwxODIuOCw4MzMuMiwxMzAuOCw3OTEuOSw4My45TDc5MS45LDgzLjl6Ii8%2BPC9zdmc%2B" alt="Donation"></a>

<br/>


<a href="https://github.com/kataras/iris/blob/v6/HISTORY.md"><img src="https://img.shields.io/badge/%20version%20-%206.2.0%20-blue.svg?style=flat-square" alt="CHANGELOG/HISTORY"></a>

<a href="https://github.com/iris-contrib/examples"><img src="https://img.shields.io/badge/%20examples-repository-3362c2.svg?style=flat-square" alt="Examples"></a>

<a href="https://docs.iris-go.com"><img src="https://img.shields.io/badge/%20docs-reference-5272B4.svg?style=flat-square" alt="Practical Guide/Docs"></a>

<a href="https://kataras.rocket.chat/channel/iris"><img src="https://img.shields.io/badge/%20community-chat-00BCD4.svg?style=flat-square" alt="Chat"></a><br/>
<br/>

Iris provides efficient and well-designed toolbox with robust set of features to<br/> <b>create your own
perfect high-performance web application</b> <br/>with unlimited portability using the Go Programming Language.

<br/><br/>
<b>Easy</b> to <a href="https://github.com/kataras/iris/tree/v6/docs">learn</a>  while it's highly customizable,
ideally suited for <br/> both experienced and novice developers.<br/><br/>

If you're coming from <a href="https://nodejs.org/en/">Node.js</a> world, this is the <a href="https://github.com/expressjs/express">expressjs</a> for the <a href="https://golang.org">Go Programming Language.</a>
<br/><br/>



<a href="https://www.youtube.com/watch?v=jGx0LkuUs4A">
<img src="https://github.com/iris-contrib/website/raw/gh-pages/assets/gif_link_to_yt2.gif" alt="What people say" />
</a>

<a href="https://www.youtube.com/watch?v=jGx0LkuUs4A">
<img src="https://github.com/iris-contrib/website/raw/gh-pages/assets/gif_link_to_yt.gif" alt="What people say" />
</a>


</p>


Installation
-----------

The only requirement is the [Go Programming Language](https://golang.org/dl/), at least v1.7.

```bash
$ go get gopkg.in/kataras/iris.v6
```


Overview
-----------

```go
package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/adaptors/view"
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

 <a href="https://www.gitbook.com/book/kataras/iris/details"><img align="right" width="125" src="https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/book/cover_4.jpg"></a>


 - The most important is to read [the practical guide](https://docs.iris-go.com/)

 - Read [godocs](https://godoc.org/github.com/kataras/iris) for the details

 - Navigate through [examples](https://github.com/iris-contrib/examples)

 - [HISTORY.md](https://github.com//kataras/iris/tree/v6/HISTORY.md) file is your best friend.


Testing
------------

You can find RESTFUL test examples by navigating to the following links:

- [gavv/_examples/iris_test.go](https://github.com/gavv/httpexpect/blob/master/_examples/iris_test.go).
- [./http_test.go](https://github.com/kataras/iris/blob/v6/http_test.go).
- [./context_test.go](https://github.com/kataras/iris/blob/v6/context_test.go).


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

Current: **v6.2.0**

v5: https://github.com/kataras/iris/tree/5.0.0


License
------------

Unless otherwise noted, the source files are distributed
under the MIT License found in the [LICENSE file](LICENSE).

Note that some optional components that you may use with Iris requires
different license agreements.

[Chat]: https://kataras.rocket.chat/channel/iris
