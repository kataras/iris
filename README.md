# News

![](https://iris-go.com/images/release.png) Iris version **12.1.8** has been [released](HISTORY.md#su-16-february-2020--v1218)!

![](https://iris-go.com/images/cli.png) The official [Iris Command Line Interface](https://github.com/kataras/iris-cli) will soon be near you in 2020!

![](https://iris-go.com/images/sponsor.png) Support your favorite web framework through [Github Sponsors Program](https://github.com/sponsors/kataras)!

# Iris Web Framework <a href="README_ZH.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a>

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=for-the-badge&logo=travis)](https://travis-ci.org/kataras/iris) [![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)<!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://godoc.org/github.com/kataras/iris)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0C8EC5.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/master/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=7E18DD&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community)<!--[![donate on PayPal](https://img.shields.io/badge/support-PayPal-blue.svg?style=for-the-badge)](https://www.paypal.me/kataras)--><!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->

<a href="https://iris-go.com"> <img align="right" src="https://iris-go.com/images/logo-w169.png"></a>

Iris is a fast, simple yet fully featured and very efficient web framework for Go.

It provides a beautifully expressive and easy to use foundation for your next website or API.

Learn what [others saying about Iris](https://iris-go.com/testimonials/) and **[star](https://github.com/kataras/iris/stargazers)** this open-source project to support its potentials.

[![](https://media.giphy.com/media/j5WLmtvwn98VPrm7li/giphy.gif)](https://iris-go.com/testimonials/)

## 📖 Learning Iris

<details>
<summary>Quick start</summary>

```sh
# https://github.com/kataras/iris/wiki/Installation
$ go get github.com/kataras/iris/v12@latest
# assume the following code in example.go file
$ cat example.go
```

```go
package main

import "github.com/kataras/iris/v12"

func main() {
    app := iris.New()
    app.Get("/ping", func(ctx iris.Context) {
        ctx.JSON(iris.Map{
            "message": "pong",
        })
    })

    app.Listen(":8080")
}
```

```sh
# run example.go and
# visit http://localhost:8080/ping on browser
$ go run example.go
```

> Routing is powered by [muxie](https://github.com/kataras/muxie), the most powerful and fastest trie-based software written in Go.

</details>

Iris contains extensive and thorough **[wiki](https://github.com/kataras/iris/wiki)** making it easy to get started with the framework.

<!-- ![](https://media.giphy.com/media/Ur8iqy9FQfmPuyQpgy/giphy.gif) -->

For a more detailed technical documentation you can head over to our [godocs](https://godoc.org/github.com/kataras/iris). And for executable code you can always visit the [\_examples](_examples/) repository's subdirectory.

### Do you like to read while traveling?

<a href="https://bit.ly/iris-req-book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg" width="200" /> </a>

<!-- [![follow author](https://img.shields.io/twitter/follow/makismaropoulos.svg?style=for-the-badge)](https://twitter.com/intent/follow?screen_name=makismaropoulos) -->

You can [request](https://bit.ly/iris-req-book) a PDF version and online access of the **E-Book** today and be participated in the development of Iris.

## 🙌 Contributing

We'd love to see your contribution to the Iris Web Framework! For more information about contributing to the Iris project please check the [CONTRIBUTING.md](CONTRIBUTING.md) file.

[List of all Contributors](https://github.com/kataras/iris/graphs/contributors)

## 🛡 Security Vulnerabilities

If you discover a security vulnerability within Iris, please send an e-mail to [iris-go@outlook.com](mailto:iris-go@outlook.com). All security vulnerabilities will be promptly addressed.

## 📝 License

This project is licensed under the [BSD 3-clause license](LICENSE), just like the Go project itself.

The project name "Iris" was inspired by the Greek mythology.
<!-- ## Stargazers over time

[![Stargazers over time](https://starchart.cc/kataras/iris.svg)](https://starchart.cc/kataras/iris) -->
