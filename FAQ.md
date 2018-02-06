# FAQ

## [![iris](https://img.shields.io/badge/iris-powered-2196f3.svg?style=for-the-badge)](https://github.com/kataras/iris)

Add a `badge` to your open-source projects powered by [Iris](https://iris-go.com) by pasting the below code snippet to the project repo's README.md:

```md
[![iris](https://img.shields.io/badge/iris-powered-2196f3.svg?style=for-the-badge)](https://github.com/kataras/iris)
```

> The badge is optionally, of course, it is just a simple and fast way to support Iris. The badge is work of a third-party, taken from https://github.com/blob-go/blob-go which was published by our friend @clover113 and we loved it<3

## Editors & IDEs Extensions

### Visual Studio Code <a href="https://marketplace.visualstudio.com/items?itemName=kataras2006.iris"><img src="https://upload.wikimedia.org/wikipedia/commons/thumb/2/2d/Visual_Studio_Code_1.18_icon.svg/2000px-Visual_Studio_Code_1.18_icon.svg.png" height="20px" width="20px" /></a>

<https://marketplace.visualstudio.com/items?itemName=kataras2006.iris>

> Please feel free to list your own Iris extension(s) here by [PR](https://github.com/kataras/iris/pulls)

## How to upgrade

```sh
go get -u github.com/kataras/iris
```

## Learning

More than 100 practical examples, tutorials and articles at:

- https://github.com/kataras/iris/tree/master/_examples
- https://github.com/iris-contrib/examples
- https://iris-go.com/v10/recipe
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

Iris may have reached version 10, but we're not stopping there. We have many feature ideas on our board that we're anxious to add and other innovative web development solutions that we're planning to build into Iris.

## Can I find a job if I learn how to use Iris?

Yes, not only because you will learn Golang in the same time, but there are some positions
open for Iris-specific developers the time we speak.

Go to our facebook page, like it and receive notifications about new job offers, we already have couple of them stay at the top of the page: https://www.facebook.com/iris.framework

<!--
## Can Iris be used in production after Dubai purchase?

Yes, now more than ever.

https://github.com/kataras/iris/issues/711

-------

UPDATE which I could mention by the beginning of the Decemember of 2017:

Nothing keeps for ever, and we should move on to greater things.

As you probably know, I was hired to develop an inside Iris version for a Dubai-based startup company's specific requirements in the same time I was developing the open-source Iris repository with your help this time as well!

As our first deal was to end this agreement via last-time negotiatations by the end of the current year (2017), the
agreement ended unofficially at 22 Novemember of 2017 (officially some weeks later, paper work), and after a week I came back to Greece as you may understood from the regularly commits and improvements to the public repository that I pushed.
 -->

## Do we have a community Chat?

Yes, https://chat.iris-go.com

https://github.com/kataras/iris/issues/646

## How this open-source project still active and shine?

By normal people like you, who help us by donating small or larger amounts of money.

Help this project to continue deliver awesome and unique features with the highest possible code quality as possible by donating any amount via [PayPal](https://www.paypal.me/kataras). Your name will be published [here](https://iris-go.com/donate) after your approval via e-mail.