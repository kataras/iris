# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a> </a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_VN.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-vietnam.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![release](https://img.shields.io/badge/release%20-v14.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) [![view examples](https://img.shields.io/badge/examples%20-59-a83adf.svg?style=for-the-badge&logo=go)](_examples/README.md) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.1) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate)

Iris 是一套快速、功能完備的 Go 網頁框架。它為你的下一個 API 或網站提供表達力十足的路由（支援型別化路徑參數）、集中統一的錯誤處理、讓程式碼保持可測試性的相依性注入，以及開箱即用即安全的正式環境預設值。

**第 14 版**是本專案史上規模最大的一次發布。整個框架圍繞 Go 泛型重建：由編譯器強制檢查的應用程式建構器、自動為你完成驗證的型別化請求輔助函式、由相依性容器解析的控制器，以及只需設定一次的錯誤處理管線。[HISTORY.md](HISTORY.md) 記錄了所有新內容；[遷移指南](MIGRATION.md)能讓 v12 應用程式一次到位完成升級。

> 🤖 **向 AI 詢問 Iris。** Google 免費的 [Iris Code Wiki](https://codewiki.google/github.com/kataras/iris) 直接由本儲存庫產生，並隨著每一次提交與儲存庫保持同步。你可以透過圖表瀏覽架構，也可以用日常語言向它內建的聊天提問，獲得以實際原始碼為依據、關於程式碼庫的解答。

看看[其他人如何評價 Iris](https://www.iris-go.com/#review)，並為這個開源專案**[點亮 Star](https://github.com/kataras/iris/stargazers)**以支持它的發展。好奇它的速度？請參閱獨立的[伺服器基準測試](https://github.com/kataras/server-benchmarks)。

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

<details><summary>一條運算式，寫出整套正式環境技術堆疊</summary>

實際的應用程式會使用 `iris.NewBuilder`：一條由編譯器強制檢查的鏈式呼叫，負責串接 CORS、壓縮、存取記錄、健康檢查端點、你的錯誤對應表、你的服務以及 API 群組。步驟順序寫錯，編譯器會直接拒絕。

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

處理器可以始終保持六行左右，因為解碼、驗證與錯誤呈現都發生在同一個集中設定的地方：

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

[入門教學](TUTORIAL.md)用一份文件，就在這套技術堆疊上打造出一個完整、附測試的服務。

</details>


正如一位 [Go 開發者](https://twitter.com/dkuye/status/1532087942696554497)所說，**Iris 面面俱到，多年來始終堅若磐石**。

Iris v14 為你提供：

* 具型別化路徑參數的路由（`{id:uuid}`、`{n:int}`、`{addr:email}` 等），支援自訂巨集與約束。無效輸入在進入處理器之前就會被拒絕。
* `iris.NewBuilder`：CORS、壓縮、存取記錄、健康檢查端點、錯誤對應、服務與 API 群組，在一條經編譯期檢查的鏈式呼叫中完成組裝。
* 相依性注入：建構函式宣告自己需要什麼，容器負責提供。支援啟動時的 `Initializer` 掛勾與自動執行的關閉清理器。
* `rest` 輔助函式與一張集中的錯誤對應表：處理器保持精簡，錯誤規範化且機器可讀，內部細節絕不外洩給用戶端。
* 能自我驗證的請求內容：實作 `Validate() error`，`ctx.BindJSON` 會在解碼後自動執行它。
* 三十一個內建中介軟體：accesslog、API 金鑰、basicauth、bodylimit、cache、CORS、hCaptcha 與 reCAPTCHA、JWT、monitor、pprof、速率限制、recovery、請求 ID、rewrite、工作階段（session）、版本控管等等。
* 涵蓋各種規模的身分驗證：一行程式碼的 Basic Auth、API 金鑰、泛型 JWT SDK、完整的身分伺服器以及一個 OAuth2 提供者。
* 以 Go 泛型打造的 MVC 控制器。
* WebSocket、gRPC 服務與 API 版本控管。
* 視圖引擎：HTML、Django、Handlebars、Pug 等，透過 iris-contrib/views 提供。
* 支援複數形式與語言偵測的在地化（i18n），外加網站地圖（sitemap）。
* 採用安全 Cookie 預設值的工作階段（`Secure`、`SameSite=Lax` 開箱即用）。
* 豐富的回應形式：JSON、JSONP、XML、YAML、Markdown、Protocol Buffers、MessagePack、內容協商、串流與 Server-Sent Events。
* AI 後端：`ai` 套件透過 HTTP 呈現並轉送模型的串流輸出（文字增量、推理軌跡、工具呼叫片段、用量）。它不綁定任何供應商，也不附帶任何廠商的 SDK。
* 一個 HTTP 用戶端，即 `client` 套件，用與伺服器端相同的 JSON 規則呼叫其他服務。
* 回應壓縮：gzip、deflate、brotli、snappy、s2、zstd。
* 基於 Let's Encrypt 的 TLS，以及內建的 ngrok 通道 (opt-in: `import _ "github.com/kataras/iris/v14/core/server/ngrok"`)，幾秒鐘就能把開發中的應用程式放上網際網路。
* 記憶體內 HTTP 測試：不佔連接埠、不經網路、無需 sleep 的全應用程式測試。
* 一本可免費線上閱讀與下載的完整書籍、一篇實戰教學和一個即時 AI wiki（見下方「學習 Iris」）。
* 還有最重要的一點：從第一天到現在，你都能獲得快速的解答與支援。

## 📖 學習 Iris

> 初次接觸 Iris？從[入門教學](TUTORIAL.md)開始——一份文件帶你從 `go mod init` 一路走到一個通過測試、具備正式環境樣貌的 REST API。正從 v12 升級？請參閱[遷移指南](MIGRATION.md)。過程中有任何疑問？向免費的**即時 AI wiki** 提問：[codewiki.google/github.com/kataras/iris](https://codewiki.google/github.com/kataras/iris)——由 Google 從 Iris 原始碼產生並與之保持同步，內建的聊天可以回答關於程式碼庫的問題。 <!-- human-writing: ok, —— 是中文破折號 -->

### 安裝

唯一的前置需求是 [Go 程式語言](https://go.dev/dl/)，1.27 或更新版本。

#### 建立新專案

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v14@latest # or @v14.0.1
```

<details><summary>在既有專案中安裝</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v14@latest
```

**執行**

```sh
$ go mod tidy
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris 提供了詳盡而完整的**[文件](https://www.iris-go.com/docs)**，讓框架上手變得輕而易舉。

需要更深入的技術文件，請造訪我們的 [godocs](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.1)。想完整、動手體驗推薦的 v14 技術堆疊，請閱讀[入門教學](TUTORIAL.md)；如果你正從 v12 升級，請依照[遷移指南](MIGRATION.md)進行。

更喜歡從程式碼學起？[**59 個範例**](_examples/README.md) 從第一條路由一路帶到依賴注入、身分驗證、測試與框架的擴充點。每一個都在提交前完成編譯，並透過 HTTP 實際跑過一遍。

### Iris 之書

<a href="https://iris-go.com/book/"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=14" /> </a>

完整的書籍包含一篇前言、24 個章節與一篇後記，每個程式碼範例都對照框架原始碼驗證過。每章開頭都有給新手的白話背景介紹，結尾附上精選的延伸閱讀。所有章節都可以在 [iris-go.com/docs](https://iris-go.com/docs/) 免費閱讀，完整目錄見[書籍頁面](https://iris-go.com/book/)。

PDF 同樣免費。[直接下載](https://iris-go.com/iris-book.pdf)即可：不必填寫表單，也不必申請。如果這本書對你有幫助，[贊助本專案](https://iris-go.com/donate/)就是你參與 Iris 開發的方式。

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

## 🤖 Iris 的 AI 工具

有兩個 AI 工具專門了解這個框架，而不只是通用模型對 Go 的了解。

[即時 AI wiki](https://codewiki.google/github.com/kataras/iris) 免費且在瀏覽器中執行：瀏覽架構，或就程式碼庫向它的聊天提問。

[Plexon AI](https://plexon.ai) 是給 Windows、macOS 與 Linux 使用的桌面助理，由 Iris 的作者開發。它內建了 Iris skill（[iris-go.com/ai](https://iris-go.com/ai/)）：v14 API、builder 鏈、`rest` 輔助函式、錯誤對應、認證 SDK 與測試模式，共二十一份參考文件，助理在為你寫程式時按需載入。在 Skills 面板中啟用該 skill，或安裝 [Software Developer 人格](https://plexon.ai/personas/software-developer/)，它會連同其餘開發工具一起啟用。[下載 Plexon AI](https://plexon.ai/download/)。

---

關於支持者、參與貢獻、安全性政策與授權條款，請參閱[英文版 README](README.md)。

本專案採用 [BSD 3-clause 授權條款](LICENSE)，與 Go 專案本身相同。
