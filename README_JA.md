<!--<h1><img width="24" height="25" src ="https://www.iris-go.com/images/logo-new-lq-45.png"/> News</h1>

 Iris version **12.2.0** has been [released](HISTORY.md#sa-11-march-2023--v1220)! As always, the latest version of Iris comes with the promise of lifetime active maintenance.

Try the official [Iris Command Line Interface](https://github.com/kataras/iris-cli) today! -->

# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /></a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![view examples](https://img.shields.io/badge/examples%20-285-a83adf.svg?style=for-the-badge&logo=go)](https://github.com/kataras/iris/tree/main/_examples) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) <!--[![FOSSA Status](https://img.shields.io/badge/LICENSE%20SCAN-PASSING❤️-CD2956?style=for-the-badge&logo=fossa)](https://app.fossa.io/projects/git%2Bgithub.com%2Fkataras%2Firis?ref=badge_shield)--> [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate) <!--[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/kataras/iris)--><!--[![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v12@v12.2.11)--> <!-- [![release](https://img.shields.io/badge/release%20-v12.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) -->

Irisは、高速でシンプルでありながら、十分な機能を備えた、非常に効率的なGo用Webフレームワークです。

あなたの次のウェブサイトやAPIのために、美しく表現力豊かで使いやすい基盤を提供します。

[Irisについての他の人々の意見](https://www.iris-go.com/#review)を学び、このオープンソースプロジェクトに **[スターをつけて](https://github.com/kataras/iris/stargazers)** 、その可能性を応援しましょう。

[![](https://iris-go.com/images/reviews.gif)](https://iris-go.com/testimonials/)

[![Benchmarks: Jul 18, 2020 at 10:46am (UTC)](https://iris-go.com/images/benchmarks.svg)](https://github.com/kataras/server-benchmarks)

```go
package main

import "github.com/kataras/iris/v12"

func main() {
  app := iris.New()
  app.Use(iris.Compression)

  app.Get("/", func(ctx iris.Context) {
    ctx.HTML("Hello <strong>%s</strong>!", "World")
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

ある[Go開発者](https://twitter.com/dkuye/status/1532087942696554497)が言ったように、 **Irisはあなたをあらゆる面でサポートし、長年にわたって力強さを保ち続けています** 。

Irisが提供する機能の一部:

* HTTP/2 (Push, Embedded data)
* Middleware (Accesslog, Basicauth, CORS, gRPC, Anti-Bot hCaptcha, JWT, MethodOverride, ModRevision, Monitor, PPROF, Ratelimit, Anti-Bot reCaptcha, Recovery, RequestID, Rewrite)
* API バージョニング
* Model-View-Controller
* Websockets
* gRPC
* Auto-HTTPS
* ngrokの組み込みサポートにより、最速の方法でアプリをインターネットに公開できる
* :uuid、:string、:int のような標準的な型を持つダイナミック・パスをパラメータとするユニークなルーター
* Compression
* View Engines (HTML, Django, Handlebars, Pug/Jade and more)
* 独自のファイルサーバーを作成し、WebDAVサーバーをホストする
* Cache
* Localization (i18n, sitemap)
* Sessions
* 豊富な Response (HTML, Text, Markdown, XML, YAML, Binary, JSON, JSONP, Protocol Buffers, MessagePack, Content Negotiation, Streaming, Server-Sent Events など)
* Response Compression (gzip, deflate, brotli, snappy, s2)
* 豊富な Requests (Bind URL Query, Headers, Form, Text, XML, YAML, Binary, JSON, Validation, Protocol Buffers, MessagePack など)
* Dependency Injection (MVC, Handlers, API Routers)
* Testing Suite
* そして最も重要なのは、初日から現在に至るまで、つまり丸6年間、迅速な回答とサポートを受けられることです！

## 👑 <a href="https://iris-go.com/donate">サポーター</a>

皆様のご協力により、オープンソース・ウェブ開発をより良いものにすることができます！

## 📖 Irisを学ぶ

### インストール

必要なのは [Goプログラミング言語](https://go.dev/dl/) だけです。

#### 新規プロジェクトの作成

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v12@latest # or @v12.2.11
```

<details><summary>既存のプロジェクトにインストールする場合</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v12@latest
```

**実行**

```sh
$ go mod tidy -compat=1.20 # -compat="1.20" for windows.
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris には広範で詳細な **[ドキュメント](https://www.iris-go.com/docs)** が含まれているので、フレームワークを簡単に使い始めることができます。

<!-- Iris contains extensive and thorough **[wiki](https://github.com/kataras/iris/wiki)** making it easy to get started with the framework. -->

<!-- ![](https://media.giphy.com/media/Ur8iqy9FQfmPuyQpgy/giphy.gif) -->

より詳細な技術文書については [godocs](https://pkg.go.dev/github.com/kataras/iris/v12@main) をご覧ください。また、実行可能なコードについては、いつでもリポジトリのサブディレクトリ  [./_examples](_examples)  にアクセスできます。

### 旅行中に本を読むのは好きですか？

<a href="https://iris-go.com/#book"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=12" /> </a>

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

**Iris E-Book**（新版、**将来のv12.2.0+**）のPDFとオンライン・アクセスを今すぐ [リクエスト](https://www.iris-go.com/#ebookDonateForm) して、Irisの開発に参加してください。

## 🙌 貢献する

Irisウェブ・フレームワークへの貢献をお待ちしています！Iris プロジェクトへの貢献についての詳細は、 [CONTRIBUTING.md](CONTRIBUTING.md) ファイルをご覧ください。

[全貢献者のリスト](https://github.com/kataras/iris/graphs/contributors)

## 🛡 セキュリティの脆弱性

Iris にセキュリティ上の脆弱性を発見した場合は、 [iris-go@outlook.com](mailto:iris-go@outlook.com) にメールを送ってください。すべてのセキュリティ脆弱性は、速やかに対処されます。

## 📝 ライセンス

このプロジェクトのライセンスは、Goプロジェクトと同様、 [BSD 3-clause license](LICENSE) です。

プロジェクト名の "Iris" はギリシャ神話からインスピレーションを得たものです。

<!-- ## Stargazers over time

[![Stargazers over time](https://starchart.cc/kataras/iris.svg)](https://starchart.cc/kataras/iris) -->
