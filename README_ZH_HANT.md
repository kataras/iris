<!-- [![黑人的命也是命](https://iris-go.com/images/blacklivesmatter_banner.png)](https://support.eji.org/give/153413/#!/donation/checkout)

# 新聞

> 此為 **開發中** 分支——功能不僅最新，而且最好。敬請期待接下來的發行版本 [v12.2.0](HISTORY.md#Next)。若需比較穩定的分支，請改前往 [v12.1.8 分支](https://github.com/kataras/iris/tree/v12.1.8)。
>
> ![](https://iris-go.com/images/cli.png) 立刻試試看官方的 [Iris 命令列介面 (CLI)](https://github.com/kataras/iris-cli)！

> 因為工作量過大，[問題](https://github.com/kataras/iris/issues) 解答的速度可能會有所延宕。 -->

<!-- ![](https://iris-go.com/images/release.png) Iris 的 **12.1.8** 版本已經 [釋出](HISTORY.md#su-16-february-2020--v1218)! -->

# Iris Web 框架 <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /></a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a>

[![組建狀態](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![查看範例](https://img.shields.io/badge/examples%20-285-a83adf.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/main/_examples) [![聊天室](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) <!--[![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)--> [![捐助](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate) <!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.10)--> <!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->

<!-- <a href="https://iris-go.com"> <img align="right" src="https://iris-go.com/images/logo-w169.png"></a> -->

Iris 是款不僅迅速、簡捷，並且功能完善、高效率的 Go 語言 Web 框架。**與 Go 生態系統中其它人提供的免費軟體套件不同，這個軟體保證終身主動維護。**

> 想要取得接下來 **v12.2.0** 穩定版本（正在逐步推進 (2023🎅)）的新消息，請收藏 🌟 並關注 👀 這個儲存庫！

Iris 能為你的下一個網站或 API，立下漂亮、富有表達性，且易於使用的基礎。

```go
package main

import "github.com/kataras/iris/v12"

func main() {
  app := iris.New()
  app.Use(iris.Compression)

  app.Get("/", func(ctx iris.Context) {
    ctx.HTML("哈囉，<strong>%s</strong>！", "世界")
  })

  app.Listen(":8080")
}
```

<!-- <details><summary>More with simple Handler</summary>

```go
package main

import "github.com/kataras/iris/v12"

type (
  request struct {
    Firstname string `json:"firstname"`
    Lastname  string `json:"lastname"`
  }

  response struct {
    ID      string `json:"id"`
    Message string `json:"message"`
  }
)

func main() {
  app := iris.New()
  app.Handle("PUT", "/users/{id:uuid}", updateUser)
  app.Listen(":8080")
}

func updateUser(ctx iris.Context) {
  id := ctx.Params().Get("id")

  var req request
  if err := ctx.ReadJSON(&req); err != nil {
    ctx.StopWithError(iris.StatusBadRequest, err)
    return
  }

  resp := response{
    ID:      id,
    Message: req.Firstname + " updated successfully",
  }
  ctx.JSON(resp)
}
```

> Read the [routing examples](https://github.com/kataras/iris/blob/main/_examples/routing) for more!

</details>

<details><summary>Handler with custom input and output arguments</summary>

[![https://github.com/kataras/iris/blob/main/_examples/dependency-injection/basic/main.go](https://user-images.githubusercontent.com/22900943/105253731-b8db6d00-5b88-11eb-90c1-0c92a5581c86.png)](https://twitter.com/iris_framework/status/1234783655408668672)

> Interesting? Read the [examples](https://github.com/kataras/iris/blob/main/_examples/dependency-injection).

</details>

<details><summary>Party Controller (NEW)</summary>

> Head over to the [full running example](https://github.com/kataras/iris/blob/main/_examples/routing/party-controller)!

</details>

<details><summary>MVC</summary>

```go
package main

import (
  "github.com/kataras/iris/v12"
  "github.com/kataras/iris/v12/mvc"
)

type (
  request struct {
    Firstname string `json:"firstname"`
    Lastname  string `json:"lastname"`
  }

  response struct {
    ID      uint64 `json:"id"`
    Message string `json:"message"`
  }
)

func main() {
  app := iris.New()
  mvc.Configure(app.Party("/users"), configureMVC)
  app.Listen(":8080")
}

func configureMVC(app *mvc.Application) {
  app.Handle(new(userController))
}

type userController struct {
  // [...dependencies]
}

func (c *userController) PutBy(id uint64, req request) response {
  return response{
    ID:      id,
    Message: req.Firstname + " updated successfully",
  }
}
```

Want to see more? Navigate through [mvc examples](_examples/mvc)!
</details>


<details><summary>API Guide <strong>HOT</strong></summary>

```go
package main

import (
  // [other packages...]

  "github.com/kataras/iris/v12"
)

func main() {
  iris.NewGuide().
    AllowOrigin("*").
    Compression(true).
    Health(true, "development", "kataras").
    Timeout(0, 20*time.Second, 20*time.Second).
    Middlewares(basicauth.New(...)).
    Services(
        // NewDatabase(),
        // NewPostgresRepositoryRegistry,
        // NewUserService,
    ).
    API("/users", new(UsersAPI)).
    Listen(":80")
}
```

</details>

<br/>

-->

據一位 [Go 開發者](https://twitter.com/dkuye/status/1532087942696554497) 所言，**Iris 能向您提供全方位的服務，並地位多年來屹立不搖**。

Iris 提供了至少這些功能：

- HTTP/2 (Push, 甚至是 Embedded 資料)
- 中介模組（存取日誌、基礎認證、CORS、gRPC、防機器人 hCaptcha、JWT、方法覆寫、模組版本顯示、監控、PPROF、速率限制、防機器人 reCaptcha、panic 救援、請求識別碼、重寫請求）
- API 分版 (versioning)
- MVC (Model-View-Controller) 模式
- Websocket
- gRPC
- 自動啟用 HTTPS
- 內建 ngrok 支援，讓您可以把 app 以最快速的方式推上網際網路
- 包含動態路徑、具唯一性的路由，支援如 :uuid、:string、:int 等等的標準類型，並且可以自己建立
- 壓縮功能
- 檢視 (View) 算繪引擎 (HTML、Django、Handlebars、Pug/Jade 等等）
- 建立自己的檔案伺服器，並寄存您自己的 WebDAV 伺服器
- 快取
- 本地化 (i18n、sitemap）
- 連線階段管理
- 豐富的回應格式（HTML、純文字、Markdown、XML、YAML、二進位、JSON、JSONP、Protocol Buffers、MessagePack、(HTTP) 內容協商、串流、Server-Sent Events 等）
- 回應壓縮功能（gzip、deflate、brotli、snappy、s2）
- 豐富的請求方式（綁定 URL 查詢、標頭、文字、XML、YAML、二進位、JSON、資料驗證、Protocol Buffers、MessagePack 等）
- 依賴注入（MVC、處理常式 (handler)、API 路由）
- 測試套件
- 最重要的是…… 從發行第一天到現在（已經整整六年），解答與支援一直都十分迅速！

看看別人 [是如何評價 Iris 的](https://www.iris-go.com/#review)，並且 **[給這個開放原始碼專案一顆小星星](https://github.com/kataras/iris/stargazers)**，支持專案的潛力。

[![](https://iris-go.com/images/reviews.gif)](https://iris-go.com/testimonials/)

[![Benchmarks: Jul 18, 2020 at 10:46am (UTC)](https://iris-go.com/images/benchmarks.svg)](https://github.com/kataras/server-benchmarks)

## 👑 <a href="https://iris-go.com/donate">支援者</a>

你的一臂之力，能夠為大家帶來更好的開放原始碼 Web 開發體驗！

> [@github](https://github.com/github) is now sponsoring you for $550.00 one time.
>
> A note from your new sponsor:
>
> To celebrate Maintainer Month we want to thank you for all you do for the open source community. Check out our blog post to learn more about how GitHub is investing in maintainers. https://github.blog/2022-06-24-thank-you-to-our-maintainers/

> 現已支援來自 [中國](https://github.com/kataras/iris/issues/1870#issuecomment-1101418349) 的捐款！

## 📖 學習 Iris

### 安裝

只要先安裝好 [Go 程式語言](https://go.dev/dl/) 即可。

#### 建立新專案

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v12@latest # 或 @v12.2.10
```

<details><summary>在現有專案安裝</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v12@latest
```

**執行**

```sh
$ go mod tidy -compat=1.20 # Windows 的話，請試試 -compat="1.20"
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris 包含極其豐富且透徹的 **[文件](https://www.iris-go.com/docs)**，讓框架的入門觸手可及。

<!-- Iris contains extensive and thorough **[wiki](https://github.com/kataras/iris/wiki)** making it easy to get started with the framework. -->

<!-- ![](https://media.giphy.com/media/Ur8iqy9FQfmPuyQpgy/giphy.gif) -->

如需更為詳細的技術性文件，您可以前往我們的 [godocs](https://pkg.go.dev/github.com/kataras/iris/v12@main)。如果要可以直接執行的程式碼，可以到 [./\_examples](_examples) 儲存庫的子目錄參閱。

### 想一邊旅行、一邊閱讀嗎？

<a href="https://iris-go.com/#book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12" /> </a>

[![在 Twitter 上追蹤作者](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![在 Twitter 上追蹤 Iris Web 框架](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![在 Facebook 上追蹤 Iris Web 框架](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

您現在可以 [請求索取](https://www.iris-go.com/#ebookDonateForm) **Iris 電子書**（新版，**針對未來版本 v12.2.0+**) 的 PDF 和線上閱讀存取權限，並共同參與 Iris 的開發。

## 🙌 貢獻

我們殷切期盼你對 Iris Web 框架的貢獻！有關貢獻 Iris 專案的更多資訊，請參閱 [CONTRIBUTING.md](CONTRIBUTING.md) 檔案。

[所有貢獻者名單](https://github.com/kataras/iris/graphs/contributors)

## 🛡 安全性漏洞

如果你發現 Iris 中有安全性漏洞，請寄一封電子郵件至 [iris-go@outlook.com](mailto:iris-go@outlook.com)。我們會儘速解決所有安全性漏洞。

## 📝 授權條款

本專案和 Go 語言相同，皆採 [BSD 3-clause 授權條款](LICENSE) 授權。

專案的名稱「Iris」取材自希臘神話。

<!-- ## Stargazers over time

[![Stargazers over time](https://starchart.cc/kataras/iris.svg)](https://starchart.cc/kataras/iris) -->
