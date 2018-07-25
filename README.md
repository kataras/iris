# Iris Web Framework <a href="README_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg?v=10" /></a> <a href="README_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> <a href="README_PT_BR.md"><img width="20px" src="https://iris-go.com/images/flag-pt-br.svg?v=10" /></a> <a href="README_JPN.md"><img width="20px" src="https://iris-go.com/images/flag-japan.svg?v=10" /></a>

<a href="https://iris-go.com"> <img align="right" width="169px" src="https://iris-go.com/images/icon.svg?v=a" title="logo created by @merry.dii" /> </a>

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)<!-- [![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)--> [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris) [![vscode-iris](https://img.shields.io/badge/ext%20-vscode-0c77e3.svg?style=flat-square)](https://marketplace.visualstudio.com/items?itemName=kataras2006.iris)<!--[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)--> [![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris) [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://iris-go.com/v10/recipe) [![release](https://img.shields.io/badge/release%20-v10.6-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/releases)

Iris is a fast, simple yet fully featured and very efficient web framework for Go.

Iris provides a beautifully expressive and easy to use foundation for your next website or API.

Finally, a real expressjs equivalent for the Go Programming Language.

Learn what [others say about Iris](#support) and [star](https://github.com/kataras/iris/stargazers) this github repository to stay [up to date](https://facebook.com/iris.framework).

## Installation

The only requirement is the [Go Programming Language](https://golang.org/dl/)

```sh
$ go get -u github.com/kataras/iris
```

Iris takes advantage of the [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) feature. You get truly reproducible builds, as this method guards against upstream renames and deletes.

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new-gray.png)](_benchmarks/README_UNIX.md)

_Updated at: [Tuesday, 21 November 2017](_benchmarks/README_UNIX.md)_

<details>
<summary>Benchmarks from third-party source over the rest web frameworks</summary>

![Comparison with other frameworks](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

</details>

## Philosophy

The Iris philosophy is to provide robust tooling for HTTP, making it a great solution for single page applications, web sites, hybrids, or public HTTP APIs. Keep note that, so far, iris is the fastest web framework ever created in terms of performance.

Iris does not force you to use any specific ORM or template engine. With support for the most used template engines, you can quickly craft the perfect application.

## Quick start
 
```sh
$ cat example.go
```

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.Default()
    app.Get("/ping", func(ctx iris.Context) {
        ctx.JSON(iris.Map{
            "message": "pong",
        })
    })

    // Listen and serve on http://localhost:8080.
    app.Run(iris.Addr(":8080"))
}
```

```
$ go run example.go
Now listening on: http://localhost:8080
Application Started. Press CTRL+C to shut down.
_
```

### Using Get, Post, Put, Patch, Delete and Options

```go
func main() {
    // Creates an application with default middleware:
    // logger and recovery (crash-free) middleware.
    app := iris.Default()

    app.Get("/someGet", getting)
    app.Post("/somePost", posting)
    app.Put("/somePut", putting)
    app.Delete("/someDelete", deleting)
    app.Patch("/somePatch", patching)
    app.Head("/someHead", head)
    app.Options("/someOptions", options)

    app.Run(iris.Addr(":8080"))
}
```

### Parameters in path

```go
func main() {
    app := iris.Default()

    // This handler will match /user/kataras but will not match neither /user/ or /user.
    app.Get("/user/{name}", func(ctx iris.Context) {
        name := ctx.Params().Get("name")
        ctx.Writef("Hello %s", name)
    })

    // This handles the /user/kataras/42
    // and fires 400 bad request if /user/kataras/string.
    // The "else 400" is optionally:
    // by-default it will fire 404 not found if alphanumeric instead
    // of number passed on the "age" parameter.
    app.Get("/user/{name:string}/{age:int else 400}", func(ctx iris.Context) {
        name := ctx.Params().Get("name")
        age, _ := ctx.Params().GetInt("age")
        ctx.Writef("%s is %d years old", name, age)
    })

    // However, this one will match /action/{user}/star and also /action/{user}/stars
    // or even /action/{user}/likes/page/2.
    // It should match anything after the /action/{user}/
    // except the /action/{user}/static which is handled by the below route.
    app.Get("/action/{user:string}/{action:path}", func(ctx iris.Context) {
        user := ctx.Params().Get("user")
        action := ctx.Params().Get("action")
        ctx.Writef("user: %s | action: %s", user, action)
    })

    // Unlike other frameworks and routers,
    // Iris is smart enough to understand that this is not the previous,
    // wildcard of type path route, it should only match the /action/{user}/static.
    app.Get("/action/{user:string}/static", func(ctx iris.Context) {
        user := ctx.Params().Get("user")
        ctx.Writef("static path for user: %s", user)
    })


    // http://localhost:8080/user/kataras
    // http://localhost:8080/user/kataras/25
    // http://localhost:8080/action/kataras/upgrade
    // http://localhost:8080/action/kataras/static
    app.Run(iris.Addr(":8080"))
}
```

> If parameter type is missing then the defaults to `string`, therefore `{name:string}` and `{name}` do the same exactly

> Learn more about path parameter's types by navigating [here](_examples/routing/dynamic-path/main.go#L31)

### Cookies

```sh
$ cat _examples/cookies/basic/main.go
```

```go
package main

import "github.com/kataras/iris"

func newApp() *iris.Application {
    app := iris.New()

    // Set A Cookie.
    app.Get("/cookies/{name}/{value}", func(ctx iris.Context) {
        name := ctx.Params().Get("name")
        value := ctx.Params().Get("value")

        ctx.SetCookieKV(name, value)

        ctx.Writef("cookie added: %s = %s", name, value)
    })

    // Retrieve A Cookie.
    app.Get("/cookies/{name}", func(ctx iris.Context) {
        name := ctx.Params().Get("name")

        value := ctx.GetCookie(name)

        ctx.WriteString(value)
    })

    // Delete A Cookie.
    app.Delete("/cookies/{name}", func(ctx iris.Context) {
        name := ctx.Params().Get("name")

        ctx.RemoveCookie(name)

        ctx.Writef("cookie %s removed", name)
    })

    return app
}

func main() {
    app := newApp()

    // GET:    http://localhost:8080/cookies/my_name/my_value
    // GET:    http://localhost:8080/cookies/my_name
    // DELETE: http://localhost:8080/cookies/my_name
    app.Run(iris.Addr(":8080"))
}
```

* Alternatively, use a regular `http.Cookie`: `ctx.SetCookie(&http.Cookie{...})`
* If you want to set custom the path: `ctx.SetCookieKV(name, value, iris.CookiePath("/custom/path/cookie/will/be/stored"))`.
* If you want to be available only to the current request path: `ctx.SetCookieKV(name, value, iris.CookieCleanPath /* or iris.CookiePath("") */)`
    * `iris.CookieExpires(time.Duration)`
    * `iris.CookieHTTPOnly(false)`
* `ctx.Request().Cookie(name)` is also available, it's the `net/http` approach
* Learn more about path parameter's types by clicking [here](_examples/routing/dynamic-path/main.go#L31).

### Testing

```go
package main

import (
    "fmt"
    "testing"

    "github.com/kataras/iris/httptest"
)

// go test -v -run=TestCookiesBasic$
func TestCookiesBasic(t *testing.T) {
    app := newApp()
    e := httptest.New(t, app, httptest.URL("http://example.com"))

    cookieName, cookieValue := "my_cookie_name", "my_cookie_value"

    // Test Set A Cookie.
    t1 := e.GET(fmt.Sprintf("/cookies/%s/%s", cookieName, cookieValue)).Expect().Status(httptest.StatusOK)
    t1.Cookie(cookieName).Value().Equal(cookieValue) // validate cookie's existence, it should be there now.
    t1.Body().Contains(cookieValue)

    path := fmt.Sprintf("/cookies/%s", cookieName)

    // Test Retrieve A Cookie.
    t2 := e.GET(path).Expect().Status(httptest.StatusOK)
    t2.Body().Equal(cookieValue)

    // Test Remove A Cookie.
    t3 := e.DELETE(path).Expect().Status(httptest.StatusOK)
    t3.Body().Contains(cookieName)

    t4 := e.GET(path).Expect().Status(httptest.StatusOK)
    t4.Cookies().Empty()
    t4.Body().Empty()
}
```

## Learn

First of all, the most correct way to begin with a web framework is to learn the basics of the programming language and the standard `http` capabilities, if your web application is a very simple personal project without performance and maintainability requirements you may want to proceed just with the standard packages. After that follow the guidelines:

- Navigate through **100+1** **[examples](_examples)** and some [iris starter kits](#iris-starter-kits) we crafted for you
- Read the [godocs](https://godoc.org/github.com/kataras/iris) for any details
- Prepare a cup of coffee or tea, whatever pleases you the most, and read some [articles](#articles) we found for you

### Iris starter kits

<!-- table form
| Description | Link |
| -----------|-------------|
| Hasura hub starter project with a ready to deploy golang helloworld webapp with IRIS! | https://hasura.io/hub/project/hasura/hello-golang-iris |
| A basic web app built in Iris for Go |https://github.com/gauravtiwari/go_iris_app |
| A mini social-network created with the awesome Iris💖💖 | https://github.com/iris-contrib/Iris-Mini-Social-Network |
| Iris isomorphic react/hot reloadable/redux/css-modules starter kit | https://github.com/iris-contrib/iris-starter-kit |
| Demo project with react using typescript and Iris | https://github.com/ionutvilie/react-ts |
| Self-hosted Localization Management Platform built with Iris and Angular | https://github.com/iris-contrib/parrot |
| Iris + Docker and Kubernetes | https://github.com/iris-contrib/cloud-native-go |
| Quickstart for Iris with Nanobox | https://guides.nanobox.io/golang/iris/from-scratch |
-->

1. [A basic web app built in Iris for Go](https://github.com/gauravtiwari/go_iris_app)
2. [A mini social-network created with the awesome Iris💖💖](https://github.com/iris-contrib/Iris-Mini-Social-Network)
3. [Iris isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/iris-contrib/iris-starter-kit)
4. [Demo project with react using typescript and Iris](https://github.com/ionutvilie/react-ts)
5. [Self-hosted Localization Management Platform built with Iris and Angular](https://github.com/iris-contrib/parrot)
6. [Iris + Docker and Kubernetes](https://github.com/iris-contrib/cloud-native-go)
7. [Quickstart for Iris with Nanobox](https://guides.nanobox.io/golang/iris/from-scratch)
8. [A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](https://hasura.io/hub/project/hasura/hello-golang-iris)

> Did you build something similar? Let us [know](https://github.com/kataras/iris/pulls)!

### Middleware

Iris has a great collection of handlers[[1]](middleware/)[[2]](https://github.com/iris-contrib/middleware) that you can use side by side with your web apps. However you are not limited to them - you are free to use any third-party middleware that is compatible with the [net/http](https://golang.org/pkg/net/http/) package, [_examples/convert-handlers](_examples/convert-handlers) will show you the way.

Iris, unlike others, is 100% compatible with the standards and that's why the majority of the big companies that adapt Go to their workflow, like a very famous US Television Network, trust Iris; it's up-to-date and it will be always aligned with the std `net/http` package which is modernized by the Go Authors on each new release of the Go Programming Language.

### Articles

* [A Todo MVC Application using Iris and Vue.js](https://hackernoon.com/a-todo-mvc-application-using-iris-and-vue-js-5019ff870064)
* [A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](https://bit.ly/2lmKaAZ)
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

### Video Courses

| Name | Producer |
| -----------|-------------|
| [Daily Coding - Web Framework Golang: Iris Framework](https://www.youtube.com/watch?v=BmOLFQ29J3s) | [WarnabiruTV](https://www.youtube.com/user/panahbiru) |
| [Playlist: Tutorial Golang MVC Iris Framework](https://www.youtube.com/watch?v=uXiNYhJqh2I&list=PLMrwI6jIZn-1tzskocnh1pptKhVmWdcbS) | [Musobar Media](https://www.youtube.com/channel/UCqOBKU-JXrM86FTt7Xzwdxw) |
| [Go/Golang 27 - Iris framework : Routage de base](https://www.youtube.com/watch?v=rQxRoN6ub78) | [stephgdesign](https://www.youtube.com/user/stephgdesign) |
| [Go/Golang 28 - Iris framework : Templating](https://www.youtube.com/watch?v=nOKYV073S2Y) | [stephgdesign](https://www.youtube.com/user/stephgdesign) |
| [Go/Golang 29 - Iris framework : Paramètres](https://www.youtube.com/watch?v=K2FsprfXs1E) | [stephgdesign](https://www.youtube.com/user/stephgdesign) |
| [Go/Golang 30 - Iris framework : Les middelwares](https://www.youtube.com/watch?v=BLPy1So6bhE) | [stephgdesign](https://www.youtube.com/user/stephgdesign) |
| [Go/Golang 31 - Iris framework : Les sessions](https://www.youtube.com/watch?v=RnBwUrwgEZ8) | [stephgdesign](https://www.youtube.com/user/stephgdesign) |

## Support

- [HISTORY](HISTORY.md#tu-05-june-2018--v1066) file is your best friend, it contains information about the latest features and changes
- Did you happen to find a bug? Post it at [github issues](https://github.com/kataras/iris/issues)
- Do you have any questions or need to speak with someone experienced to solve a problem at real-time? Join us to the [community chat](https://chat.iris-go.com)
- Complete our form-based user experience report by clicking [here](https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link)
- Do you like the framework? Tweet something about it! The People have spoken:

<a href="https://twitter.com/gelnior/status/769100480706379776"> 
    <img src="https://comments.iris-go.com/comment27_mini.png" width="350px">
</a>

<a href="https://twitter.com/MeAlex07/status/822799954188075008"> 
    <img src="https://comments.iris-go.com/comment28_mini.png" width="350px">
</a>

<a href="https://twitter.com/_mgale/status/818591490305761280"> 
    <img src="https://comments.iris-go.com/comment29_mini.png" width="350px">
</a>
<a href="https://twitter.com/VeayoX/status/813273328550973440"> 
    <img src="https://comments.iris-go.com/comment30_mini.png" width="350px">
</a>

<a href="https://twitter.com/pvsukale/status/745328224876408832"> 
    <img src="https://comments.iris-go.com/comment31_mini.png" width="350px">
</a>

<a href="https://twitter.com/blainsmith/status/745338092211560453"> 
    <img src="https://comments.iris-go.com/comment32_mini.png" width="350px">
</a>

<a href="https://twitter.com/tjbyte/status/758287014210867200"> 
    <img src="https://comments.iris-go.com/comment33_mini.png" width="350px">
</a>

<a href="https://twitter.com/tangzero/status/751050577220698112"> 
    <img src="https://comments.iris-go.com/comment34_mini.png" width="350px">
</a>

<a href="https://twitter.com/tjbyte/status/758287244947972096"> 
    <img src="https://comments.iris-go.com/comment33_2_mini.png" width="350px">
</a>

<a href="https://twitter.com/ferarias/status/902468752364773376"> 
    <img src="https://comments.iris-go.com/comment41.png" width="350px">
</a>

<br/><br/>

For more information about contributing to the Iris project please check the [CONTRIBUTING.md](CONTRIBUTING.md) file.

[List of all Contributors](https://github.com/kataras/iris/graphs/contributors)

### Get hired

There are many companies and start-ups looking for Go web developers with Iris experience as requirement, we are searching for you every day and we post those information via our [facebook page](https://www.facebook.com/iris.framework), like the page to get notified, we have already posted some of them.

### Backers

Thank you to all our backers! 🙏 [Become a backer](https://iris-go.com/donate)

<a href="https://iris-go.com/donate" target="_blank"><img src="https://iris-go.com/backers.svg?v=2"/></a>

## License

Iris is licensed under the [3-Clause BSD License](LICENSE). Iris is 100% free and open-source software.

For any questions regarding the license please send [e-mail](mailto:kataras2006@hotmail.com?subject=Iris%20License).
