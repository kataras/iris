[![黑人的命也是命](https://iris-go.com/images/blacklivesmatter_banner.png)](https://support.eji.org/give/153413/#!/donation/checkout)

<!-- # News -->

> 这是一个**开发中的版本**。敬请关注即将发布的版本 [v12.2.0](HISTORY.md#Next)。如果想使用稳定版本，请查看 [v12.1.8 分支](https://github.com/kataras/iris/tree/v12.1.8) 。
>
> ![](https://iris-go.com/images/cli.png) 立即尝试官方的[Iris命令行工具](https://github.com/kataras/iris-cli)！

<!-- ![](https://iris-go.com/images/release.png) Iris version **12.1.8** has been [released](HISTORY.md#su-16-february-2020--v1218)! -->

# Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /></a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![view examples](https://img.shields.io/badge/examples%20-253-a83adf.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/main/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) <!--[![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)--> [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate) <!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.11)--> <!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->

<!-- <a href="https://iris-go.com"> <img align="right" src="https://iris-go.com/images/logo-w169.png"></a> -->

Iris 是基于 Go 编写的一个快速，简单但功能齐全且非常高效的 Web 框架。 

它为您的下一个网站或 API 提供了一个非常富有表现力且易于使用的基础。

看看 [其他人如何评价 Iris](https://iris-go.com/testimonials/)，同时欢迎各位为此开源项目点亮 **[star](https://github.com/kataras/iris/stargazers)**。

[![](https://iris-go.com/images/reviews.gif)](https://iris-go.com/testimonials/)

[![Benchmarks: Jul 18, 2020 at 10:46am (UTC)](https://iris-go.com/images/benchmarks.svg)](https://github.com/kataras/server-benchmarks)

## 📖 开始学习 Iris

```sh
# 安装Iris：https://www.iris-go.com/#ebookDonateForm
$ go get github.com/kataras/iris/v12@latest
# 假设main.go文件中已存在以下代码
$ cat main.go
```

```go
package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()

	booksAPI := app.Party("/books")
	{
		booksAPI.Use(iris.Compression)

		// GET: http://localhost:8080/books
		booksAPI.Get("/", list)
		// POST: http://localhost:8080/books
		booksAPI.Post("/", create)
	}

	app.Listen(":8080")
}

// Book example.
type Book struct {
	Title string `json:"title"`
}

func list(ctx iris.Context) {
	books := []Book{
		{"Mastering Concurrency in Go"},
		{"Go Design Patterns"},
		{"Black Hat Go"},
	}

	ctx.JSON(books)
	// 提示: 在服务器优先级和客户端请求中进行响应协商，
	// 以此来代替 ctx.JSON:
	// ctx.Negotiation().JSON().MsgPack().Protobuf()
	// ctx.Negotiate(books)
}

func create(ctx iris.Context) {
	var b Book
	err := ctx.ReadJSON(&b)
	// 提示: 使用 ctx.ReadBody(&b) 代替，来绑定所有类型的入参
	if err != nil {
		ctx.StopWithProblem(iris.StatusBadRequest, iris.NewProblem().
			Title("Book creation failure").DetailErr(err))
		// 提示: 如果仅有纯文本（plain text）错误响应，
        // 可使用 ctx.StopWithError(code, err) 
		return
	}

	println("Received Book: " + b.Title)

	ctx.StatusCode(iris.StatusCreated)
}
```

同样地，在**MVC**中 :

```go
import "github.com/kataras/iris/v12/mvc"
```

```go
m := mvc.New(booksAPI)
m.Handle(new(BookController))
```

```go
type BookController struct {
	/* dependencies */
}

// GET: http://localhost:8080/books
func (c *BookController) Get() []Book {
	return []Book{
		{"Mastering Concurrency in Go"},
		{"Go Design Patterns"},
		{"Black Hat Go"},
	}
}

// POST: http://localhost:8080/books
func (c *BookController) Post(b Book) int {
	println("Received Book: " + b.Title)

	return iris.StatusCreated
}
```

**启动** 您的 Iris web 服务:

```sh
$ go run main.go
> Now listening on: http://localhost:8080
> Application started. Press CTRL+C to shut down.
```

Books **列表查询** :

```sh
$ curl --header 'Accept-Encoding:gzip' http://localhost:8080/books

[
  {
    "title": "Mastering Concurrency in Go"
  },
  {
    "title": "Go Design Patterns"
  },
  {
    "title": "Black Hat Go"
  }
]
```

**创建** 新的Book:

```sh
$ curl -i -X POST \
--header 'Content-Encoding:gzip' \
--header 'Content-Type:application/json' \
--data "{\"title\":\"Writing An Interpreter In Go\"}" \
http://localhost:8080/books

> HTTP/1.1 201 Created
```

这是**错误**响应所展示的样子：

```sh
$ curl -X POST --data "{\"title\" \"not valid one\"}" \
http://localhost:8080/books

> HTTP/1.1 400 Bad Request

{
  "status": 400,
  "title": "Book creation failure"
  "detail": "invalid character '\"' after object key",
}
```

</details>

[![run in the browser](https://img.shields.io/badge/Run-in%20the%20Browser-348798.svg?style=for-the-badge&logo=repl.it)](https://replit.com/@kataras/Iris-Hello-World-v1220?v=1)

Iris 有完整且详尽的 **[使用文档](https://www.iris-go.com/#ebookDonateForm)** ，让您可以轻松地使用此框架。

<!-- ![](https://media.giphy.com/media/Ur8iqy9FQfmPuyQpgy/giphy.gif) -->

要了解更详细的技术文档，请访问我们的 [godocs](https://pkg.go.dev/github.com/kataras/iris/v12@main)。如果想要寻找代码示例，您可以到仓库的 [./_examples](_examples) 子目录下获取。

### 你喜欢在旅行时阅读吗？

<a href="https://iris-go.com/#book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12" /> </a>

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge)](https://twitter.com/intent/follow?screen_name=iris_framework)

您可以[获取](https://www.iris-go.com/#ebookDonateForm)PDF版本或在线访问**电子图书**，并参与到Iris的开发中。

## 🙌 贡献

我们欢迎您为Iris框架做出贡献！想要知道如何为Iris项目做贡献，请查看[CONTRIBUTING.md](CONTRIBUTING.md)。

[贡献者名单](https://github.com/kataras/iris/graphs/contributors)

## 🛡 安全漏洞

如果您发现在 Iris 存在安全漏洞，请发送电子邮件至 [iris-go@outlook.com](mailto:iris-go@outlook.com)。所有安全漏洞将会得到及时解决。

## 📝 开源协议（License）

就像Go语言的协议一样，此项目也采用 [BSD 3-clause license](LICENSE)。

项目名称 "Iris" 的灵感来自于希腊神话。

<!-- ## Stargazers over time

[![Stargazers over time](https://starchart.cc/kataras/iris.svg)](https://starchart.cc/kataras/iris) -->
