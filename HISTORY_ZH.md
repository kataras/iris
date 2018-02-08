# 更新记录 <a href="HISTORY.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /></a> <a href="HISTORY_GR.md"> <img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a>

### 想得到免费即时的支持?

    https://github.com/kataras/iris/issues
    https://chat.iris-go.com

### 获取历史版本?

    https://github.com/kataras/iris/releases

### 我是否应该升级 Iris?

如果没有必要，不会强制升级。如果你已经准备好了，可以随时升级。

> Iris 使用 Golang 的 [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) 特性, 避免依赖包的更改带来影响。

**如何升级**: 打开命令行执行以下命令: `go get -u github.com/kataras/iris` 或者等待自动更新。

# Th, 08 February 2018 | v10.2.0

This history entry is not yet translated to Chinese. Please read [the english version instead](https://github.com/kataras/iris/blob/master/HISTORY.md#th-08-february-2018--v1020).

# 2018 2月6号 | v10.1.0 版本更新

新特性：

- 多级域名跳转, 相关示例 [here](https://github.com/kataras/iris/blob/master/_examples/subdomains/redirect/main.go)
- 缓存中间件携带 `304` 状态码, 缓存期间的请求，服务器只响应状态, 相关示例 [here](https://github.com/kataras/iris/blob/master/_examples/cache/client-side/main.go)
- `websocket/Connection#IsJoined(roomName string)` 新增方法，检查用户是否加入房间。 未加入的连接不能发送消息，此检查是可选的.

详情：

- 更新上游 vendor/golang/crypto 包到最新版本, 我们总是跟进依赖关系的任何修复和有意义的更新.
- [改进：不在gzip响应的WriteString和Writef上强制设置内容类型（如果已经存在的话）](https://github.com/kataras/iris/commit/af79aad11932f1a4fcbf7ebe28274b96675d0000)
- [新增：websocket/Connection#IsJoined](https://github.com/kataras/iris/commit/cb9e30948c8f1dd099f5168218d110765989992e)
- [修复：#897](https://github.com/kataras/iris/commit/21cb572b638e82711910745cfae3c52d836f01f9)
- [新增：context#StatusCodeNotSuccessful 变量用来定制 rfc2616-sec10](https://github.com/kataras/iris/commit/c56b7a3f04d953a264dfff15dadd2b4407d62a6f)
- [修复：示例 routing/dynamic-path/main.go#L101](https://github.com/kataras/iris/commit/0fbf1d45f7893cb1393759b7362444f3d381d182)
- [新增：缓存中间件 `iris.Cache304`](https://github.com/kataras/iris/commit/1722355870174cecbc12f7beff8514b058b3b912)
- [修复：示例  csrf](https://github.com/kataras/iris/commit/a39e3d7d6cf528e51e6c7e32a884a8d9f2fadc0b)
- [取消：Configuration.RemoteAddrHeaders 默认值](https://github.com/kataras/iris/commit/47108dc5a147a8b23de61bef86fe9327f0781396)
- [新增：vscode 扩展链接和徽章](https://github.com/kataras/iris/commit/6f594c0a7c641cc98bd683163fffbf5fa5fc8de6)
- [新增：`app.View` 示例 用于解析和编写HTTP之外的模板（类似于上下文＃视图)](_examples/view/write-to)
- [新增：支持多级域名跳转](https://github.com/kataras/iris/commit/12d7df113e611a75088c2a72774dab749d2c7685).

# 2018 1月16号 | v10.0.2 版本更新

## 安全更新 | `iris.AutoTLS`

**建议升级**, 包含几天前修复了 letsencrypt.org 禁用 tls-sni 的问题，这导致几乎每个启用了 https 的 golang 服务器都无法正常工作，因此支持添加了 http-01 类型。 现在服务器会尝试所有可用的 letsencrypt 类型。

更多相关资讯:

- https://letsencrypt.status.io/pages/incident/55957a99e800baa4470002da/5a55777ed9a9c1024c00b241
- https://github.com/golang/crypto/commit/13931e22f9e72ea58bb73048bc752b48c6d4d4ac

# 2018 1月15号 | v10.0.1 版本更新

该版本暂未发现重大问题，但如果你使用 [cache](cache) 包的话，这里有些更新或许正好解决某些问题。

- 修复缓存在同一控制器多个方法中，返回相同内容问题 https://github.com/kataras/iris/pull/852, 问题报告：https://github.com/kataras/iris/issues/850
- 问题修正 https://github.com/kataras/iris/pull/862
- 当 `view#Engine##Reload` 为 true，`ExecuteWriter -> Load` 不能同时使用问题，相关问题 ：https://github.com/kataras/iris/issues/872
- 由Iris提供支持的开源项目的徽章, 学习如何将徽章添加到您的开源项目中 [FAQ.md](FAQ.md)
- 上游更新 `golang/crypto` 修正 [tls-sni challenge disabled](https://letsencrypt.status.io/pages/incident/55957a99e800baa4470002da/5a55777ed9a9c1024c00b241) https://github.com/golang/crypto/commit/13931e22f9e72ea58bb73048bc752b48c6d4d4ac (**关系到 iris.AutoTLS**)

## 新增捐助

1. https://opencollective.com/cetin-basoz

## 新增翻译

1. 中文版 README_ZH.md and HISTORY_ZH.md 由 @Zeno-Code 翻译 https://github.com/kataras/iris/pull/858
2. 俄语版 README_RU.md 由 @merrydii 翻译 https://github.com/kataras/iris/pull/857
3. 希腊版 README_GR.md and HISTORY_GR.md https://github.com/kataras/iris/commit/8c4e17c2a5433c36c148a51a945c4dc35fbe502a#diff-74b06c740d860f847e7b577ad58ddde0 and https://github.com/kataras/iris/commit/bb5a81c540b34eaf5c6c8e993f644a0e66a78fb8

## 新增示例

1. [MVC - Register Middleware](_examples/mvc/middleware)

## 新增文章

1. [A Todo MVC Application using Iris and Vue.js](https://hackernoon.com/a-todo-mvc-application-using-iris-and-vue-js-5019ff870064)
2. [A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](bit.ly/2lmKaAZ)

# 2018 元旦 | v10.0.0 版本发布

我们必须感谢 [Mrs. Diana](https://www.instagram.com/merry.dii/) 帮我们绘制的漂亮 [logo](https://iris-go.com/images/icon.svg)!

如果有设计相关的需求，你可以[发邮件](mailto:Kovalenkodiana8@gmail.com)给他，或者通过 [instagram](https://www.instagram.com/merry.dii/) 给他发信息。

<p align="center">
<img width="145px" src="https://iris-go.com/images/icon.svg?v=a" />
</p>

在这个版本中，有许多内部优化改进，但只有两个重大变更和新增一个叫做 **hero** 的特性。

> 新版本有 75 + 的变更提交, 如果你需要升级 Iris 请仔细阅读本文档。 为什么版本 9 跳过了? 你猜...

## Hero 特性

新增包 [hero](hero) 可以绑定处理任何依赖 `handlers` 的对象或函数。Hero funcs 可以返回任何类型的值，并发送给客户端。

> 之前的绑定没有编辑器的支持, 新包 `hero` 为 Iris 带来真正的安全绑定。 Iris 会在服务器运行之前计算所有内容，所以它执行速度高，接近于原生性能。

下面你会看到我们为你准备的一些截图，以便于理解:

### 1. 路径参数 - 构建依赖

![](https://github.com/kataras/explore/raw/master/iris/hero/hero-1-monokai.png)

### 2. 服务 - 静态依赖

![](https://github.com/kataras/explore/raw/master/iris/hero/hero-2-monokai.png)

### 3. 请求之前 - 动态依赖

![](https://github.com/kataras/explore/raw/master/iris/hero/hero-3-monokai.png)

`hero funcs` 非常容易理解，当你用过之后 **在也回不去了**.

示例：

- [基本用法](_examples/hero/basic/main.go)
- [使用概览](_examples/hero/overview)

## MVC

如果要使用 `mvc` ，必须先理解 `hero` 包，因为`mvc`在内部使用`hero`作为路由控制器的方法，同样的规则也适用于你的控制器的方法。

With this version you can register **any controller's methods as routes manually**, you can **get a route based on a method name and change its `Name` (useful for reverse routing inside templates)**, you can use any **dependencies** registered from `hero.Register` or `mvc.New(iris.Party).Register` per mvc application or per-controller, **you can still use `BeginRequest` and `EndRequest`**, you can catch **`BeforeActivation(b mvc.BeforeActivation)` to add dependencies per controller and `AfterActivation(a mvc.AfterActivation)` to make any post-validations**, **singleton controllers when no dynamic dependencies are used**, **Websocket controller, as simple as a `websocket.Connection` dependency** and more...

示例:

**如果你之前使用过 MVC ，请仔细阅读：MVC 包含一些破坏性的改进，但新的方式可以做更多，会让程序执行更快**

**请阅读我们为你准备的示例**

如果你现在需要升级，请对比新旧版本示例的不同，便于理解。

| NEW | OLD |
| -----------|-------------|
| [Hello world](_examples/mvc/hello-world/main.go) | [OLD Hello world](https://github.com/kataras/iris/blob/v8/_examples/mvc/hello-world/main.go) |
| [Session Controller](_examples/mvc/session-controller/main.go) | [OLD Session Controller](https://github.com/kataras/iris/blob/v8/_examples/mvc/session-controller/main.go) |
| [Overview - Plus Repository and Service layers](_examples/mvc/overview) | [OLD Overview - Plus Repository and Service layers](https://github.com/kataras/iris/tree/v8/_examples/mvc/overview) |
| [Login showcase - Plus Repository and Service layers](_examples/mvc/login) | [OLD Login showcase - Plus Repository and Service layers](https://github.com/kataras/iris/tree/v8/_examples/mvc/login) |
| [Singleton](_examples/mvc/singleton) |  **新增** |
| [Websocket Controller](_examples/mvc/websocket) |  **新增** |
| [Vue.js Todo MVC](_examples/tutorial/vuejs-todo-mvc) |  **新增** |

## context#PostMaxMemory

移除旧版本的常量 `context.DefaultMaxMemory` 替换为配置 `WithPostMaxMemory` 方法.

```go
// WithPostMaxMemory 设置客户端向服务器 post 提交数据的最大值
// 他不同于 request body 的值大小，如果有相关需求请使用
// `context#SetMaxRequestBodySize` 或者 `iris#LimitRequestBodySize`
//
// 默认值为 32MB 或者 32 << 20
func WithPostMaxMemory(limit int64) Configurator
```

如果你使用老版本的常量，你需要更改一行代码.

使用方式：

```go
import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // [...]

    app.Run(iris.Addr(":8080"), iris.WithPostMaxMemory(10 << 20))
}
```

## context#UploadFormFiles

新方法可以多文件上传, 应用于常见的上传操作, 它是一个非常有用的函数。

```go
// UploadFormFiles 将所有接收到的文件从客户端上传到系统物理位置 destDirectory。
//
// The second optional argument "before" gives caller the chance to
// modify the *miltipart.FileHeader before saving to the disk,
// it can be used to change a file's name based on the current request,
// all FileHeader's options can be changed. You can ignore it if
// you don't need to use this capability before saving a file to the disk.
//
// Note that it doesn't check if request body streamed.
//
// Returns the copied length as int64 and
// a not nil error if at least one new file
// can't be created due to the operating system's permissions or
// http.ErrMissingFile if no file received.
//
// If you want to receive & accept files and manage them manually you can use the `context#FormFile`
// instead and create a copy function that suits your needs, the below is for generic usage.
//
// The default form's memory maximum size is 32MB, it can be changed by the
//  `iris#WithPostMaxMemory` configurator at main configuration passed on `app.Run`'s second argument.
//
// See `FormFile` to a more controlled to receive a file.
func (ctx *context) UploadFormFiles(
        destDirectory string,
        before ...func(string, string),
    ) (int64, error)
```

这里是相关示例 [here](_examples/http_request/upload-files/main.go).

## context#View

这里有个小更新，增加可选的第二个参数，用来绑定模版变量。提示：这种绑定方式，会忽略其他变量的绑定。
如果要忽略其他模版变量，之前是在 `ViewData` 上绑定一个空字符串，现在可以直接通过 View 方法添加。

```go
func(ctx iris.Context) {
    ctx.ViewData("", myItem{Name: "iris" })
    ctx.View("item.html")
}
```

等同于：

```go
func(ctx iris.Context) {
    ctx.View("item.html", myItem{Name: "iris" })
}
```

```html
html 模版中调用: {{.Name}}
```

## context#YAML

新增 `context#YAML` 函数, 解析结构体到 yaml。

```go
//使用 yaml 包的 Marshal 的方法解析，并发送到客户端。
func YAML(v interface{}) (int, error)
```

## Session#GetString

`sessions/session#GetString` 可以获取 session 的变量值（可以是 integer 类型），就像内存缓存、Context 上下文储存的值。
