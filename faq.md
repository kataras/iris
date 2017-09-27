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
| -----------|--------|--------|--------|
| `undefinied iris.Context` | caused of using the **optional type alias** `iris.Context` instead of the `context.Context` when building with Go 1.8 | import the original package `github.com/kataras/iris/context` and declare as `func(context.Context){})` **or** download and install the [latest go version](https://golang.org/dl) |

Type alias is a new feature, introduced at Go version 1.9, so if you want to use Iris' type aliases you have to build using the latest Go version. Nothing really changes for your application if you use type alias or not, Iris' type aliases helps you to omit import statements -- to reduce lines of code, nothing more.

> README.md has a section which helps you understand more about this new feature, [read it here](https://github.com/kataras/iris#-type-aliases).

## Active development mode

Iris may have reached version 8, but we're not stopping there. We have many feature ideas on our board that we're anxious to add and other innovative web development solutions that we're planning to build into Iris.

## Can I found a job if I learn how to use Iris?

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

Help this project to continue deliver awesome and unique features with the higher code quality as possible by donating any amount.

[![](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=kataras2006%40hotmail%2ecom&lc=GR&item_name=Iris%20web%20framework&item_number=iriswebframeworkdonationid2016&currency_code=EUR&bn=PP%2dDonationsBF%3abtn_donateCC_LG%2egif%3aNonHosted)