<!-- Same as README_ZH_HANS.md -->
# <a href="https://iris-go.com"><img src="https://iris-go.com/images/logo-new-lq-45.png"></a> Iris Web Framework <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg" /> <a href="README_JA.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-japan.svg" /></a> </a> <a href="README_FR.md"><img width="20px" src="https://iris-go.com/images/flag-france.svg" /></a> <a href="README_ZH_HANT.md"><img width="20px" src="https://iris-go.com/images/flag-taiwan.svg" /></a> <a href="README_ZH_HANS.md"><img width="20px" src="https://iris-go.com/images/flag-china.svg" /></a> <a href="README_ES.md"><img width="20px" src="https://iris-go.com/images/flag-spain.png" /></a> <a href="README_FA.md"><img width="20px" src="https://iris-go.com/images/flag-iran.svg" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg" /></a> <a href="README_KO.md"><img width="20px" src="https://iris-go.com/images/flag-south-korea.svg?v=12" /></a> <a href="README_PT_BR.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-brazil.svg" /></a> <a href="README_VN.md"><img width="20px" height="20px" src="https://iris-go.com/images/flag-vietnam.svg" /></a>

[![build status](https://img.shields.io/github/actions/workflow/status/kataras/iris/ci.yml?branch=main&style=for-the-badge)](https://github.com/kataras/iris/actions/workflows/ci.yml) [![release](https://img.shields.io/badge/release%20-v14.0-0077b3.svg?style=for-the-badge)](https://github.com/kataras/iris/releases) [![view examples](https://img.shields.io/badge/examples%20-59-a83adf.svg?style=for-the-badge&logo=go)](_examples/README.md) [![godocs](https://img.shields.io/badge/go-%20docs-488AC7.svg?style=for-the-badge)](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0) [![chat](https://img.shields.io/gitter/room/iris_go/community.svg?color=cc2b5e&logo=gitter&style=for-the-badge)](https://gitter.im/iris_go/community) [![donate](https://img.shields.io/badge/support-Iris-blue.svg?style=for-the-badge&logo=paypal)](https://iris-go.com/donate)

Iris 是一个快速、功能齐备的 Go Web 框架。它为你的下一个 API 或网站提供表达力十足的路由（支持类型化路径参数）、集中统一的错误处理、让代码保持可测试性的依赖注入，以及开箱即用即安全的生产级默认配置。

**第 14 版**是本项目历史上规模最大的一次发布。整个框架围绕 Go 泛型重建：由编译器强制校验的应用构建器、自动为你完成校验的类型化请求辅助函数、由依赖容器解析的控制器，以及只需配置一次的错误处理管线。[HISTORY.md](HISTORY.md) 记录了全部新内容；[迁移指南](MIGRATION.md)能让 v12 应用一次性完成升级。

> 🤖 **向 AI 咨询 Iris。** Google 免费的 [Iris Code Wiki](https://codewiki.google/github.com/kataras/iris) 由本仓库直接生成，并随每一次提交与仓库保持同步。你可以借助图表浏览架构，也可以用自然语言向它的内置聊天提问，获得基于真实源码的、关于代码库的解答。

看看[其他人如何评价 Iris](https://www.iris-go.com/#review)，并为这个开源项目**[点亮 Star](https://github.com/kataras/iris/stargazers)**以支持它的发展。想了解速度表现？请查看独立的[服务器基准测试](https://github.com/kataras/server-benchmarks)。

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

<details><summary>一条表达式，写出整套生产技术栈</summary>

真实的应用会使用 `iris.NewBuilder`：一条由编译器强制校验的链式调用，负责装配 CORS、压缩、访问日志、健康检查端点、你的错误映射、你的服务以及 API 分组。步骤顺序写错，编译器会直接拒绝。

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

处理器可以始终保持六行左右，因为解码、校验和错误渲染都发生在同一个集中配置的地方：

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

[入门教程](TUTORIAL.md)用一篇文档，就在这套技术栈上构建出一个完整、带测试的服务。

</details>


正如一位 [Go 开发者](https://twitter.com/dkuye/status/1532087942696554497)所说，**Iris 面面俱到，多年来始终坚如磐石**。

Iris v14 为你提供：

* 带类型化路径参数的路由（`{id:uuid}`、`{n:int}`、`{addr:email}` 等），支持自定义宏与约束。非法输入在进入处理器之前就会被拒绝。
* `iris.NewBuilder`：CORS、压缩、访问日志、健康检查端点、错误映射、服务与 API 分组，在一条经编译期检查的链式调用中完成装配。
* 依赖注入：构造函数声明自己需要什么，容器负责提供。支持启动时的 `Initializer` 钩子和自动执行的关闭清理器。
* `rest` 辅助函数与一张集中的错误映射表：处理器保持轻薄，错误规范化且机器可读，内部细节绝不泄露给客户端。
* 能自我校验的请求负载：实现 `Validate() error`，`ctx.BindJSON` 会在解码后自动执行它。
* 三十一个内置中间件：accesslog、API 密钥、basicauth、bodylimit、cache、CORS、hCaptcha 与 reCAPTCHA、JWT、monitor、pprof、限流、recovery、请求 ID、rewrite、会话、版本控制等等。
* 覆盖各种规模的身份认证：一行代码的 Basic Auth、API 密钥、泛型 JWT SDK、完整的身份服务器以及一个 OAuth2 提供者。
* 基于 Go 泛型构建的 MVC 控制器。
* WebSocket、gRPC 服务与 API 版本控制。
* 视图引擎：HTML、Django、Handlebars、Pug 等，通过 iris-contrib/views 提供。
* 支持复数形式与语言检测的本地化（i18n），外加站点地图。
* 采用安全 Cookie 默认值的会话（`Secure`、`SameSite=Lax` 开箱即用）。
* 丰富的响应形式：JSON、JSONP、XML、YAML、Markdown、Protocol Buffers、MessagePack、内容协商、流式传输与 Server-Sent Events。
* AI 后端：`ai` 包通过 HTTP 渲染并转发模型的流式输出（文本增量、推理轨迹、工具调用片段、用量）。它不绑定任何提供商，也不附带任何厂商的 SDK。
* 一个 HTTP 客户端，即 `client` 包，用与服务端相同的 JSON 规则调用其他服务。
* 响应压缩：gzip、deflate、brotli、snappy、s2、zstd。
* 基于 Let's Encrypt 的 TLS，以及内置的 ngrok 隧道 (opt-in: `import _ "github.com/kataras/iris/v14/core/server/ngrok"`)，几秒钟就能把开发中的应用放到公网上。
* 内存内 HTTP 测试：不占端口、不走网络、无需 sleep 的全应用测试。
* 一本可免费在线阅读和下载的完整图书、一篇实战教程和一个实时 AI wiki（见下方"学习 Iris"）。
* 还有最重要的一点：从第一天直到今天，你都能获得快速的解答与支持。

## 📖 学习 Iris

> 初次接触 Iris？从[入门教程](TUTORIAL.md)开始——一篇文档带你从 `go mod init` 一路走到一个经过测试、具备生产形态的 REST API。正在从 v12 升级？请看[迁移指南](MIGRATION.md)。过程中有任何疑问？向免费的**实时 AI wiki** 提问：[codewiki.google/github.com/kataras/iris](https://codewiki.google/github.com/kataras/iris)——由 Google 从 Iris 源码生成并与之保持同步，内置的聊天可以回答关于代码库的问题。 <!-- human-writing: ok, —— 是中文破折号 -->

### 安装

唯一的前置要求是 [Go 编程语言](https://go.dev/dl/)，1.27 或更高版本。

#### 创建新项目

```sh
$ mkdir myapp
$ cd myapp
$ go mod init myapp
$ go get github.com/kataras/iris/v14@latest # or @v14.0.0
```

<details><summary>在现有项目中安装</summary>

```sh
$ cd myapp
$ go get github.com/kataras/iris/v14@latest
```

**运行**

```sh
$ go mod tidy
$ go run .
```

</details>

![](https://www.iris-go.com/images/gifs/install-create-iris.gif)

Iris 提供了详尽而全面的**[文档](https://www.iris-go.com/docs)**，让框架上手变得轻而易举。

如需更细致的技术文档，请前往我们的 [godocs](https://pkg.go.dev/github.com/kataras/iris/v14@v14.0.0)。想完整、动手地了解推荐的 v14 技术栈，请阅读[入门教程](TUTORIAL.md)；如果你正在从 v12 升级，请按照[迁移指南](MIGRATION.md)操作。

更喜欢从代码里学？[**59 个示例**](_examples/README.md) 从第一个路由一路讲到依赖注入、认证、测试和框架的扩展点。每一个都在提交前完成编译，并通过 HTTP 实际跑通。

### Iris 之书

<a href="https://iris-go.com/book/"> <img alt="Book cover" src="https://iris-go.com/images/iris-book-cover-sm.jpg?v=14" /> </a>

完整的图书包含一篇前言、24 个章节和一篇后记，每个代码示例都对照框架源码验证过。每章开头都有面向新手的通俗背景介绍，结尾附有精选的延伸阅读。所有章节都可以在 [iris-go.com/docs](https://iris-go.com/docs/) 免费阅读，完整目录见[图书页面](https://iris-go.com/book/)。

PDF 同样免费。[直接下载](https://iris-go.com/iris-book.pdf)即可：无需填写表单，也无需申请。如果这本书对你有帮助，[赞助本项目](https://iris-go.com/donate/)就是你参与 Iris 开发的方式。

[![follow author on twitter](https://img.shields.io/twitter/follow/makismaropoulos?color=3D8AA3&logoColor=3D8AA3&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=makismaropoulos)

[![follow Iris web framework on twitter](https://img.shields.io/twitter/follow/iris_framework?color=ee7506&logoColor=ee7506&style=for-the-badge&logo=twitter)](https://twitter.com/intent/follow?screen_name=iris_framework)

[![follow Iris web framework on facebook](https://img.shields.io/badge/Follow%20%40Iris.framework-569-2D88FF.svg?style=for-the-badge&logo=facebook)](https://www.facebook.com/iris.framework)

## 🤖 Iris 的 AI 工具

有两个 AI 工具专门了解这个框架，而不只是通用模型对 Go 的了解。

[实时 AI wiki](https://codewiki.google/github.com/kataras/iris) 免费且在浏览器中运行：浏览架构，或就代码库向它的聊天提问。

[Plexon AI](https://plexon.ai) 是面向 Windows、macOS 和 Linux 的桌面助手，由 Iris 的作者开发。它内置了 Iris skill（[iris-go.com/ai](https://iris-go.com/ai/)）：v14 API、builder 链、`rest` 辅助函数、错误映射、认证 SDK 和测试模式，共二十一份参考文档，助手在为你写代码时按需加载。在 Skills 面板中启用该 skill，或安装 [Software Developer 人格](https://plexon.ai/personas/software-developer/)，它会连同其余开发工具一起启用。[下载 Plexon AI](https://plexon.ai/download/)。

---

关于支持者、参与贡献、安全策略与许可证，请参阅[英文版 README](README.md)。

本项目基于 [BSD 3-clause 许可证](LICENSE)授权，与 Go 项目本身一致。
