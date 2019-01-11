# Iris ウェブフレームワーク <a href="README.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /></a> <a href="README_ZH.md"> <img width="20px" src="https://iris-go.com/images/flag-china.svg?v=10" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg?v=10" /></a> <a href="README_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a> <a href="README_PT_BR.md"><img width="20px" src="https://iris-go.com/images/flag-pt-br.svg?v=10" /></a>

<a href="https://iris-go.com"> <img align="right" width="169px" src="https://iris-go.com/images/icon.svg?v=a" title="logo created by @merry.dii" /> </a>

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)<!-- [![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)--> [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris) [![vscode-iris](https://img.shields.io/badge/ext%20-vscode-0c77e3.svg?style=flat-square)](https://marketplace.visualstudio.com/items?itemName=kataras2006.iris)<!--[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)--> [![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris) [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples/routing) [![release](https://img.shields.io/badge/release%20-v11.1-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/releases)

Irisはシンプルで高速、それにも関わらず充実した機能を有する効率的なGo言語のウェブフレームワークです。

Irisは表現力豊かなウェブサイトやAPIの基礎構造をいとも簡単に提供します。

Go言語におけるExpressjsと言っても過言ではないでしょう。

[皆様の声](#支援)をご覧ください。このレポジトリを[Star](https://github.com/kataras/iris/stargazers)し、[最新情報](https://facebook.com/iris.framework)を受け取りましょう。

## 支援者

支援者の方々、ありがとうございます! 🙏 [支援者になる](https://iris-go.com/donate)

<a href="https://iris-go.com/donate" target="_blank"><img src="https://iris-go.com/backers.svg?v=2"/></a>

```sh
$ cat example.go
```

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // Load all templates from the "./views" folder
    // where extension is ".html" and parse them
    // using the standard `html/template` package.
    app.RegisterView(iris.HTML("./views", ".html"))

    // Method:    GET
    // Resource:  http://localhost:8080
    app.Get("/", func(ctx iris.Context) {
        // Bind: {{.message}} with "Hello world!"
        ctx.ViewData("message", "Hello world!")
        // Render template file: ./views/hello.html
        ctx.View("hello.html")
    })

    // Method:    GET
    // Resource:  http://localhost:8080/user/42
    //
    // Need to use a custom regexp instead?
    // Easy,
    // just mark the parameter's type to 'string'
    // which accepts anything and make use of
    // its `regexp` macro function, i.e:
    // app.Get("/user/{id:string regexp(^[0-9]+$)}")
    app.Get("/user/{id:long}", func(ctx iris.Context) {
        userID, _ := ctx.Params().GetInt64("id")
        ctx.Writef("User ID: %d", userID)
    })

    // Start the server using a network address.
    app.Run(iris.Addr(":8080"))
}
```

> [here](_examples/routing/dynamic-path/main.go#L31)をクリックしてパスパラメータについて学ぶ

```html
<!-- file: ./views/hello.html -->
<html>
<head>
    <title>Hello Page</title>
</head>
<body>
    <h1>{{.message}}</h1>
</body>
</html>
```

```sh
$ go run example.go
Now listening on: http://localhost:8080
Application Started. Press CTRL+C to shut down.
_
```

## インストール

[Go Programming Language](https://golang.org/dl/)をインストールしていることが唯一の前提条件です。

```sh
$ go get -u github.com/kataras/iris
```

Irisは[vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo)機能の利点を活かしています。これが上流レポジトリの変更や削除を防ぐため、再現可能なビルドを実現します。

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new-gray.png)](_benchmarks/README_UNIX.md)

_Updated at: [Tuesday, 21 November 2017](_benchmarks/README_UNIX.md)_

<details>
<summary>他のウェブフレームワークとの比較</summary>

![Comparison with other frameworks](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

</details>

## 支援

- [HISTORY](HISTORY.md#fr-11-january-2019--v1111)ファイルはあなたの友人です。このファイルには、機能に関する最新の情報や変更点が記載されています。
- バグを発見しましたか？[github issues](https://github.com/kataras/iris/issues)に投稿をお願い致します。
- 質問がありますか？または問題を即時に解決するため、熟練者に相談する必要がありますか？[community chat](https://chat.iris-go.com)に参加しましょう。
- [here](https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link)をクリックしてユーザーとしての体験を報告しましょう。
- フレームワークを愛していますか?それならばツイートしましょう!他の人はこのようにツイートしています:

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

Irisプロジェクトに貢献して頂ける方は、[CONTRIBUTING.md](CONTRIBUTING.md) をお読みください.

[全貢献者リスト](https://github.com/kataras/iris/graphs/contributors)

## 学習する

ウェブフレームワークで開発を行う時には、まず言語の基本を学ぶこと、標準的なhttpで何ができるのか知ることが重要です。あなたのアプリケーションが個人的なもので、とてもシンプル、パフォーマンスとメンテナンス性にそこまで拘らない場合、標準パッケージでの開発が推奨されます。以下のガイドラインを参照してください。

- **100+1** **[examples](_examples)** や[Irisスターターキット](#Irisスターターキット)を学習する
- より詳しく知るために[godocs](https://godoc.org/github.com/kataras/iris)を読む
- 一息ついて、私たちが発見した[記事](#記事)を読む

### Irisスターターキット

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

1. [snowlyg/IrisApiProject: Iris + gorm + jwt + sqlite3](https://github.com/snowlyg/IrisApiProject) **NEW-Chinese**
2. [yz124/superstar: Iris + xorm to implement the star library](https://github.com/yz124/superstar) **NEW-Chinese**
3. [jebzmos4/Iris-golang: A basic CRUD API in golang with Iris](https://github.com/jebzmos4/Iris-golang)
4. [gauravtiwari/go_iris_app: A basic web app built in Iris for Go](https://github.com/gauravtiwari/go_iris_app)
5. [A mini social-network created with the awesome Iris💖💖](https://github.com/iris-contrib/Iris-Mini-Social-Network)
6. [Iris isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/iris-contrib/iris-starter-kit)
7. [ionutvilie/react-ts: Demo project with react using typescript and Iris](https://github.com/ionutvilie/react-ts)
8. [Self-hosted Localization Management Platform built with Iris and Angular](https://github.com/iris-contrib/parrot)
9. [Iris + Docker and Kubernetes](https://github.com/iris-contrib/cloud-native-go)
10. [nanobox.io: Quickstart for Iris with Nanobox](https://guides.nanobox.io/golang/iris/from-scratch)
11. [hasura.io: A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](https://hasura.io/hub/project/hasura/hello-golang-iris)

> 似たようなものを開発しましたか？ [私たちにも教えてください！](https://github.com/kataras/iris/pulls)

### ミドルウェア

Irisはあなたのウェブアプリケーションにご使用いただけるハンドラー[[1]](middleware/)[[2]](https://github.com/iris-contrib/middleware)を多く有しています。さらに、[net/http](https://golang.org/pkg/net/http/)と互換性のある外部のミドルウェアもご使用いただけます。[examples/convert-handlers](_examples/convert-handlers)を参照してください。

Irisは他のフレームワークと異なり、標準パッケージと１００％互換性があります。故に、米国の有名なテレビ局を含め、大企業のの大半がGoをワークフローに取り入れています。Irisは常にGo言語の最新版リリースに対応し、Goの作成者によって開発されている標準的な`net/http`パッケージに沿っています。 

### 記事

* [CRUD REST API in Iris (a framework for golang)](https://medium.com/@jebzmos4/crud-rest-api-in-iris-a-framework-for-golang-a5d33652401e)
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

### 動画

* [Daily Coding - Web Framework Golang: Iris Framework]( https://www.youtube.com/watch?v=BmOLFQ29J3s) by WarnabiruTV, source: youtube, cost: **FREE**
* [Tutorial Golang MVC dengan Iris Framework & Mongo DB](https://www.youtube.com/watch?v=uXiNYhJqh2I&list=PLMrwI6jIZn-1tzskocnh1pptKhVmWdcbS) (19 parts so far) by Musobar Media, source: youtube, cost: **FREE**
* [Go/Golang 27 - Iris framework : Routage de base](https://www.youtube.com/watch?v=rQxRoN6ub78) by stephgdesign, source: youtube, cost: **FREE**
* [Go/Golang 28 - Iris framework : Templating](https://www.youtube.com/watch?v=nOKYV073S2Y) by stephgdesignn, source: youtube, cost: **FREE**
* [Go/Golang 29 - Iris framework : Paramètres](https://www.youtube.com/watch?v=K2FsprfXs1E) by stephgdesign, source: youtube, cost: **FREE**
* [Go/Golang 30 - Iris framework : Les middelwares](https://www.youtube.com/watch?v=BLPy1So6bhE) by stephgdesign, source: youtube, cost: **FREE**
* [Go/Golang 31 - Iris framework : Les sessions](https://www.youtube.com/watch?v=RnBwUrwgEZ8) by stephgdesign, source: youtube, cost: **FREE**

### 雇用

多くの企業やスタートアップがIrisの使用経験を有するGo言語開発者を探しています。私たちは募集情報を毎日検索し、[facebook page](https://www.facebook.com/iris.framework)に投稿しています。既に投稿されている情報をご覧ください。Likeを押して通知を受け取りましょう。

## ライセンス

Iris[3-Clause BSD License](LICENSE)に基いています。Irisは完全無料のオープンソースソフトウェアです。

ライセンスに関するご質問は[e-mail](mailto:kataras2006@hotmail.com?subject=Iris%20License)までご連絡ください。
