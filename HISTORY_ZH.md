# 更新记录 <a href="HISTORY.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /></a> <a href="HISTORY_ID.md"> <img width="20px" src="https://iris-go.com/images/flag-indonesia.svg?v=10" /></a> <a href="HISTORY_GR.md"> <img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a>

### 想得到免费即时的支持?

    https://github.com/kataras/iris/issues
    https://chat.iris-go.com

### 获取历史版本?

    https://github.com/kataras/iris/releases

### 我是否应该升级 Iris?

如果没有必要，不会强制升级。如果你已经准备好了，可以随时升级。

> Iris 使用 Golang 的 [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) 特性, 避免依赖包的更改带来影响。

**如何升级**: 打开命令行执行以下命令: `go get -u github.com/kataras/iris` 或者等待自动更新。

# Fr, 11 January 2019 | v11.1.1

This history entry is not translated yet to the Chinese language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#fr-11-january-2019--v1111) instead.

# Su, 18 November 2018 | v11.1.0

This history entry is not translated yet to the Chinese language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#su-18-november-2018--v1110) instead.

# Fr, 09 November 2018 | v11.0.4

This history entry is not translated yet to the Chinese language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#fr-09-november-2018--v1104) instead.

# Tu, 30 October 2018 | v11.0.2

This history entry is not translated yet to the Chinese language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#tu-30-october-2018--v1102) instead.

# Su, 28 October 2018 | v11.0.1

This history entry is not translated yet to the Chinese language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#su-28-october-2018--v1101) instead.

# Su, 21 October 2018 | v11.0.0

This history entry is not translated yet to the Chinese language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#su-21-october-2018--v1100) instead.

# Sat, 11 August 2018 | v10.7.0

This history entry is not translated yet to the Chinese language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#sat-11-august-2018--v1070) instead.

# Tu, 05 June 2018 | v10.6.6

This history entry is not translated yet to the Chinese language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#tu-05-june-2018--v1066) instead.

# Mo, 21 May 2018 | v10.6.5

This history entry is not translated yet to the Chinese language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#mo-21-may-2018--v1065) instead.

# We, 09 May 2018 | v10.6.4

