# [![Logo created by @santoshanand](logo_white_35_24.png)](https://iris-go.com) Iris

[![build status](https://img.shields.io/travis/kataras/iris/master.svg?style=flat-square)](https://travis-ci.org/kataras/iris)[![Backers on Open Collective](https://opencollective.com/iris/backers/badge.svg?style=flat-square)](#backers)[![Sponsors on Open Collective](https://opencollective.com/iris/sponsors/badge.svg?style=flat-square)](#sponsors)[![report card](https://img.shields.io/badge/report%20card-a%2B-ff3333.svg?style=flat-square)](http://goreportcard.com/report/kataras/iris)[![github closed issues](https://img.shields.io/github/issues-closed-raw/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aclosed)[![release](https://img.shields.io/github/release/kataras/iris.svg?style=flat-square)](https://github.com/kataras/iris/releases)[![view examples](https://img.shields.io/badge/learn%20by-examples-0077b3.svg?style=flat-square)](https://github.com/kataras/iris/tree/master/_examples)[![chat](https://img.shields.io/badge/community-%20chat-00BCD4.svg?style=flat-square)](https://kataras.rocket.chat/channel/iris)[![CLA assistant](https://cla-assistant.io/readme/badge/kataras/iris?style=flat-square)](https://cla-assistant.io/kataras/iris)

<a target='_blank' rel='nofollow' href='https://app.codesponsor.io/link/Qw6E1MTHvUJW6BtwUUf9qwsy/kataras/iris'>
  <img alt='Sponsor' width='888' height='68' src='https://app.codesponsor.io/embed/Qw6E1MTHvUJW6BtwUUf9qwsy/kataras/iris.svg' />
</a>

Iris是一个超快，使用简单并且非常高效的Go语言Web开发框架。
Iris is a fast, simple and efficient web framework for Go.

Iris功能很强大，但使用又很简单，它将会是你下一个网站、API服务或者分布式应用基础框架的不二之选。
Iris provides a beautifully expressive and easy to use foundation for your next website, API, or distributed app.

看看[别人是如何评价Iris](https://www.youtube.com/watch?v=jGx0LkuUs4A)，同时欢迎各位[成为Iris星探](https://github.com/kataras/iris/stargazers)，或者关注[Iris facebook主页](https://facebook.com/iris.framework)。
Learn what [others say about Iris](https://www.youtube.com/watch?v=jGx0LkuUs4A) and [star](https://github.com/kataras/iris/stargazers) this github repository to stay [up to date](https://facebook.com/iris.framework).

[![Iris vs .NET Core(C#) vs Node.js (Express)](https://iris-go.com/images/benchmark-new-gray.png)](_benchmarks)

<details>
<summary>上图是第三发机构发布的REST Web框架的基准测试Benchmarks from third-party source over the rest web frameworks</summary>

![Comparison with other frameworks](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

_Updated at: [Friday, 29 September 2017](_benchmarks)_
</details>

## Built with ♥️
## Built with ♥️

在发现Iris之前，我想你一定也看过其它Go Web开发框架，或许你已经摩拳擦掌并马上就要用上了，但我会很遗憾的告诉你，不管怎样，你将来还是会使用Iris的。不仅仅因为Iris性能卓越和使用简单，更重要的是Iris非常独特，他能让你成为真正的极客界的摇滚明星。

We have no doubt you will able to find other web frameworks written in Go
and even put up a real fight to learn and use them for quite some time but
make no mistake, sooner or later you will be using Iris, not because of the ergonomic, high-performant solution that it provides but its well-documented unique features, as these will transform you to a real rockstar geek.

不管你是想开发微服务或者大型Web应用，Iris都能满足你的需求，Iris可能是你在网上能找到最好的Web后台开发软件了。

No matter what you're trying to build, Iris covers 
every type of application, from micro services to large monolithic web applications.
It's actually the best piece of software for back-end web developers
you can find online.

Iris现在已经到第8版了，但是我们从未停止开发。有很多非常棒的功能已经提上开发日程了，而且我们非常乐意加入很多有创意的想法。

Iris may have reached version 8, but we're not stopping there. We have many feature ideas on our board that we're anxious to add and other innovative web development solutions that we're planning to build into Iris.

如果你想用CDN加速，我推荐用[KeyCDN](https://www.keycdn.com/)，因为KeyCDN简单、速度快而且稳定。

Accelerated by [KeyCDN](https://www.keycdn.com/), a simple, fast and reliable CDN.

我们用[微软](https://www.microsoft.com)开发的[Visual Studio Code](https://code.visualstudio.com/)来做为开发Golang的IDE。 

We are developing this project using the best code editor for Golang; [Visual Studio Code](https://code.visualstudio.com/) supported by [Microsoft](https://www.microsoft.com).

如果你之前使用[nodejs](https://nodejs.org)做开发，恭喜你，Iris使用基本和[expressjs](https://github.com/expressjs/express)一样。

If you're coming from [nodejs](https://nodejs.org) world, Iris is the [expressjs](https://github.com/expressjs/express) equivalent for Gophers.

## 内容列表

## Table Of Content

* [安装](#installation)
* [最近更新](https://github.com/kataras/iris/blob/master/HISTORY.md#tu-07-november-2017--v857)
* [快速入门](#getting-started)
* [进阶](_examples/)
    * [MVC (Model View Controller)](_examples/#mvc) **NEW**
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
    * [POC: Convert the medium-sized project "Parrot" from native to Iris](https://github.com/iris-contrib/parrot)
    * [POC: Isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/kataras/iris-starter-kit)
    * [Typescript Automation Tools](typescript/#table-of-contents)
    * [Tutorial: A URL Shortener Service using Go, Iris and Bolt](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)
    * [Tutorial: Online Visitors](_examples/tutorial/online-visitors)
    * [Tutorial: Caddy](_examples/tutorial/caddy)
    * [Tutorial: DropzoneJS Uploader](https://hackernoon.com/how-to-build-a-file-upload-form-using-dropzonejs-and-go-8fb9f258a991)
    * [Tutorial:Iris Go Framework + MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)
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

### MVC指南

```go
package main

import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/mvc"
)

func main() {
    app := iris.New()

    app.Controller("/helloworld", new(HelloWorldController))

    app.Run(iris.Addr("localhost:8080"))
}

type HelloWorldController struct {
    mvc.C

    // [ Your fields here ]
    // Request lifecycle data
    // Models
    // Database
    // Global properties
}

//
// GET: /helloworld

func (c *HelloWorldController) Get() string {
    return "This is my default action..."
}

//
// GET: /helloworld/{name:string}

func (c *HelloWorldController) GetBy(name string) string {
    return "Hello " + name
}

//
// GET: /helloworld/welcome

func (c *HelloWorldController) GetWelcome() (string, int) {
    return "This is the GetWelcome action func...", iris.StatusOK
}

//
// GET: /helloworld/welcome/{name:string}/{numTimes:int}

func (c *HelloWorldController) GetWelcomeBy(name string, numTimes int) {
    // Access to the low-level Context,
    // output arguments are optional of course so we don't have to use them here.
    c.Ctx.Writef("Hello %s, NumTimes is: %d", name, numTimes)
}

```
> [_examples/mvc](_examples/mvc) 和 [mvc/controller_test.go](https://github.com/kataras/iris/blob/master/mvc/controller_test.go) 两个简单的例子可以让你更好的了解 Iris MVC 的使用方式

每一个在controller中导出的Go方法名都和HTTP方法(`Get`, `Post`, `Put`, `Delete`...) 一一对应

在Web应用中一个HTTP访问的资源就是一个URL（统一资源定位符），比如`http://localhost:8080/helloworld`是由HTTP协议、Web服务网络位置（包括TCP端口）：`localhost:8080`以及资源名称URI（统一资源标志符） `/helloworld`组成的。

上面例子第一个方法映射到[HTTP GET](https://www.w3schools.com/tags/ref_httpmethods.asp)方法，访问资源是"/helloworld"，第三个方法映射到[HTTP GET](https://www.w3schools.com/tags/ref_httpmethods.asp)方法，访问资源是"/helloworld/welcome"


Controller在处理`GetBy`方法时可以识别路径‘name’参数，`GetWelcomeBy`方法可以识别路径‘name’和‘numTimes’参数，因为Controller在识别`By`关键字后可以动态灵活的处理路由；上面第四个方法指示使用 [HTTP GET](https://www.w3schools.com/tags/ref_httpmethods.asp)方法，而且只处理以"/helloworld/welcome"开头的资源位置路径，并且此路径还得包括两部分，第一部分类型没有限制，第二部分只能是数字类型，比如"http://localhost:8080/helloworld/welcome/golang/32719" 是合法的，其它的就会给客户端返回[404 找不到](https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html#sec10.4.5)的提示


### MVC 快速指南 2

Iris对MVC的支持非常 **棒[高性能](_benchmarks)** ， Iris通过方法的返回值，可以给客户端返回任意类型的数据：

*  如果返回的是 `string` 类型，就直接给客户端返回字符串
*  如果第二个返回值是 `string` 类型，那么这个值就是ContentType(HTTP header)的值
*  如果返回的是 `int` 类型，这个值就是HTTP状态码
*  如果返回 `error` 值不是空，Iris 将会把这个值作为HTTP 400页面的返回值内容
*  如果返回 `(int, error)` 类型，并且error不为空，那么Iris返回error的内容，同时把 `int` 值作为HTTP状态码
*  如果返回 `bool` 类型，并且值是 false ，Iris直接返回404页面
*  如果返回自定义` struct` 、 `interface{}` 、 `slice` 及 `map` ，Iris 将按照JSON的方式返回，注意如果第二个返回值是 `string`，那么Iris就按照这个 `string` 值的ContentType处理了(不一定是'application/json')
*  如果 `mvc.Result` 调用了 `Dispatch` 函数, 就会按照自己的逻辑重新处理

下面这些例子仅供参考，生产环境谨慎使用

```go
package main

import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/middleware/basicauth"
    "github.com/kataras/iris/mvc"
)

// Movie 是自定义数据结构
type Movie struct {
    Name   string `json:"name"`
    Year   int    `json:"year"`
    Genre  string `json:"genre"`
    Poster string `json:"poster"`
}

// movies 对象模拟数据源
var movies = []Movie{
    {
        Name:   "Casablanca",
        Year:   1942,
        Genre:  "Romance",
        Poster: "https://iris-go.com/images/examples/mvc-movies/1.jpg",
    },
    {
        Name:   "Gone with the Wind",
        Year:   1939,
        Genre:  "Romance",
        Poster: "https://iris-go.com/images/examples/mvc-movies/2.jpg",
    },
    {
        Name:   "Citizen Kane",
        Year:   1941,
        Genre:  "Mystery",
        Poster: "https://iris-go.com/images/examples/mvc-movies/3.jpg",
    },
    {
        Name:   "The Wizard of Oz",
        Year:   1939,
        Genre:  "Fantasy",
        Poster: "https://iris-go.com/images/examples/mvc-movies/4.jpg",
    },
}


var basicAuth = basicauth.New(basicauth.Config{
    Users: map[string]string{
        "admin": "password",
    },
})


func main() {
    app := iris.New()

    app.Use(basicAuth)

    app.Controller("/movies", new(MoviesController))

    app.Run(iris.Addr(":8080"))
}

// MoviesController 是 /movies controller.
type MoviesController struct {
    mvc.C
}

// 返回 movies列表
// 例子:
// curl -i http://localhost:8080/movies
func (c *MoviesController) Get() []Movie {
    return movies
}

// GetBy 返回一个 movie
// 例子:
// curl -i http://localhost:8080/movies/1
func (c *MoviesController) GetBy(id int) Movie {
    return movies[id]
}

// PutBy 更新一个 movie
// 例子:
// curl -i -X PUT -F "genre=Thriller" -F "poster=@/Users/kataras/Downloads/out.gif" http://localhost:8080/movies/1
func (c *MoviesController) PutBy(id int) Movie {
    // 获取一个 movie
    m := movies[id]

    // 获取一个poster文件
    file, info, err := c.Ctx.FormFile("poster")
    if err != nil {
        c.Ctx.StatusCode(iris.StatusInternalServerError)
        return Movie{}
    }
    file.Close()            // 我们不需要这个文件
    poster := info.Filename // 比如这就是上传的文件url
    genre := c.Ctx.FormValue("genre")

    // 更新poster
    m.Poster = poster
    m.Genre = genre
    movies[id] = m

    return m
}

// DeleteBy 删除一个 movie
// 例子:
// curl -i -X DELETE -u admin:password http://localhost:8080/movies/1
func (c *MoviesController) DeleteBy(id int) iris.Map {
    //从movies slice中删除索引
    deleted := movies[id].Name
    movies = append(movies[:id], movies[id+1:]...)
    // 返回删除movie的名称
    return iris.Map{"deleted": deleted}
}
```

### MVC 快速指南 3

Iris是一个底层的Web开发框架，如果你喜欢按 **目录结构** 的约定方式开发，那么Iris框架对此毫无影响。

你可以根据自己的需求来创建目录结构，但是我建议你还是最好看看如下的目录结构例子：

[![目录结构例子](_examples/mvc/overview/folder_structure.png)](_examples/mvc/overview)

好了，直接上代码。


#### 数据模型层

```go
// file: datamodels/movie.go

package datamodels

// Movie是我们例子数据结构
// 此Movie可能会定义在类似"web/viewmodels/movie.go"的文件
// Movie的数据模型在应用中只有一个，这样使用就很简单了
type Movie struct {
    ID     int64  `json:"id"`
    Name   string `json:"name"`
    Year   int    `json:"year"`
    Genre  string `json:"genre"`
    Poster string `json:"poster"`
}
```

#### Data Source / Data Store Layer

```go
// file: datasource/movies.go

package datasource

import "github.com/kataras/iris/_examples/mvc/overview/datamodels"

// Movies是模拟的数据源
var Movies = map[int64]datamodels.Movie{
    1: {
        ID:     1,
        Name:   "Casablanca",
        Year:   1942,
        Genre:  "Romance",
        Poster: "https://iris-go.com/images/examples/mvc-movies/1.jpg",
    },
    2: {
        ID:     2,
        Name:   "Gone with the Wind",
        Year:   1939,
        Genre:  "Romance",
        Poster: "https://iris-go.com/images/examples/mvc-movies/2.jpg",
    },
    3: {
        ID:     3,
        Name:   "Citizen Kane",
        Year:   1941,
        Genre:  "Mystery",
        Poster: "https://iris-go.com/images/examples/mvc-movies/3.jpg",
    },
    4: {
        ID:     4,
        Name:   "The Wizard of Oz",
        Year:   1939,
        Genre:  "Fantasy",
        Poster: "https://iris-go.com/images/examples/mvc-movies/4.jpg",
    },
    5: {
        ID:     5,
        Name:   "North by Northwest",
        Year:   1959,
        Genre:  "Thriller",
        Poster: "https://iris-go.com/images/examples/mvc-movies/5.jpg",
    },
}
```

#### 数据仓库

数据仓库层直接访问数据源

```go
// file: repositories/movie_repository.go

package repositories

import (
    "errors"
    "sync"

    "github.com/kataras/iris/_examples/mvc/overview/datamodels"
)

// Query 是数据访问的集合入口
type Query func(datamodels.Movie) bool

// MovieRepository 中会有对movie实体的基本操作
type MovieRepository interface {
    Exec(query Query, action Query, limit int, mode int) (ok bool)

    Select(query Query) (movie datamodels.Movie, found bool)
    SelectMany(query Query, limit int) (results []datamodels.Movie)

    InsertOrUpdate(movie datamodels.Movie) (updatedMovie datamodels.Movie, err error)
    Delete(query Query, limit int) (deleted bool)
}

// NewMovieRepository 返回movie内存数据
func NewMovieRepository(source map[int64]datamodels.Movie) MovieRepository {
    return &movieMemoryRepository{source: source}
}

// movieMemoryRepository 就是 "MovieRepository"，它管理movie的内存数据
type movieMemoryRepository struct {
    source map[int64]datamodels.Movie
    mu     sync.RWMutex
}

const (
    // 只读模式
    ReadOnlyMode = iota
    // 读写模式
    ReadWriteMode
)

func (r *movieMemoryRepository) Exec(query Query, action Query, actionLimit int, mode int) (ok bool) {
    loops := 0

    if mode == ReadOnlyMode {
        r.mu.RLock()
        defer r.mu.RUnlock()
    } else {
        r.mu.Lock()
        defer r.mu.Unlock()
    }

    for _, movie := range r.source {
        ok = query(movie)
        if ok {
            if action(movie) {
                loops++
                if actionLimit >= loops {
                    break // break
                }
            }
        }
    }

    return
}

// Select方法返回从模拟数据源找出的一个movie数据。
// 当找到时就返回true，并停止迭代
//
// Select 将会返回查询到的最新找到的movie数据，这样可以减少代码量
//
// 自从我第一次想到用这种简单的原型函数后，我就经常用它了，希望这也对你有用
func (r *movieMemoryRepository) Select(query Query) (movie datamodels.Movie, found bool) {
    found = r.Exec(query, func(m datamodels.Movie) bool {
        movie = m
        return true
    }, 1, ReadOnlyMode)

    // 如果没有找到就让datamodels.Movie为空
    // set an empty datamodels.Movie if not found at all.
    if !found {
        movie = datamodels.Movie{}
    }

    return
}

// 如果要查找很多值，用法基本一致，不过会返回datamodels.Movie slice。
// 如果limit<=0，将返回全部数据
func (r *movieMemoryRepository) SelectMany(query Query, limit int) (results []datamodels.Movie) {
    r.Exec(query, func(m datamodels.Movie) bool {
        results = append(results, m)
        return true
    }, limit, ReadOnlyMode)

    return
}

// 插入或更新数据
//
// 返回一个新的movie对象和error对象
func (r *movieMemoryRepository) InsertOrUpdate(movie datamodels.Movie) (datamodels.Movie, error) {
    id := movie.ID

    if id == 0 { // Create new action
        var lastID int64
        // 为了数据不重复，找到最大的ID。
        // 生产环境你可以用第三方库生成一个UUID字串
        r.mu.RLock()
        for _, item := range r.source {
            if item.ID > lastID {
                lastID = item.ID
            }
        }
        r.mu.RUnlock()

        id = lastID + 1
        movie.ID = id

        // map-specific thing
        r.mu.Lock()
        r.source[id] = movie
        r.mu.Unlock()

        return movie, nil
    }
    //通过movie.ID更新数据
    //这里举个例子看如果更新非空的poster和genre
    //其实我们可以直接更新对象r.source[id] = movie
    //用Select的话如下所示
    current, exists := r.Select(func(m datamodels.Movie) bool {
        return m.ID == id
    })

    if !exists { // ID不存在，返回error ID
        return datamodels.Movie{}, errors.New("failed to update a nonexistent movie")
    }

    // 或者直接对象操作替换
    // or comment these and r.source[id] = m for pure replace
    if movie.Poster != "" {
        current.Poster = movie.Poster
    }
    
    if movie.Genre != "" {
        current.Genre = movie.Genre
    }

    // 类map结构的处理
    r.mu.Lock()
    r.source[id] = current
    r.mu.Unlock()

    return movie, nil
}

func (r *movieMemoryRepository) Delete(query Query, limit int) bool {
    return r.Exec(query, func(m datamodels.Movie) bool {
        delete(r.source, m.ID)
        return true
    }, limit, ReadWriteMode)
}
```

#### 服务层

服务层主要调用“数据仓库”和“数据模型”的方法（即使是数据模型很简单的应用）。这一层将包含主要的数据处理逻辑。


```go
// file: services/movie_service.go

package services

import (
    "github.com/kataras/iris/_examples/mvc/overview/datamodels"
    "github.com/kataras/iris/_examples/mvc/overview/repositories"
)

// MovieService主要包括对movie的CRUID（增删改查）操作。
// MovieService主要调用movie 数据仓库的方法。
// 下面例子的数据源是更高级别的组件
// 这样可以用同样的逻辑可以返回不同的数据仓库
// MovieService是一个接口，任何实现的地方都能用，这样可以替换不同的业务逻辑用来测试
type MovieService interface {
    GetAll() []datamodels.Movie
    GetByID(id int64) (datamodels.Movie, bool)
    DeleteByID(id int64) bool
    UpdatePosterAndGenreByID(id int64, poster string, genre string) (datamodels.Movie, error)
}

// NewMovieService 返回一个 movie 服务.
func NewMovieService(repo repositories.MovieRepository) MovieService {
    return &movieService{
        repo: repo,
    }
}

type movieService struct {
    repo repositories.MovieRepository
}

// GetAll 返回所有 movies.
func (s *movieService) GetAll() []datamodels.Movie {
    return s.repo.SelectMany(func(_ datamodels.Movie) bool {
        return true
    }, -1)
}

// GetByID 是通过id找到movie.
func (s *movieService) GetByID(id int64) (datamodels.Movie, bool) {
    return s.repo.Select(func(m datamodels.Movie) bool {
        return m.ID == id
    })
}


// UpdatePosterAndGenreByID 更新一个 movie的 poster 和 genre.
func (s *movieService) UpdatePosterAndGenreByID(id int64, poster string, genre string) (datamodels.Movie, error) {
    // update the movie and return it.
    return s.repo.InsertOrUpdate(datamodels.Movie{
        ID:     id,
        Poster: poster,
        Genre:  genre,
    })
}

// DeleteByID 通过id删除一个movie
//
// 返回true表示成功，其它都是失败
func (s *movieService) DeleteByID(id int64) bool {
    return s.repo.Delete(func(m datamodels.Movie) bool {
        return m.ID == id
    }, 1)
}
```

#### 视图模型

视图模型将处理结果返回给客户端

例子：
Example:

```go
import (
    "github.com/kataras/iris/_examples/mvc/overview/datamodels"

    "github.com/kataras/iris/context"
)

type Movie struct {
    datamodels.Movie
}

func (m Movie) IsValid() bool {
    /* 做一些检测，如果ID合法就返回true */
    return m.ID > 0
}
```

Iris允许在HTTP Response Dispatcher中使用任何自定义数据结构，
所以理论上来说，除非万不得已，下面的代码不建议使用

```go
// Dispatch实现了`kataras/iris/mvc#Result`接口。在函数最后发送了一个`Movie`对象作为http response对象。
// 如果ID小于等于0就回返回404，或者就返回json数据。
//（这样就像控制器的方法默认返回自定义类型一样）
//
// 不要在这里写过多的代码，应用的主要逻辑不在这里
// 在方法返回之前可以做个简单验证处理等等；
//
// 这里只是一个小例子，想想这个优势在设计大型应用是很有作用的
//
// 这个方法是在`Movie`类型的控制器调用的。
// 例子在这里：`controllers/movie_controller.go#GetBy`。
func (m Movie) Dispatch(ctx context.Context) {
    if !m.IsValid() {
        ctx.NotFound()
        return
    }
    ctx.JSON(m, context.JSON{Indent: " "})
}
```
然而，我们仅仅用"datamodels"作为一个数据模型包，是因为Movie数据结构没有包含敏感数据，客户端可以访问到其所有字段，我们不需要再有额外的功能去做验证处理了


#### 控制器

控制器处理Web请求，它是服务层和客户端之间的桥梁

```go
// file: web/controllers/movie_controller.go

package controllers

import (
    "errors"

    "github.com/kataras/iris/_examples/mvc/overview/datamodels"
    "github.com/kataras/iris/_examples/mvc/overview/services"

    "github.com/kataras/iris"
    "github.com/kataras/iris/mvc"
)

// MovieController是/movies的控制器
type MovieController struct {
    mvc.C

    // MovieService是一个接口，主app对象会持有它
    Service services.MovieService
}

// 获取movies列表
// 例子：
// curl -i http://localhost:8080/movies
//
// 如果你有一些敏感的数据要处理的话，可以按照如下所示的方式：
// func (c *MovieController) Get() (results []viewmodels.Movie) {
//  data := c.Service.GetAll()
//
//  for _, movie := range data {
// 	  results = append(results, viewmodels.Movie{movie})
//  }
//  return
// }
//否则直接返回数据模型
func (c *MovieController) Get() (results []datamodels.Movie) {
    return c.Service.GetAll()
}

// GetBy返回一个movie对象
// 例子：
// curl -i http://localhost:8080/movies/1
func (c *MovieController) GetBy(id int64) (movie datamodels.Movie, found bool) {
    return c.Service.GetByID(id) // 404 没有找到
}

// PutBy更新一个movie.
// 例子:
// curl -i -X PUT -F "genre=Thriller" -F "poster=@/Users/kataras/Downloads/out.gif" http://localhost:8080/movies/1
func (c *MovieController) PutBy(id int64) (datamodels.Movie, error) {
    // 从请求中获取poster和genre
    file, info, err := c.Ctx.FormFile("poster")
    if err != nil {
        return datamodels.Movie{}, errors.New("failed due form file 'poster' missing")
    }
    // 关闭文件
    file.Close()

    //想象这就是一个上传文件的url
    poster := info.Filename
    genre := c.Ctx.FormValue("genre")

    return c.Service.UpdatePosterAndGenreByID(id, poster, genre)
}

// DeleteBy删除一个movie对象
// 例子:
// curl -i -X DELETE -u admin:password http://localhost:8080/movies/1
func (c *MovieController) DeleteBy(id int64) interface{} {
    wasDel := c.Service.DeleteByID(id)
    if wasDel {
        // 返回要删除的ID
        return iris.Map{"deleted": id}
    }
    //现在我们可以看到这里可以返回一个有2个返回值(map或int)的函数
    //我们并没有指定一个返回的类型
    return iris.StatusBadRequest
}
```

```go
// file: web/controllers/hello_controller.go

package controllers

import (
    "errors"

    "github.com/kataras/iris/mvc"
)

// HelloController是控制器的例子
// 下面会处理GET: /hello and GET: /hello/{name}
type HelloController struct {
    mvc.C
}

var helloView = mvc.View{
    Name: "hello/index.html",
    Data: map[string]interface{}{
        "Title":     "Hello Page",
        "MyMessage": "Welcome to my awesome website",
    },
}

// Get会返回预定义绑定数据的视图
//
// `mvc.Result`是一个含有`Dispatch`方法的接口
// `mvc.Response` 和 `mvc.View` dispatchers 内置类型
// 你也可以通过实现`github.com/kataras/iris/mvc#Result`接口来自定义dispatchers
func (c *HelloController) Get() mvc.Result {
    return helloView
}

// 你可以定义一个标准通用的error
var errBadName = errors.New("bad name")

//你也可以将error包裹在mvc.Response中，这样就和mvc.Result类型兼容了
var badName = mvc.Response{Err: errBadName, Code: 400}

// GetBy 返回 "Hello {name}" response
// 例子:
// curl -i http://localhost:8080/hello/iris
// curl -i http://localhost:8080/hello/anything
func (c *HelloController) GetBy(name string) mvc.Result {
    if name != "iris" {
        return badName
        // 或者
        // GetBy(name string) (mvc.Result, error) {
        //  return nil, errBadName
        // }
    }

    // 返回 mvc.Response{Text: "Hello " + name} 或者:
    return mvc.View{
        Name: "hello/name.html",
        Data: name,
    }
}
```

```go
// file: web/middleware/basicauth.go

package middleware

import "github.com/kataras/iris/middleware/basicauth"

// BasicAuth 中间件例
var BasicAuth = basicauth.New(basicauth.Config{
    Users: map[string]string{
        "admin": "password",
    },
})
```

```html
<!-- file: web/views/hello/index.html -->
<html>

<head>
    <title>{{.Title}} - My App</title>
</head>

<body>
    <p>{{.MyMessage}}</p>
</body>

</html>
```

```html
<!-- file: web/views/hello/name.html -->
<html>

<head>
    <title>{{.}}' Portfolio - My App</title>
</head>

<body>
    <h1>Hello {{.}}</h1>
</body>

</html>
```

> 戳[_examples/view](_examples/#view) 可以找到更多关于layouts，tmpl，routing的例子


#### 程序入口

程序入口可以将任何组件包含进来

```go
// file: main.go

package main

import (
    "github.com/kataras/iris/_examples/mvc/overview/datasource"
    "github.com/kataras/iris/_examples/mvc/overview/repositories"
    "github.com/kataras/iris/_examples/mvc/overview/services"
    "github.com/kataras/iris/_examples/mvc/overview/web/controllers"
    "github.com/kataras/iris/_examples/mvc/overview/web/middleware"

    "github.com/kataras/iris"
)

func main() {
    app := iris.New()

    // 加载模板文件
    app.RegisterView(iris.HTML("./web/views", ".html"))

    // 注册控制器
    app.Controller("/hello", new(controllers.HelloController))

    // 创建movie 数据仓库，次仓库包含的是内存级的数据源
    repo := repositories.NewMovieRepository(datasource.Movies)
    // 创建movie服务, 然后将其与控制器绑定
    movieService := services.NewMovieService(repo)

    app.Controller("/movies", new(controllers.MovieController),
        // 将"movieService"绑定在 MovieController的Service接口
        movieService,
        // 为/movies请求添加basic authentication(admin:password)中间件
        middleware.BasicAuth)

    // 启动应用localhost:8080
    // http://localhost:8080/hello
    // http://localhost:8080/hello/iris
    // http://localhost:8080/movies
    // http://localhost:8080/movies/1
    app.Run(
        iris.Addr("localhost:8080"),
        iris.WithoutVersionChecker,
        iris.WithoutServerError(iris.ErrServerClosed),
        iris.WithOptimizations, // 可以启用快速json序列化等优化配置
    )
}
```

更多指南戳 [_examples/#structuring](_examples/#structuring)

## 现在你已经准备好进入下一阶段，又向专家级gopher迈进一步了

恭喜你看到这里了，我们为你准备了更高水平的内容，向真正的专家级gopher进军吧😃

> 准备好咖啡，尽情享受吧！

* [Iris Go 矿建+ MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)
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

非常感谢所有对Iris的贡献者，没有你们就没有Iris [Contribute](CONTRIBUTING.md)
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
