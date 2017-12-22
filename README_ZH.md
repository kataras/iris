# [![Logo created by @santoshanand](logo_white_35_24.png)](https://iris-go.com) Iris

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)[![Backers on Open Collective](https://opencollective.com/iris/backers/badge.svg?style=flat-square)](#backers)[![Sponsors on Open Collective](https://opencollective.com/iris/sponsors/badge.svg?style=flat-square)](#sponsors)[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris)[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)[![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)[![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples)[![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris)[![CLA assistant](https://cla-assistant.io/readme/badge/kataras/iris?style=flat-square)](https://cla-assistant.io/kataras/iris)

<a target='_blank' rel='nofollow' href='https://app.codesponsor.io/link/Qw6E1MTHvUJW6BtwUUf9qwsy/kataras/iris'>
  <img alt='Sponsor' width='888' height='68' src='https://app.codesponsor.io/embed/Qw6E1MTHvUJW6BtwUUf9qwsy/kataras/iris.svg' />
</a>


Iris是一个超快、简单并且高效的Go语言Web开发框架。

Iris功能很强大，使用又很简单，它将会是你下一个网站、API服务或者分布式应用基础框架的不二之选。

看看[别人是如何评价Iris](https://www.youtube.com/watch?v=jGx0LkuUs4A)，同时欢迎各位[成为Iris星探](https://github.com/kataras/iris/stargazers)，或者关注[Iris facebook主页](https://facebook.com/iris.framework)。

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new-gray.png)](_benchmarks)

<details>
<summary>上图是第三发机构发布的REST Web框架的基准测试</summary>
  
![Comparison with other frameworks](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

更新于: [2017年9月29，星期五](_benchmarks)_
</details>

## 前言 ♥️

在发现Iris之前，我想你一定也看过其它Go Web开发框架。也许你已经摩拳擦掌并马上要使用了，但我会很遗憾的告诉你，将来你还是会使用Iris的。不仅仅因为Iris性能卓越和使用简单，更重要的是Iris独树一帜，他可以让你成为真正的极客界摇滚明星。

不管你是想开发微服务或者大型Web应用，Iris都能满足你的需求。Iris可能是你在网上能找到最好的Web后台开发软件之一了。

Iris现在已经到第8版了，但是我们从未停止开发。有很多非常棒的功能已经提上开发日程了，而且我们非常乐意加入很多有创意的想法。

如果你想用CDN加速，我推荐用[KeyCDN](https://www.keycdn.com/)，因为KeyCDN简单、速度快而且稳定。

我们用[微软](https://www.microsoft.com)开发的[Visual Studio Code](https://code.visualstudio.com/)来开发Golang应用。

如果你之前使用[nodejs](https://nodejs.org)做开发，恭喜你，Iris使用基本和[expressjs](https://github.com/expressjs/express)一样。

## 内容列表

* [安装](#installation)
* [最近更新](https://github.com/kataras/iris/blob/master/HISTORY.md#tu-07-november-2017--v857)
* [快速入门](#getting-started)
* [进阶](_examples/)
    * [MVC (模型 视图 控制器)](_examples/#mvc) **NEW**
    * [结构](_examples/#structuring) **NEW**
    * [HTTP 监听](_examples/#http-listening)
    * [配置](_examples/#configuration)
    * [路由，分组，动态参数，“宏定义”已经自定义Context](_examples/#routing-grouping-dynamic-path-parameters-macros-and-custom-context)
    * [子域名处理](_examples/#subdomains)
    * [`http.Handler/HandlerFunc` 使用](_examples/#convert-httphandlerhandlerfunc)
    * [视图处理](_examples/#view)
    * [认证](_examples/#authentication)
    * [文件服务器](_examples/#file-server)
    * [如何从`context.Request() *http.Request` 读数据](_examples/#how-to-read-from-contextrequest-httprequest)
    * [如何给`context.ResponseWriter() http.ResponseWriter`写数据](_examples/#how-to-write-to-contextresponsewriter-httpresponsewriter)
    * [测试](_examples/#testing)	
    * [缓存](_examples/#caching)
    * [会话](_examples/#sessions)
    * [Websockets](_examples/#websockets)
    * [其它杂项](_examples/#miscellaneous)
    * [将"Parrot"项目转换为Iris实现](https://github.com/iris-contrib/parrot)
    * [Iris和react/hot reloadable/redux/css-modules配合使用](https://github.com/kataras/iris-starter-kit)
    * [Typescript自动化操作工具](typescript/#table-of-contents)
    * [指南: 用Iris和Bolt实现短连接服务](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)
    * [指南: 如何统计在线访问人数](_examples/tutorial/online-visitors)
    * [指南: Caddy](_examples/tutorial/caddy)
    * [指南: 如何用DropzoneJS上传文件](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
    * [指南: Iris+MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)
* [中间件](middleware/)
* [Docker例子](https://github.com/iris-contrib/cloud-native-go)
* [贡献](CONTRIBUTING.md)
* [常见问题](FAQ.md)
* [更新计划?](#now-you-are-ready-to-move-to-the-next-step-and-get-closer-to-becoming-a-pro-gopher)
* [开发者](#people)

## 安装

仅仅依赖[Go语言](https://golang.org/dl/)

```sh
$ go get -u github.com/kataras/iris
```
Iris使用[vendor](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) 包依赖管理方式。vendor包管理的方式可以有效处理包依赖更新问题

## 入门

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // 从"./views"目录加载HTML模板
    // 模板解析html后缀文件
    // 此方式是用`html/template`标准包(Iris的模板引擎)
    app.RegisterView(iris.HTML("./views", ".html"))

    // HTTP方法： GET
    // 路径：     http://localhost:8080
    app.Get("/", func(ctx iris.Context) {
        // {{.message}} 和 "Hello world!" 字串绑定
        ctx.ViewData("message", "Hello world!")
        // 映射HTML模板文件路径 ./views/hello.html
        ctx.View("hello.html")
    })

    // HTTP方法:    GET
    // 路径:  http://localhost:8080/user/42
    //
    // 想在路径中用正则吗？So easy!
    // 如下所示
    // app.Get("/user/{id:string regexp(^[0-9]+$)}")
    app.Get("/user/{id:long}", func(ctx iris.Context) {
        userID, _ := ctx.Params().GetInt64("id")
        ctx.Writef("User ID: %d", userID)
    })

    // 绑定端口并启动服务.
    app.Run(iris.Addr(":8080"))
}
```

> 想要了解更多关于路径参数配置，戳[这里](https://github.com/kataras/iris/blob/master/_examples/routing/dynamic-path/main.go#L31).

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
$ go run main.go
> 在这里监听服务: http://localhost:8080
> 应用已经启动按键 CTRL+C 停止服务
```

> 想要实现当代码改变后自动重启应用吗？那就装个[rizla](https://github.com/kataras/rizla)工具，启动go文件用 `rizla main.go` 来代替 `go run main.go`。

Iris的一些开发约定可以看看这里[_examples/structuring](_examples/#structuring)。


## 现在你已经准备好进入下一阶段，又向专家级gopher迈进一步了

恭喜你看到这里了，我们为你准备了更高水平的内容，向真正的专家级gopher进军吧😃

> 准备好咖啡，尽情享受吧！

* [Iris框架+MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)
* [用DropzoneJS 和 Go来构建表单文件上传](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
* [用DropzoneJS 和 Go来呈现服务器上的问题](https://hackernoon.com/how-to-display-existing-files-on-server-using-dropzonejs-and-go-53e24b57ba19)
* [Iris模块化Web开发框架](https://medium.com/@corebreaker/iris-web-cd684b4685c7)
* [按照 HTTP 性能来比较Go 和 .NET Core](https://medium.com/@kataras/go-vs-net-core-in-terms-of-http-performance-7535a61b67b8)
* [按照 HTTP 性能来比较Go 和 .NET Core Kestrel](https://hackernoon.com/iris-go-vs-net-core-kestrel-in-terms-of-http-performance-806195dc93d5)
* [在Android设备上搭建Web服务器](https://twitter.com/ThePracticalDev/status/892022594031017988)
* [在hasura上部署Iris应用](https://medium.com/@HasuraHQ/deploy-an-iris-golang-app-with-backend-apis-in-minutes-25a559bf530b)
* [用Iris 和 Bolt实现短连接服务](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)

## 作者

 Iris的作者是[@kataras](https://github.com/kataras), 你可以通过以下方式来了解作者：

* [Medium](https://medium.com/@kataras)
* [Twitter](https://twitter.com/makismaropoulos)
* [Dev.to](https://dev.to/@kataras)
* [Facebook](https://facebook.com/iris.framework)
* [Mail](mailto:kataras2006@hotmail.com?subject=Iris%20I%20need%20some%20help%20please)

[作者](AUTHORS)

[贡献者列表](https://github.com/kataras/iris/graphs/contributors)

你可以通过[PayPal](https://www.paypal.me/kataras) 或 [BTC](https://iris-go.com/v8/donate)来捐赠这个项目，这样可以促进开发者们创造更棒、更优秀的Iris。

[如何贡献代码](CONTRIBUTING.md)

### 我们期待你能帮助我们翻译Iris文档

Iris需要你的帮助，帮助我们翻译[README](README.md)和https://iris-go.com ，同时你也会得到奖励的。

你可以在这里https://github.com/kataras/iris/issues/796 看到详细的有关翻译的信息


### Iris 用户体验反馈 | 2017年10月3号 

**请放心** Iris用户体验反馈就是一些简单的表单提交，**2分钟**就能搞定。

这些表单里有些问题是为了更好的了解你，了解你可以让我们更好的为你服务。

https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link

## 贡献者列表

非常感谢所有对Iris的贡献者，没有你们就没有Iris [贡献者](CONTRIBUTING.md)
<a href="graphs/contributors"><img src="https://opencollective.com/iris/contributors.svg?width=890" /></a>

## 资助者

万分感谢所有的资助者🙏 [成为资助者](https://opencollective.com/iris#backer)

<a href="https://opencollective.com/iris#backers" target="_blank"><img src="https://opencollective.com/iris/backers.svg?width=890"></a>

## 赞助商

资助Iris，你将是Iris的赞助商，你的logo将会出现在下面的列表中，[成为赞助商](https://opencollective.com/iris#sponsor)

<a href="https://opencollective.com/iris/sponsor/0/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/0/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/1/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/1/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/2/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/2/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/3/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/3/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/4/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/4/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/5/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/5/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/6/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/6/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/7/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/7/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/8/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/8/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/9/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/9/avatar.svg"></a>

## 开源许可证

Iris使用3-Clause BSD [许可证](LICENSE)开源许可 。Iris绝对是100%开源的。
对这个许可有任何疑问请[联系我们](mailto:kataras2006@hotmail.com?subject=Iris%20License)