- [fix issue 995](https://github.com/kataras/iris/commit/62457279f41a1f157869a19ef35fb5198694fddb)
- [fix issue 996](https://github.com/kataras/iris/commit/a11bb5619ab6b007dce15da9984a78d88cd38956)

# We, 02 May 2018 | v10.6.3

This history entry is not translated yet to the Chinese language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#we-02-may-2018--v1063) instead.

# Tu, 01 May 2018 | v10.6.2

This history entry is not translated yet to the Chinese language yet, please refer to the english version of the [HISTORY entry](https://github.com/kataras/iris/blob/master/HISTORY.md#tu-01-may-2018--v1062) instead.

# 2018 4月25日 | v10.6.1 版本更新

- 用最新版 BoltDB 重新实现 session (`sessiondb`) 存储：[/sessions/sessiondb/boltdb/database.go](sessions/sessiondb/boltdb/database.go), 相关示例 [/_examples/sessions/database/boltdb/main.go](_examples/sessions/database/boltdb/main.go).
- 修正 一个小问题 on [Badger sessiondb example](_examples/sessions/database/badger/main.go). `sessions.Config { Expires }` 字段由 `2 *time.Second` 调整为 `45 *time.Minute` .
- badger sessiondb 其他小改进.

# 2018 4月22日 | v10.6.0 版本更新

- 修正 重定向问题 由 @wozz 提交: https://github.com/kataras/iris/pull/972.
- 修正 无法销毁子域名 session 问题 由 @Chengyumeng 提交: https://github.com/kataras/iris/pull/964.
- 添加 `OnDestroy(sid string)` 当 session 销毁时注册监听器 相关细节: https://github.com/kataras/iris/commit/d17d7fecbe4937476d00af7fda1c138c1ac6f34d.
- sessions 现在与注册数据库完全同步。 这涉及到很多内部改动，但 **这不影响你当前项目代码**. 我们只保留了 `badger` 和 `redis` 作为底部支持。 相关细节: https://github.com/kataras/iris/commit/f2c3a5f0cef62099fd4d77c5ccb14f654ddbfb5c 

# 2018 3月 24日 | v10.5.0 版本更新

### 新增

新增 缓存中间件客户端，更快的静态文件服务器. 详情 [点击](https://github.com/kataras/iris/pull/935).

### 破坏式更新

改变 `Value<T>Default(<T>, error)` 为 `Value<T>Default(key, defaultValue) <T>`  如同 `ctx.PostValueIntDefault` 或 `ctx.Values().GetIntDefault` 或 `sessions/session#GetIntDefault` 或 `context#URLParamIntDefault`.
由 @jefurry 提出 https://github.com/kataras/iris/issues/937.

#### 如何升级现有代码

只需要移除第二个返回值即可.

示例:  [_examples/mvc/basic/main.go line 100](_examples/mvc/basic/main.go#L100)  `count,_ := c.Session.GetIntDefault("count", 1)` **变更为:** `count := c.Session.GetIntDefault("count", 1)`.

> 请记住，如果您无法升级，那么就不要这样做，我们在此版本中没有任何安全修复程序，但在某些时候建议您最好进行升级，我们总是会添加您喜欢的新功能！


# 2018 3月14日 | v10.4.0 版本更新

- 修正 `APIBuilder, Party#StaticWeb` 和 `APIBuilder, Party#StaticEmbedded` 子分组内的前缀错误
- 保留 `iris, core/router#StaticEmbeddedHandler` 并移除 `core/router/APIBuilder#StaticEmbeddedHandler`,  (`Handler` 后缀) 这是全局性的，与 `Party` `APIBuilder` 无关。
- 修正 路径 `{}` 中的路径清理 (我们已经在 [解释器](macro/interpreter) 级别转义了这些字符， 但是一些符号仍然被更高级别的API构建器删除) , 例如 `\\` 字符串的宏函数正则表达式内容 [927](https://github.com/kataras/iris/issues/927) by [commit e85b113476eeefffbc7823297cc63cd152ebddfd](https://github.com/kataras/iris/commit/e85b113476eeefffbc7823297cc63cd152ebddfd)
- 同步 `golang.org/x/sys/unix`

## 重要变更

我们使用新工具将静态文件的速度提高了8倍, <https://github.com/kataras/bindata> 这是 go-bindata 的一个分支，对我们来说，一些不必要的东西被移除了，并且包含一些提高性能的补充。

## Reqs/sec 使用 [shuLhan/go-bindata](https://github.com/shuLhan/go-bindata) 和 备选方案对比

![go-bindata](https://github.com/kataras/bindata/raw/master/go-bindata-benchmark.png)

## Reqs/sec 使用 [kataras/bindata](https://github.com/kataras/bindata)

![bindata](https://github.com/kataras/bindata/raw/master/bindata-benchmark.png)

**新增** 方法 `Party#StaticEmbeddedGzip` 与 `Party#StaticEmbedded` 参数相同. 不同处在于 **新增** `StaticEmbeddedGzip` 从 `bindata` 接收 `GzipAsset` 和 `GzipAssetNames` (go get -u github.com/kataras/bindata/cmd/bindata).

你可以在同个文件夹里同时使用 `bindata` 和 `go-bindata` 工具, 第一个用于嵌入静态文件 (javascript, css, ...) 第二个用于静态编译模板!

完整示例: [_examples/file-server/embedding-gziped-files-into-app/main.go](_examples/file-server/embedding-gziped-files-into-app/main.go).


# 2018 3月10号 | v10.3.0 版本更新

- 只有一项 API 更改 [Application/Context/Router#RouteExists](https://godoc.org/github.com/kataras/iris/core/router#Router.RouteExists), 将 `Context` 作为第一参数，而不是最后一个。

- 修正 cors 中间件 https://github.com/iris-contrib/middleware/commit/048e2be034ed172c6754448b8a54a9c55debad46, 相关问题: https://github.com/kataras/iris/issues/922 (目前仍在等待验证).

- 添加 `Context#NextOr` 和 `Context#NextOrNotFound` 方法

```go
// NextOr 检查程序链上是否有下一个处理程序，如果是，则执行它
// 否则根据给定的处理程序设置分配给 Context 程序链，并且执行第一个控制器。
//
// 如果下一个处理器存在并执行，则返回true，否则返回false
//
// 请注意，如果没有找到下一个处理程序并且处理程序缺失，
// 会发送 (404) 状态码到客户端，并停止执行。
NextOr(handlers ...Handler) bool
// NextOrNotFound 检查程序链上是否存在下一个处理程序，如果有则执行
// 其他情况会发送 404 状态码，并停止执行。
//
// 如果下一个控制器存在并执行，返回 true , 其他情况 false.
NextOrNotFound() bool
```

- 新增方法 `Party#AllowMethods` 如果在 `Handle, Get, Post...` 之前调用，则会将路由克隆到该方法.

- 修复 POST 请求尾部斜杠重定向问题: https://github.com/kataras/iris/issues/921 https://github.com/kataras/iris/commit/dc589d9135295b4d080a9a91e942aacbfe5d56c5

- 新增示例 通过 `iris#UnmarshalerFunc` 自定义解码， 新增 `context#ReadXML` 使用示例, [相关示例](https://github.com/kataras/iris/tree/master/_examples#how-to-read-from-contextrequest-httprequest)via https://github.com/kataras/iris/commit/78cd8e5f677fe3ff2c863c5bea7d1c161bf4c31e.

- 新增自定义路由宏功能示例, 相关讨论 https://github.com/kataras/iris/issues/918, [示例代码](https://github.com/kataras/iris/blob/master/_examples/routing/dynamic-path/main.go#L144-L158), https://github.com/kataras/iris/commit/a7690c71927cbf3aa876592fab94f04cada91b72

- 为 `Pongo` 新增 `AsValue()` 和 `AsSaveValue()` @neenar https://github.com/kataras/iris/pull/913

- 删除 `context#UnmarshalBody` 上不必要的反射 https://github.com/kataras/iris/commit/4b9e41458b62035ea4933789c0a132c3ef2a90cc

# 2018 2月15号 | v10.2.1 版本更新

修正 子域名 (subdomain) 的 `StaticEmbedded` 和 `StaticWeb` 不存在错误, 由 [@speedwheel](https://github.com/speedwheel) 通过 [facebook page's chat](https://facebook.com/iris.framework) 反馈。

# 2018 2月8号 | v10.2.0 版本更新

新的小版本， 因为它包含一个 **破坏性变动** 和一个新功能 `Party#Reset`

### Party#Done 特性变动 和 新增 Party#DoneGlobal 介绍

正如 @likakuli 指出的那样 https://github.com/kataras/iris/issues/901, 以前 `Done` 注册的处理器，在全局范围内会替代子处理器，因为在引入 `UseGlobal` 这概念之前，缺少稳定性. 现在是时候了, 新的 `Done` 应该在相关的路由之前调用， **新增** `DoneGlobal` 之前的`Done` 使用相同; 顺序无关紧要，他只是结束处理附加到当前的注册程序, 全局性的 (所有子域名，分组).

[routing/writing-a-middleware](_examples/routing/writing-a-middleware) 路由中间件示例更新, 列举了使用方式变化, 如果之前使用过 Iris ,并熟悉内置函数方法名称，请区分 `DoneGlobal` 和 `Done` 的不同.

### Party#Reset

新增 `Party#Reset()` 函数，以便重置上级分组通过 `Use` 和 `Done` 注册的处理方法, 没有什么特别之处，它只是清除当前分组实例的 `middleware` 和 `doneHandlers`，详情参见 `core/router#APIBuilder`.

### 更新方法

只需要将现有的 `.Done` 替换为 `.DoneGlobal` 就可以了。

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
