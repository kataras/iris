# Iris Web Framework

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=for-the-badge&logo=travis)](https://travis-ci.org/kataras/iris) [![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)<!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://godoc.org/github.com/kataras/iris)--> [![view examples](https://img.shields.io/badge/learn%20by-examples-0C8EC5.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/master/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=7E18DD&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community)<!--[![donate on PayPal](https://img.shields.io/badge/support-PayPal-blue.svg?style=for-the-badge)](https://www.paypal.me/kataras)--><!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->

Iris 是基于 Go 编写的一个快速，简单但功能齐全且非常高效的 Web 框架。 它为您的下一个网站或 API 提供了一个非常富有表现力且易于使用的基础。

看看 [其他人如何评价 Iris](https://iris-go.com/testimonials/)，同时欢迎各位点亮 **star**。

[![](https://media.giphy.com/media/j5WLmtvwn98VPrm7li/giphy.gif)](https://iris-go.com/testimonials/)

## 学习 Iris

<details>
<summary>快速入门</summary>

```sh
# 假设文件已经存在
$ cat example.go
```

```go
package main

import "github.com/kataras/iris/v12"

func main() {
    app := iris.Default()
    app.Get("/ping", func(ctx iris.Context) {
        ctx.JSON(iris.Map{
            "message": "pong",
        })
    })

    app.Listen(":8080")
}
```

```sh
# 运行 example.go
# 在浏览器中访问 http://localhost:8080/ping
$ go run example.go
```

> 路由由 [muxie](https://github.com/kataras/muxie) 提供支持，muxie 是基于 Go 编写的最强大最快速的基于 trie 的路由

</details>

Iris 包含详细而完整的 **[文档](https://github.com/kataras/iris/wiki)**，使你很容易开始使用该框架。

要了解更多详细的技术文档，可以访问我们的 [godocs](https://godoc.org/github.com/kataras/iris)。对于可执行代码，可以随时访问示例代码，在仓库的 [\_examples](_examples/) 目录下。

### 你喜欢在旅行中看书吗？

你现在可以 [获取](https://bit.ly/iris-req-book) PDF 版本和在线访问我们的 **电子书** 并参与 Iris 的开发。

<a href="https://bit.ly/iris-req-book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12" /> </a>

[![follow author](https://img.shields.io/twitter/follow/makismaropoulos.svg?style=for-the-badge)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

## 贡献

我们很高兴看到你对 Iris Web 框架的贡献！有关为 Iris 做出贡献的更多信息，请查看 [CONTRIBUTING.md](CONTRIBUTING.md)。

[所有贡献者名单](https://github.com/kataras/iris/graphs/contributors)

## 安全漏洞

如果你发现在 Iris 存在安全漏洞，请发送电子邮件至 [iris-go@outlook.com](mailto:iris-go@outlook.com)，所有安全漏洞都会被及时解决。

## 授权协议

项目名称 "Iris" 的灵感来自于希腊神话。

Iris Web 框架授权基于 [3-Clause BSD License](LICENSE) 许可的免费开源软件。
