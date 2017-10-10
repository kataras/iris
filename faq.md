# FAQ

## How to upgrade

```sh
go get -u github.com/kataras/iris
```

## Learning

More than 50 practical examples, tutorials and articles at:

- https://github.com/kataras/iris/tree/master/_examples
- https://github.com/iris-contrib/examples
- https://iris-go.com/v8/recipe
- https://docs.iris-go.com (in-progress)
- https://godoc.org/github.com/kataras/iris

> [Stay tuned](https://github.com/kataras/iris/stargazers), community prepares even more tutorials.

Want to help and join to the greatest community? Describe your skills and push your own sections at: https://github.com/kataras/build-a-better-web-together/issues/new

### common errors that new gophers may meet

#### type aliases

| build error | reason | solution |
| -----------|--------|--------|
| `undefined iris.Context` | caused of using the **optional type alias** `iris.Context` instead of the `context.Context` when building with Go 1.8 | import the original package `github.com/kataras/iris/context` and declare as `func(context.Context){})` **or** download and install the [latest go version](https://golang.org/dl) _recommended_ |

Type alias is a new feature, introduced at Go version 1.9, so if you want to use Iris' type aliases you have to build using the latest Go version. Nothing really changes for your application if you use type alias or not, Iris' type aliases helps you to omit import statements -- to reduce lines of code, nothing more.

**Details...**

Go version 1.9 introduced the [type alias](https://golang.org/doc/go1.9#language) feature.

Iris uses the `type alias` feature to help you writing less code by omitting some package imports. The examples and documentation are written using Go 1.9 as well.

If you build your Go app with Go 1.9 you can, optionally, use all Iris web framework's features by importing one single package, the `github.com/kataras/iris`.

Available type aliases;

| Go 1.8 | Go 1.8 usage | Go 1.9 usage (optionally) |
| -----------|--------|--------|
| `import "github.com/kataras/iris/context"` | `func(context.Context) {}`, `context.Handler`, `context.Map` |  `func(iris.Context) {}`, `iris.Handler`,  `iris.Map` |
| `import "github.com/kataras/iris/mvc"` | `type MyController struct { mvc.Controller }` , `mvc.SessionController` | `type MyController struct { iris.Controller }`, `iris.SessionController` |
| `import "github.com/kataras/iris/core/router"` | `app.PartyFunc("/users", func(p router.Party) {})` |  `app.PartyFunc("/users", func(p iris.Party) {})` |
| `import "github.com/kataras/iris/core/host"` | `app.ConfigureHost(func(s *host.Supervisor) {})` | `app.ConfigureHost(func(s *iris.Supervisor) {})` |

You can find all type aliases and their original package import statements at the [./context.go file](context.go).

> Remember; this doesn't mean that you have to use those type aliases, you can still import the original packages as you did with Go version 1.8, it's up to you.

## Active development mode

Iris may have reached version 8, but we're not stopping there. We have many feature ideas on our board that we're anxious to add and other innovative web development solutions that we're planning to build into Iris.

## Can I find a job if I learn how to use Iris?

Yes, not only because you will learn Golang in the same time, but there are some positions
open for Iris-specific developers the time we speak.

- https://glints.id/opportunities/jobs/5553

## Can Iris be used in production after Dubai purchase?

Yes, now more than ever.

https://github.com/kataras/iris/issues/711

## Do we have a community Chat?

Yes, https://kataras.rocket.chat/channel/iris.

https://github.com/kataras/iris/issues/646

## How this open-source project still active and shine?

By normal people like you, who help us by donating small or larger amounts of money.

Help this project to continue deliver awesome and unique features with the higher code quality as possible by donating any amount via [PayPal](https://www.paypal.me/kataras)!