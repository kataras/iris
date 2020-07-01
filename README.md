# News

> This is the under-development branch. Stay tuned for the upcoming release [v12.2.0](HISTORY.md#Next).

<!-- ![](https://iris-go.com/images/release.png) Iris version **12.1.8** has been [released](HISTORY.md#su-16-february-2020--v1218)! -->

![](https://iris-go.com/images/cli.png) The official [Iris Command Line Interface](https://github.com/kataras/iris-cli) will soon be near you in 2020!

![](https://iris-go.com/images/sponsor.png) Support your favorite web framework through [Github Sponsors Program](https://github.com/sponsors/kataras)!

# Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /></a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a>

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=for-the-badge&logo=travis)](https://travis-ci.org/kataras/iris) [![view examples](https://img.shields.io/badge/examples%20-173-a83adf.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/master/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) <!--[![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)--> [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate) <!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.0)--> <!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->

<a href="https://iris-go.com"> <img align="right" src="https://iris-go.com/images/logo-w169.png"></a>

Iris is a fast, simple yet fully featured and very efficient web framework for Go.

It provides a beautifully expressive and easy to use foundation for your next website or API.

Learn what [others saying about Iris](https://iris-go.com/testimonials/) and **[star](https://github.com/kataras/iris/stargazers)** this open-source project to support its potentials.

[![](https://media.giphy.com/media/j5WLmtvwn98VPrm7li/giphy.gif)](https://iris-go.com/testimonials/)

[![Benchmarks: Apr 2, 2020 at 12:13pm (UTC)](https://iris-go.com/images/benchmarks.svg)](https://github.com/kataras/server-benchmarks)

## 📖 Learning Iris

<details>
<summary>Quick start</summary>

```sh
# https://github.com/kataras/iris/wiki/Installation
$ go get github.com/kataras/iris/v12@master
# assume the following code in main.go file
$ cat main.go
```

```go
package main

import "github.com/kataras/iris/v12"

func main() {
  app := iris.New()
  app.Get("/", index)
  app.Listen(":8080")
}

func index(ctx iris.Context) {
  ctx.HTML("<h1>Hello, World!</h1>")
}
```

```sh
$ go run main.go
```

</details>

[![run in the browser](https://img.shields.io/badge/Run-in%20the%20Browser-348798.svg?style=for-the-badge&logo=repl.it)](https://bit.ly/2YJeSZe)

Iris contains extensive and thorough **[wiki](https://github.com/kataras/iris/wiki)** making it easy to get started with the framework.

<!-- ![](https://media.giphy.com/media/Ur8iqy9FQfmPuyQpgy/giphy.gif) -->

For a more detailed technical documentation you can head over to our [godocs](https://godoc.org/github.com/kataras/iris). And for executable code you can always visit the [./_examples](_examples) repository's subdirectory.

### Do you like to read while traveling?

<a href="https://bit.ly/iris-req-book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12" /> </a>

[![follow author](https://img.shields.io/twitter/follow/makismaropoulos?color=ee7506&logoColor=ee7506&style=for-the-badge)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

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
