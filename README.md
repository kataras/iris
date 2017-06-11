# ![Logo created by @santoshanand](logo_white_35_24.png) Iris 

A fast, cross-platform and efficient web framework with robust set of well-designed features, written entirely in Go.

[![Build status](https://api.travis-ci.org/kataras/iris.svg?branch=master&style=flat-square)](https://travis-ci.org/kataras/iris)
[![Report card](https://img.shields.io/badge/report%20card%20-a%2B-F44336.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris)
[![Support forum](https://img.shields.io/badge/support-page-ec2eb4.svg?style=flat-square)](http://support.iris-go.com)
[![Examples](https://img.shields.io/badge/howto-examples-3362c2.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples#table-of-contents)
[![Godocs](https://img.shields.io/badge/7.0.5-%20documentation-5272B4.svg?style=flat-square)](https://godoc.org/github.com/kataras/iris)
[![Chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris)
[![Buy me a cup of coffee](https://img.shields.io/badge/support-%20open--source-F4A460.svg?logo=data:image%2Fsvg%2Bxml%3Bbase64%2CPHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHZpZXdCb3g9IjAgMCAxMDAwIDEwMDAiPjxwYXRoIGZpbGw9InJnYigyMjAsMjIwLDIyMCkiIGQ9Ik04ODYuNiwzMDUuM2MtNDUuNywyMDMuMS0xODcsMzEwLjMtNDA5LjYsMzEwLjNoLTc0LjFsLTUxLjUsMzI2LjloLTYybC0zLjIsMjEuMWMtMi4xLDE0LDguNiwyNi40LDIyLjYsMjYuNGgxNTguNWMxOC44LDAsMzQuNy0xMy42LDM3LjctMzIuMmwxLjUtOGwyOS45LTE4OS4zbDEuOS0xMC4zYzIuOS0xOC42LDE4LjktMzIuMiwzNy43LTMyLjJoMjMuNWMxNTMuNSwwLDI3My43LTYyLjQsMzA4LjktMjQyLjdDOTIxLjYsNDA2LjgsOTE2LjcsMzQ4LjYsODg2LjYsMzA1LjN6Ii8%2BPHBhdGggZmlsbD0icmdiKDIyMCwyMjAsMjIwKSIgZD0iTTc5MS45LDgzLjlDNzQ2LjUsMzIuMiw2NjQuNCwxMCw1NTkuNSwxMEgyNTVjLTIxLjQsMC0zOS44LDE1LjUtNDMuMSwzNi44TDg1LDg1MWMtMi41LDE1LjksOS44LDMwLjIsMjUuOCwzMC4ySDI5OWw0Ny4zLTI5OS42bC0xLjUsOS40YzMuMi0yMS4zLDIxLjQtMzYuOCw0Mi45LTM2LjhINDc3YzE3NS41LDAsMzEzLTcxLjIsMzUzLjItMjc3LjVjMS4yLTYuMSwyLjMtMTIuMSwzLjEtMTcuOEM4NDUuMSwxODIuOCw4MzMuMiwxMzAuOCw3OTEuOSw4My45TDc5MS45LDgzLjl6Ii8%2BPC9zdmc%2B)](https://github.com/kataras/iris#buy-me-a-cup-of-coffee)
 
<p>
<img src="https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png" alt="This benchmark measures results from 'real-world' instead of 'hello-world' application source code. | Last Update At: July 21, 2016. | Shows: Processing Time Horizontal Graph. | Who did:  Third-party source. Transparent achievement." />
</p>

Build your own web applications and portable APIs with the highest performance and countless potentials.

If you're coming from [Node.js](https://nodejs.org) world, this is the [expressjs](https://github.com/expressjs/express)++ equivalent for the [Go Programming Language](https://golang.org).

Installation
-----------

The only requirement is the [Go Programming Language](https://golang.org/dl/), at least version 1.8

```sh
$ go get -u github.com/kataras/iris
```

> Iris uses the [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) feature, so you get truly reproducible builds, as this method guards against upstream renames and deletes.
For further installation support, please navigate [here](http://support.iris-go.com/d/16-how-to-install-iris-web-framework).

```go
package main

import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/context"
    "github.com/kataras/iris/view"
)

// User is just a bindable object structure.
type User struct {
    Username  string `json:"username"`
    Firstname string `json:"firstname"`
    Lastname  string `json:"lastname"`
    City      string `json:"city"`
    Age       int    `json:"age"`
}

func main() {
    app := iris.New()
    
    // Define templates using the std html/template engine.
    // Parse and load all files inside "./views" folder with ".html" file extension.
    // Reload the templates on each request (development mode).
    app.AttachView(view.HTML("./views", ".html").Reload(true))
    
    // Regster custom handler for specific http errors.
    app.OnErrorCode(iris.StatusInternalServerError, func(ctx context.Context) {
    	// .Values are used to communicate between handlers, middleware.
    	errMessage := ctx.Values().GetString("error")
    	if errMessage != "" {
    		ctx.Writef("Internal server error: %s", errMessage)
    		return
    	}
    
    	ctx.Writef("(Unexpected) internal server error")
    })
    
    app.Use(func(ctx context.Context) {
    	ctx.Application().Log("Begin request for path: %s", ctx.Path())
    	ctx.Next()
    })
    
    // app.Done(func(ctx context.Context) {})
    
    // Method POST: http://localhost:8080/decode
    app.Post("/decode", func(ctx context.Context) {
    	var user User
    	ctx.ReadJSON(&user)
    	ctx.Writef("%s %s is %d years old and comes from %s", user.Firstname, user.Lastname, user.Age, user.City)
    })
    
    // Method GET: http://localhost:8080/encode
    app.Get("/encode", func(ctx context.Context) {
    	doe := User{
    		Username:  "Johndoe",
    		Firstname: "John",
    		Lastname:  "Doe",
    		City:      "Neither FBI knows!!!",
    		Age:       25,
    	}
    
    	ctx.JSON(doe)
    })
    
    // Method GET: http://localhost:8080/profile/anytypeofstring
    app.Get("/profile/{username:string}", profileByUsername)
    
    usersRoutes := app.Party("/users", logThisMiddleware)
    {
    	// Method GET: http://localhost:8080/users/42
    	usersRoutes.Get("/{id:int min(1)}", getUserByID)
    	// Method POST: http://localhost:8080/users/create
    	usersRoutes.Post("/create", createUser)
    }
    
    // Listen for incoming HTTP/1.x & HTTP/2 clients on localhost port 8080.
    app.Run(iris.Addr(":8080"), iris.WithCharset("UTF-8"))
}

func logThisMiddleware(ctx context.Context) {
    ctx.Application().Log("Path: %s | IP: %s", ctx.Path(), ctx.RemoteAddr())
    
    // .Next is required to move forward to the chain of handlers,
    // if missing then it stops the execution at this handler.
    ctx.Next()
}

func profileByUsername(ctx context.Context) {
    // .Params are used to get dynamic path parameters.
    username := ctx.Params().Get("username")
    ctx.ViewData("Username", username)
    // renders "./views/users/profile.html"
    // with {{ .Username }} equals to the username dynamic path parameter.
    ctx.View("users/profile.html")
}

func getUserByID(ctx context.Context) {
    userID := ctx.Params().Get("id") // Or convert directly using: .Values().GetInt/GetInt64 etc...
    // your own db fetch here instead of user :=...
    user := User{Username: "username" + userID}
    
    ctx.XML(user)
}

func createUser(ctx context.Context) {
    var user User
    err := ctx.ReadForm(&user)
    if err != nil {
    	ctx.Values().Set("error", "creating user, read and parse form failed. "+err.Error())
    	ctx.StatusCode(iris.StatusInternalServerError)
    	return
    }
    // renders "./views/users/create_verification.html"
    // with {{ . }} equals to the User object, i.e {{ .Username }} , {{ .Firstname}} etc...
    ctx.ViewData("", user)
    ctx.View("users/create_verification.html")
}
```

### Reload on source code changes

```sh
$ go get -u github.com/kataras/rizla
$ cd $GOPATH/src/mywebapp
$ rizla main.go
```

> Psst: Wanna go to [_examples](https://github.com/kataras/iris/tree/master/_examples) to see more code-snippets?

<details>
<summary>Legends</summary>

I'm sorry for taking this personally but I really need to thank each one of them because they stood up [♡](https://github.com/kataras/iris#support) for me when others tried to "bullying" my personality in order to deflame Iris.

All of us should read and respect the official [golang](https://golang.org/conduct) and [iris](https://github.com/iris-contrib/community-board/blob/master/CODE-OF-CONDUCT.md) community **Code of Conduct**. This type of commitment and communication is the way of making Go great.

<!-- on each chance, i.e when iris posts were trending on sites like dzone and medium they spamming an awful full of lies and deflamation blog post. We all should read and respect the official golang's and iris' Code of Conduct. -->

[Juan Sebastián Suárez Valencia](https://github.com/Juanses) donated 20 EUR at September 11 of 2016

[Bob Lee](https://github.com/li3p) donated 20 EUR at September 16 of 2016

[Celso Luiz](https://github.com/celsosz) donated 50 EUR at September 29 of 2016

[Ankur Srivastava](https://github.com/ansrivas) donated 20 EUR at October 2 of 2016

[Damon Zhao](https://github.com/se77en) donated 20 EUR at October 21 of 2016

[exponity - consulting & digital transformation](https://github.com/exponity) donated 30 EUR at November 4 of 2016

[Thomas Fritz](https://github.com/thomasfr) donated 25 EUR at Jenuary 8 of 2017

[Thanos V.](http://mykonosbiennale.com/) donated 20 EUR at Jenuary 16 of 2017

[George Opritescu](https://github.com/International) donated 20 EUR at February 7 of 2017

[Lex Tang](https://github.com/lexrus) donated 20 EUR at February 22 of 2017

[Conrad Steenberg](https://github.com/hengestone) donated 25 EUR at March 23 of 2017

<p>

<a href="https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7">
<img width="300" src="http://iris-go.com/comment28.png" />
</a>

<a href="https://twitter.com/gelnior/status/769100480706379776">
<img width="300" src="http://iris-go.com/comment27.png" />
</a>

<br/>

<a href="https://www.youtube.com/watch?v=jGx0LkuUs4A">
<img  width="300" src="https://github.com/iris-contrib/website/raw/gh-pages/assets/gif_link_to_yt.gif" alt="What people say" />
</a>

<a href="https://www.youtube.com/watch?v=jGx0LkuUs4A">
<img width ="300" src="https://github.com/iris-contrib/website/raw/gh-pages/assets/gif_link_to_yt2.gif" alt="What people say" />
</a>

<br/>

<a href="https://twitter.com/_mgale/status/818591490305761280">
<img width="300" width="630" src="http://iris-go.com/comment25.png" />
</a>

<a href="https://twitter.com/MeAlex07/status/822799954188075008">
<img width="300" width="627" src="http://iris-go.com/comment26.png" />
</a>

</p>

</details>

Feature Overview
-----------

- Focus on high performance
- Build RESTful APIs with our expressionist path syntax, i.e `{userid:int min(1)}`, `{asset:path}`, `{custom regexp([a-z]+)}`
- Automatically install and serve certificates from https://letsencrypt.org
- Robust routing and middleware ecosystem
- Request-Scoped Transactions
- Group API's and subdomains with wildcard support
- Body binding for JSON, XML, Forms, can be extended to use your own custom binders
- More than 50 handy functions to send HTTP responses
- View system: supporting more than 6+ template engines, with prerenders. You can still use your favorite
- Define virtual hosts and (wildcard) subdomains with path level routing
- Graceful shutdown
- Limit request body
- Localization i18N
- Serve static and embedded files
- Cache
- Log requests
- Customizable format and output for the logger
- Customizable HTTP errors
- Compression (Gzip)
- Authentication
 - OAuth, OAuth2 supporting 27+ popular websites
 - JWT
 - Basic Authentication
- HTTP Sessions
- Add / Remove trailing slash from the URL with option to redirect
- Redirect requests
 - HTTP to HTTPS
 - HTTP to HTTPS WWW
 - HTTP to HTTPS non WWW
 - Non WWW to WWW
 - WWW to non WWW
- Highly scalable rich content render (Markdown, JSON, JSONP, XML...)
- Websocket-only API similar to socket.io
- Hot Reload on source code changes
- Typescript integration + Web IDE
- Checks for updates at startup
- Highly customizable
- Feels like you used iris forever, thanks to its Fluent API
- And many others...


Documentation
-----------

 <a href="https://github.com/kataras/iris/tree/master/_examples#table-of-contents"><img width="155" src="https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/book/cover_4.jpg"></a>

Small but practical [examples](https://github.com/kataras/iris/tree/master/_examples#table-of-contents) --they cover each feature.

Wanna create your own fast URL Shortener Service Using Iris? --click [here](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7) to learn how.

[Godocs](https://godoc.org/github.com/kataras/iris) --for deep understanding.


Support
------------

- [Post](http://support.iris-go.com) a feature request or report a bug, will help to make the framework even better.
- :star: and watch [the project](https://github.com/kataras/iris/stargazers), will notify you about updates.
- :earth_americas: publish [an article](https://medium.com/) or share a [tweet](https://twitter.com/) about Iris.
- Donations, will help me to continue.

I'll be glad to talk with you about **your awesome feature requests**, 
open a new [discussion](http://support.iris-go.com), you will be heard!

Thanks in advance!

Buy me a cup of coffee?
------------

Iris is free and open source but developing it has taken thousands of hours of my time and a large part of my sanity. If you feel this web framework useful to you, it would go a great way to ensuring that I can afford to take the time to continue to develop it.


I spend all my time in the construction of Iris, therefore I have no income value.

Feel free to send **any** amount through paypal:

[![](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=kataras2006%40hotmail%2ecom&lc=GR&item_name=Iris%20web%20framework&item_number=iriswebframeworkdonationid2016&currency_code=EUR&bn=PP%2dDonationsBF%3abtn_donateCC_LG%2egif%3aNonHosted)

> Please check your e-mail after your donation.

Thanks for your gratitude and finance help ♡
<!-- 

Some of the benefits are listed here:

- Your github username, after your approval, is visible at the top of the README page.
- Access to the 'donors' [private chat room](https://kataras.rocket.chat/group/donors) gives you real-time assistance by Iris' Author.

--> 

<!-- 

### Become An Iris Sponsor

Want to add your company's logo to our [website](http://iris-go.com)?

Please contact me via email: kataras2006@hotmail.com

Thank you!

-->

Third Party Middleware
------------

Iris has its own middleware form of `func(ctx context.Context)` but it's also compatible with all `net/http` middleware forms. See [here](https://github.com/kataras/iris/blob/master/_examples/beginner/convert-handlers/negroni-like/main.go).

I'm sure that each of you have, already, found his own favorite list but here's a small list of third-party handlers:

| Middleware | Author | Description |
| -----------|--------|-------------|
| [binding](https://github.com/mholt/binding) | [Matt Holt](https://github.com/mholt) | Data binding from HTTP requests into structs |
| [cloudwatch](https://github.com/cvillecsteele/negroni-cloudwatch) | [Colin Steele](https://github.com/cvillecsteele) | AWS cloudwatch metrics middleware |
| [csp](https://github.com/awakenetworks/csp) | [Awake Networks](https://github.com/awakenetworks) | [Content Security Policy](https://www.w3.org/TR/CSP2/) (CSP) support |
| [delay](https://github.com/jeffbmartinez/delay) | [Jeff Martinez](https://github.com/jeffbmartinez) | Add delays/latency to endpoints. Useful when testing effects of high latency |
| [New Relic Go Agent](https://github.com/yadvendar/negroni-newrelic-go-agent) | [Yadvendar Champawat](https://github.com/yadvendar) | Official [New Relic Go Agent](https://github.com/newrelic/go-agent) (currently in beta)  |
| [gorelic](https://github.com/jingweno/negroni-gorelic) | [Jingwen Owen Ou](https://github.com/jingweno) | New Relic agent for Go runtime |
| [JWT Middleware](https://github.com/auth0/go-jwt-middleware) | [Auth0](https://github.com/auth0) | Middleware checks for a JWT on the `Authorization` header on incoming requests and decodes it|
| [logrus](https://github.com/meatballhat/negroni-logrus) | [Dan Buch](https://github.com/meatballhat) | Logrus-based logger |
| [onthefly](https://github.com/xyproto/onthefly) | [Alexander Rødseth](https://github.com/xyproto) | Generate TinySVG, HTML and CSS on the fly |
| [permissions2](https://github.com/xyproto/permissions2) | [Alexander Rødseth](https://github.com/xyproto) | Cookies, users and permissions |
| [prometheus](https://github.com/zbindenren/negroni-prometheus) | [Rene Zbinden](https://github.com/zbindenren) | Easily create metrics endpoint for the [prometheus](http://prometheus.io) instrumentation tool |
| [render](https://github.com/unrolled/render) | [Cory Jacobsen](https://github.com/unrolled) | Render JSON, XML and HTML templates |
| [RestGate](https://github.com/pjebs/restgate) | [Prasanga Siripala](https://github.com/pjebs) | Secure authentication for REST API endpoints |
| [secure](https://github.com/unrolled/secure) | [Cory Jacobsen](https://github.com/unrolled) | Middleware that implements a few quick security wins |
| [stats](https://github.com/thoas/stats) | [Florent Messa](https://github.com/thoas) | Store information about your web application (response time, etc.) |
| [VanGoH](https://github.com/auroratechnologies/vangoh) | [Taylor Wrobel](https://github.com/twrobel3) | Configurable [AWS-Style](http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html) HMAC authentication middleware |
| [xrequestid](https://github.com/pilu/xrequestid) | [Andrea Franz](https://github.com/pilu) | Middleware that assigns a random X-Request-Id header to each request |
| [digits](https://github.com/bamarni/digits) | [Bilal Amarni](https://github.com/bamarni) | Middleware that handles [Twitter Digits](https://get.digits.com/) authentication |

Feel free to put up a [PR](https://github.com/iris-contrib/middleware) your middleware!

Testing
------------

The `httptest` package is your way for end-to-end HTTP testing, it uses the httpexpect library created by our friend, [gavv](https://github.com/gavv).

A simple test is located to [./_examples/intermediate/httptest/main_test.go](https://github.com/kataras/iris/blob/master/_examples/intermediate/httptest/main_test.go)

Philosophy
------------

The Iris philosophy is to provide robust tooling for HTTP, making it a great solution for single page applications, web sites, hybrids, or public HTTP APIs. Keep note that, today, iris is faster than apache+nginx itself.

Iris does not force you to use any specific ORM or template engine. With support for the most popular template engines, you can quickly craft your perfect application.


People
------------

The author of Iris is [@kataras](https://github.com/kataras).

However the real Success of Iris belongs to you with your bug reports and feature requests that made this Framework so Unique.

Contact
------------

Besides the fact that we have a [community chat][Chat] for questions or reports and ideas, [stackoverflow](http://stackoverflow.com/) section for generic go+iris questions and the [iris support](http://support.iris-go.com) for bug reports and feature requests, you can also contact with me, as a person who is always open to help you:

- [Twitter](https://twitter.com/MakisMaropoulos)
- [Facebook](https://facebook.com/kataras.gopher)
- [Linkedin](https://www.linkedin.com/in/gerasimos-maropoulos)

Version
------------

Current: **7.0.5**

Each new release is pushed to the master. It stays there until the next version. When a next version is released then the previous version goes to its own branch with `gopkg.in` as its import path (and its own vendor folder), in order to keep it working "for-ever".

Community members can request additional features or report a bug fix for a specific iris version. 


### Should I upgrade my Iris?

Developers are not forced to use the latest Iris version, they can use any version in production, they can update at any time they want.

Testers should upgrade immediately, if you're willing to use Iris in production you can wait a little more longer, transaction should be as safe as possible.

### Where can I find older versions?

Each Iris version is independent. Only bug fixes, Router's API and experience are kept.

Previous versions can be found at [releases page](https://github.com/kataras/iris/releases).

License
------------

Unless otherwise noted, the source files are distributed
under the BSD-3-Clause License found in the [LICENSE file](LICENSE).

Note that some third-party packages that you use with Iris may requires
different license agreements.

[Chat]: https://kataras.rocket.chat/channel/iris
