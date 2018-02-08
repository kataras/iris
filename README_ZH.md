# Iris Web Framework <a href="README.md"> <img width="20px" src="https://iris-go.com/images/flag-unitedkingdom.svg?v=10" /></a> <a href="README_RU.md"><img width="20px" src="https://iris-go.com/images/flag-russia.svg?v=10" /></a> <a href="README_GR.md"><img width="20px" src="https://iris-go.com/images/flag-greece.svg?v=10" /></a>

<img align="right" width="169px" src="https://iris-go.com/images/icon.svg?v=a" title="logo created by @merry.dii" />

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)<!-- [![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)--> [![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris) [![vscode-iris](https://img.shields.io/badge/ext%20-vscode-0c77e3.svg?style=flat-square)](https://marketplace.visualstudio.com/items?itemName=kataras2006.iris)<!--[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)--> [![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris) [![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](_examples/) [![release](https://img.shields.io/badge/release%20-v10.2-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/releases)

Iris 是一款超快、简洁高效的 Go 语言 Web开发框架。

Iris 功能强大、使用简单，它将会是你下一个网站、API 服务或者分布式应用基础框架的不二之选。

总之，是一款与 express.js 旗鼓相当的 Go 语言框架。

看看[别人是如何评价 Iris](#support)，同时欢迎各位点亮 Iris [Star](https://github.com/kataras/iris/stargazers)，或者关注 [Iris facebook 主页](https://facebook.com/iris.framework)。

## 支持者

感谢所有的支持者! 🙏 [支持我们](https://opencollective.com/iris#backer)

<a href="https://opencollective.com/iris#backers" target="_blank"><img src="https://opencollective.com/iris/backers.svg?width=890"></a>

```sh
$ cat example.go
```

```go
package main

import "github.com/kataras/iris"

func main() {
    app := iris.New()
    // 从 "./views" 目录加载HTML模板
    // 模板解析 html 后缀文件
    // 此方式使用 `html/template` 标准包 (Iris 的模板引擎)
    app.RegisterView(iris.HTML("./views", ".html"))

    // 方法：GET
    // 路径：http://localhost:8080
    app.Get("/", func(ctx iris.Context) {
        // {{.message}} 和 "Hello world!" 字符串变量绑定
        ctx.ViewData("message", "Hello world!")
        // 映射 HTML 模板文件路径 ./views/hello.html
        ctx.View("hello.html")
    })

    //方法：GET
    //路径：http://localhost:8080/user/42
    //
    // 使用正则表达式必须设置参数类型为 string
    // app.Get("/user/{id:string regexp(^[0-9]+$)}")
    app.Get("/user/{id:long}", func(ctx iris.Context) {
        userID, _ := ctx.Params().GetInt64("id")
        ctx.Writef("User ID: %d", userID)
    })

    // 绑定端口并启动服务.
    app.Run(iris.Addr(":8080"))
}
```

> 想要了解更多关于路径参数配置，戳[这里](_examples/routing/dynamic-path/main.go#L31)。

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

## 安装

请确保安装 [Go Programming Language](https://golang.org/dl/)

```sh
$ go get -u github.com/kataras/iris
```

Iris 使用 [vendor](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) 包依赖管理方式。vendor 包管理的方式可以有效处理包依赖更新问题

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new-gray.png)](_benchmarks/README_UNIX.md)

_更新于: [2017年11月21日星期二](_benchmarks/README_UNIX.md)_

<details>
<summary>来自第三方的其他网络框架的基准测试</summary>

![Comparison with other frameworks](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

</details>

## 支持

- [更新记录](HISTORY_ZH.md#th-08-february-2018--v1020) 是您最好的朋友，它包含有关最新功能和更改的信息
- 你碰巧找到了一个错误？ 请提交 [github issues](https://github.com/kataras/iris/issues)
- 您是否有任何疑问或需要与有经验的人士交谈以实时解决问题？ [加入我们的聊天](https://chat.iris-go.com)
- [点击这里完成我们基于表单的用户体验报告](https://docs.google.com/forms/d/e/1FAIpQLSdCxZXPANg_xHWil4kVAdhmh7EBBHQZ_4_xSZVDL-oCC_z5pA/viewform?usp=sf_link) 
- 你喜欢这个框架吗？ Twitter 上关于 Iris 的评价:

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


[如何贡献代码](CONTRIBUTING.md)

[贡献者列表](https://github.com/kataras/iris/graphs/contributors)

## 学习

首先，从 Web 框架开始的最正确的方法是学习 Golang 标准库 [net/http](https://golang.org/pkg/net/http/ "net/http") 的基础知识，如果您的 web 应用程序是一个非常简单的个人项目，没有性能和可维护性要求，您可能只需使用标准库即可。 之后，遵循以下指导原则：

- 浏览 **100+** **[例子](_examples)** 和 我们提供的 [一些入门经验](#iris-starter-kits)
- 通过 [godocs](https://godoc.org/github.com/kataras/iris) 阅读细节
- 准备一杯咖啡或茶，无论你喜欢什么，并阅读我们为你推荐的 [一些文章](#articles)

### Iris 入门

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

1. [A basic web app built in Iris for Go](https://github.com/gauravtiwari/go_iris_app)
2. [A mini social-network created with the awesome Iris💖💖](https://github.com/iris-contrib/Iris-Mini-Social-Network)
3. [Iris isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/iris-contrib/iris-starter-kit)
4. [Demo project with react using typescript and Iris](https://github.com/ionutvilie/react-ts)
5. [Self-hosted Localization Management Platform built with Iris and Angular](https://github.com/iris-contrib/parrot)
6. [Iris + Docker and Kubernetes](https://github.com/iris-contrib/cloud-native-go)
7. [Quickstart for Iris with Nanobox](https://guides.nanobox.io/golang/iris/from-scratch)
8. [A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](https://hasura.io/hub/project/hasura/hello-golang-iris)

> 如果你有类似的使用经验吗 [请提交给我们](https://github.com/kataras/iris/pulls)!

### 中间件

Iris 拥有大量的中间件 [[1]](middleware/)[[2]](https://github.com/iris-contrib/middleware) 供您的 Web 应用程序使用。 不过，您并不局限于此，您可以自由使用与 [net/http](https://golang.org/pkg/net/http/) 包兼容的任何第三方中间件，相关示例 [_examples/convert-handlers](_examples/convert-handlers) 。

### 相关文章（英文）

* [A Todo MVC Application using Iris and Vue.js](https://hackernoon.com/a-todo-mvc-application-using-iris-and-vue-js-5019ff870064)
* [A Hasura starter project with a ready to deploy Golang hello-world web app with IRIS](bit.ly/2lmKaAZ)
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

### 工作机会

有很多公司都在寻找具有 Iris 经验的 Go 网站开发者，我们通过 [facebook page](https://www.facebook.com/iris.framework) 发布这些招聘信息。

### 赞助

感谢所有赞助者! (希望贵公司赞助支持这个开源项目)

<a href="https://opencollective.com/iris/sponsor/0/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/0/avatar.svg"></a>
<a href="https://opencollective.com/iris/sponsor/1/website" target="_blank"><img src="https://opencollective.com/iris/sponsor/1/avatar.svg"></a>

## 授权协议

Iris 授权基于 [3-Clause BSD License](LICENSE). Iris 是 100％ 免费和开源软件。

有关授权的任何问题，[请发送电子邮件](mailto:kataras2006@hotmail.com?subject=Iris%20License)。
