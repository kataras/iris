# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a> </a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_VN.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-vietnam.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![release](https://img.shields.io/badge/release%20-v14.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) [![view examples](https://img.shields.io/badge/examples%20-59-a83adf.svg?style=for-the-badge&logo=go)](_examples/README.md) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.1) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate)

Iris は、Go のための高速でバッテリー同梱(batteries-included)の Web フレームワークです。次に作る API や Web サイトに、型付きパスパラメータによる表現力豊かなルーティング、一元化されたエラーハンドリング、コードをテスト可能に保つ依存性注入、そして最初から安全なプロダクション向けデフォルトを提供します。

**バージョン 14** は、プロジェクト史上最大のリリースです。フレームワークは Go のジェネリクスを軸に再構築されました。コンパイル時に強制されるアプリケーションビルダー、バリデーションを肩代わりする型付きリクエストヘルパー、依存性コンテナが解決するコントローラ、そして一度設定すれば済むエラーパイプライン。新しい内容はすべて [HISTORY.md](HISTORY.md) に記録されており、[移行ガイド](MIGRATION.md) を使えば v12 アプリケーションを一気に最新化できます。

> 🤖 **AI に Iris について聞いてみましょう。** Google が無料で提供する [Iris の Code Wiki](https://codewiki.google/github.com/kataras/iris) は、このリポジトリから生成され、コミットごとに同期され続けます。図とともにアーキテクチャを眺めることも、チャットにコードベースについて平易な言葉で質問して、実際のソースに基づいた回答を得ることもできます。

[Iris についての評判](https://www.iris-go.com/#review) をチェックし、このオープンソースプロジェクトの開発を支援するために **[スター](https://github.com/kataras/iris/stargazers)** を付けてください。速度が気になりますか?独立系の [サーバーベンチマーク](https://github.com/kataras/server-benchmarks) をご覧ください。

[![](https://iris-go.com/images/reviews.gif)](https://iris-go.com/testimonials/)

```go
package main

import "github.com/kataras/iris/v14"

func main() {
    app := iris.New()

    app.Get("/", func(ctx iris.Context) {
        ctx.JSON(iris.Map{"message": "Hello, World!"})
    })

    app.Get("/greet/{name:string}", func(ctx iris.Context) {
        ctx.WriteString("Hello, " + ctx.Params().GetString("name"))
    })

    app.Listen(":8080")
}
```

<details><summary>プロダクションスタックを、ひとつの式で</summary>

実際のアプリケーションでは `iris.NewBuilder` を使います。CORS、圧縮、
アクセスログ、ヘルスエンドポイント、エラーマップ、サービス、API グループを
配線する、コンパイル時に強制されるチェーンです。ステップの順序を間違えると
コンパイラが拒否します。

```go
iris.NewBuilder().
    Prefix("/api").
    AllowOrigin("*").
    Compression(true).
    LogRequests(true).
    Health(true, "production", "kataras").
    Errors(errors.NewOptions().
        MapErrors(errors.NotFound, catalog.ErrNotFound)).
    Services(catalog.NewRepository, catalog.NewService).
    RouterMiddlewares().
    Middlewares().
    API("/products", api.NewProductsAPI).
    Build().
    Listen(":8080")
```

デコード、バリデーション、エラー描画は設定済みの一箇所で行われるため、
ハンドラは 6 行のまま保てます。

```go
func (api *ProductsAPI) create(ctx iris.Context) {
    input, ok := ctx.BindJSON[catalog.ProductInput]()
    if !ok {
        return // 400 already written: malformed JSON or failed Validate.
    }

    id, err := api.svc.Create(ctx, input)
    ctx.Created(id, err) // 201, or the centrally mapped error.
}
```

[入門チュートリアル](TUTORIAL.md) は、このスタックの上に完全でテスト済みの
サービスを 1 つのドキュメントで構築します。

</details>


ある [Go 開発者](https://twitter.com/dkuye/status/1532087942696554497) がかつて言ったように、**Iris はあらゆる面であなたをカバーし、長年にわたって堅牢であり続けています**。

Iris v14 で得られるもの:

* 型付きパスパラメータ(`{id:uuid}`、`{n:int}`、`{addr:email}` など)、カスタムマクロと制約を備えたルーティング。不正な入力はハンドラが実行される前に拒否されます。
* `iris.NewBuilder`: CORS、圧縮、アクセスログ、ヘルスエンドポイント、エラーマッピング、サービス、API グループを、コンパイル時に検査されるひとつのチェーンに組み立てます。
* 依存性注入: コンストラクタが必要なものを宣言し、コンテナがそれを供給します。起動時の `Initializer` フック、シャットダウン時の自動クローズ。
* `rest` ヘルパーと一元化されたエラーマップ: 薄いハンドラ、正規化された機械可読エラー、内部情報はクライアントに決して漏れません。
* 自らバリデーションするリクエストペイロード: `Validate() error` を実装すれば、`ctx.BindJSON` がデコード後に実行します。
* 31 の組み込みミドルウェア: accesslog、API キー、basicauth、bodylimit、cache、CORS、hCaptcha と reCAPTCHA、JWT、monitor、pprof、レートリミット、recovery、リクエスト ID、rewrite、セッション、バージョニングなど。
* あらゆる規模の認証: 1 行で使える Basic Auth、API キー、ジェネリックな JWT SDK、完全なアイデンティティサーバー、OAuth2 プロバイダ。
* Go のジェネリクスの上に構築された MVC コントローラ。
* WebSocket、gRPC サービング、API バージョニング。
* ビューエンジン: HTML、Django、Handlebars、Pug など(iris-contrib/views 経由)。
* 複数形処理と言語検出を備えたローカライゼーション、そしてサイトマップ。
* 安全なクッキーデフォルトを持つセッション(`Secure`、`SameSite=Lax` が標準)。
* 豊富なレスポンス: JSON、JSONP、XML、YAML、Markdown、Protocol Buffers、MessagePack、コンテントネゴシエーション、ストリーミング、Server-Sent Events。
* AI バックエンド: `ai` パッケージは、ストリーミングされるモデル出力(テキストデルタ、推論トレース、ツール呼び出しの断片、使用量)を HTTP 経由でレンダリングして中継します。特定のプロバイダに依存せず、ベンダーの SDK も同梱しません。
* サーバー側と同じ JSON ルールで他のサービスを呼び出すための HTTP クライアント、`client` パッケージ。
* レスポンス圧縮: gzip、deflate、brotli、snappy、s2、zstd。
* Let's Encrypt による TLS と、開発中のアプリを数秒でインターネットに公開する組み込みの ngrok トンネリング (opt-in: `import _ "github.com/kataras/iris/v14/core/server/ngrok"`)。
* インメモリ HTTP テスト: ポートもネットワークもスリープも不要な、アプリケーション全体のテスト。
* オンラインでの閲覧もダウンロードも無料の完全な書籍、ハンズオンのチュートリアル、ライブな AI ウィキ(下記「Iris を学ぶ」を参照)。
* そして何より大切なこと: 初日から今日まで、迅速な回答とサポートが得られます。

## 📖 Iris を学ぶ

> Iris は初めてですか?[入門チュートリアル](TUTORIAL.md) から始めましょう。
> `go mod init` からテスト済みでプロダクション水準の REST API まで、
> 1 つのドキュメントで導きます。v12 からのアップグレードは
> [移行ガイド](MIGRATION.md) をご覧ください。途中で疑問が湧いたら、無料の
> **ライブ AI ウィキ**
> [codewiki.google/github.com/kataras/iris](https://codewiki.google/github.com/kataras/iris)
> に質問してください。Google が Iris のソースから生成して常に同期を保って
> おり、組み込みのチャットがコードベースに関する質問に答えてくれます。

### インストール

必要なのは [Go プログラミング言語](https://go.dev/dl/) のバージョン 1.27 以降だけです。

#### 新しいプロジェクトを作成する

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v14@latest # or @v14.0.1
```

<details><summary>既存プロジェクトへのインストール</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v14@latest
```

**実行**

```sh
$ go mod tidy
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris には充実した詳細な **[ドキュメント](https://www.iris-go.com/docs)** が揃っており、フレームワークを簡単に使い始められます。

より詳細な技術ドキュメントは [godocs](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.1) をご覧ください。推奨される v14 スタックへの完全なハンズオン入門としては [入門チュートリアル](TUTORIAL.md) を、v12 からアップグレードする場合は [移行ガイド](MIGRATION.md) をどうぞ。

コードから学びたい方へ。[**59 個のサンプル**](_examples/README.md) は最初のルートから依存性注入、認証、テスト、そしてフレームワークの拡張ポイントまでを網羅しています。そのすべてがコミット前にコンパイルされ、HTTP 経由で実際に動作確認されています。

### Iris ブック

<a href="https://iris-go.com/book/"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=14" /> </a>

完全な書籍は序文、24 の章、エピローグで構成され、すべてのコード例はフレームワークのソースに照らして検証済みです。各章は初心者向けの平易な背景説明で始まり、厳選された参考文献で締めくくられます。各章は [iris-go.com/docs](https://iris-go.com/docs/) で無料で読めます。全体の目次は[書籍ページ](https://iris-go.com/book/)にあります。

PDF も無料です。[直接ダウンロード](https://iris-go.com/iris-book.pdf)できます。フォームへの入力もリクエストも必要ありません。本書が役に立ったなら、[プロジェクトのスポンサーになる](https://iris-go.com/donate/)ことが Iris の開発に参加する方法です。

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

## 🤖 Iris のための AI ツール

一般的なモデルが Go について知っている範囲を超えて、この framework を具体的に理解している AI ツールが 2 つあります。

[ライブ AI wiki](https://codewiki.google/github.com/kataras/iris) は無料で、ブラウザ上で動作します。アーキテクチャを閲覧したり、チャットにコードベースについて質問したりできます。

[Plexon AI](https://plexon.ai) は Windows、macOS、Linux 向けのデスクトップアシスタントで、Iris の作者が開発しています。Iris skill（[iris-go.com/ai](https://iris-go.com/ai/)）を内蔵しており、v14 の API、builder チェーン、`rest` ヘルパー、エラーマッピング、認証 SDK、テストのパターンを 21 個のリファレンス文書として持ち、コードを書きながら必要なものを読み込みます。Skills パネルから skill を有効にするか、[Software Developer ペルソナ](https://plexon.ai/personas/software-developer/)をインストールしてください。ペルソナを入れると、他の開発ツールと一緒に skill も有効になります。[Plexon AI をダウンロード](https://plexon.ai/download/)。

---

サポーター、コントリビューション、セキュリティポリシー、ライセンスについては [英語版 README](README.md) をご覧ください。

このプロジェクトは、Go プロジェクト自身と同じく [BSD 3-clause ライセンス](LICENSE) の下でライセンスされています。
