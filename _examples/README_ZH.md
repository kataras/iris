
# 示例

请先学习如何使用 [net/http](https://golang.org/pkg/net/http/) 

这里包含大部分 [iris](https://github.com/kataras/iris) 网络微框架的简单使用示例

这些示例不一定是最优解，但涵盖了 Iris 的大部分重要功能。

### 概览

- [Hello world!](hello-world/main.go)
- [基础](overview/main.go)
- [教程: 在线人数](tutorial/online-visitors/main.go)
- [教程: A Todo MVC Application using Iris and Vue.js](https://hackernoon.com/a-todo-mvc-application-using-iris-and-vue-js-5019ff870064)
- [教程: 结合 BoltDB 生成短网址](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)
- [教程: 用安卓设备搭建服务器 (**MUST**)](https://twitter.com/ThePracticalDev/status/892022594031017988)
- [POC: Convert the medium-sized project "Parrot" from native to Iris](https://github.com/iris-contrib/parrot)
- [POC: Isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/kataras/iris-starter-kit)
- [教程: DropzoneJS 上传](tutorial/dropzonejs)
- [教程: Caddy 服务器使用](tutorial/caddy)
- [教程: Iris + MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)

### 目录结构

Iris 是个底层框架, 对 MVC 模式有很好的支持，但不限制文件夹结构，你可以随意组织你的代码。

如何组织代码取决于你的需求. 我们无法告诉你如何设计程序，但你可以仔细查看下面的示例，也许有些片段可以直接放到你的程序里。

- [引导模式架构](structuring/bootstrap)
- [MVC 存储层与服务层](structuring/mvc-plus-repository-and-service-layers)
- [登录演示 (MVC 使用独立包组织)](structuring/login-mvc-single-responsibility-package)
- [登录演示 (MVC 数据模型, 数据源, 存储 和 服务层)](structuring/login-mvc)

### HTTP 监听

- [基础用法](http-listening/listen-addr/main.go)
    * [忽略错误信息](http-listening/listen-addr/omit-server-errors/main.go)
- [UNIX socket file](http-listening/listen-unix/main.go)
- [TLS](http-listening/listen-tls/main.go)
- [Letsencrypt (自动认证)](http-listening/listen-letsencrypt/main.go)
- [进程关闭通知](http-listening/notify-on-shutdown/main.go)
- 自定义 TCP 监听器
    * [通用 net.Listener](http-listening/custom-listener/main.go)
    * [SO_REUSEPORT for unix systems](http-listening/custom-listener/unix-reuseport/main.go)
- 自定义 HTTP 服务
    * [easy way](http-listening/custom-httpserver/easy-way/main.go)
    * [std way](http-listening/custom-httpserver/std-way/main.go)
    * [多个服务示例](http-listening/custom-httpserver/multi/main.go)
- 优雅关闭
    * [使用 `RegisterOnInterrupt`](http-listening/graceful-shutdown/default-notifier/main.go)
    * [自定义通知](http-listening/graceful-shutdown/custom-notifier/main.go)

### 配置

- [基本配置方式](configuration/functional/main.go)
- [Struct 方式配置](configuration/from-configuration-structure/main.go)
- [导入 YAML 配置文件](configuration/from-yaml-file/main.go)
    * [多实例共享配置](configuration/from-yaml-file/shared-configuration/main.go)
- [导入 TOML 配置文件](configuration/from-toml-file/main.go)

### 路由、路由分组、路径动态参数、路由参数处理宏 、 自定义上下文

* `app.Get("{userid:int min(1)}", myHandler)`
* `app.Post("{asset:path}", myHandler)`
* `app.Put("{custom:string regexp([a-z]+)}", myHandler)`

提示: 不同于其他路由处理, iris 路由可以处理以下各种情况:
```go
// 匹配静态前缀 "/assets/" 的各种请求
app.Get("/assets/{asset:path}", assetsWildcardHandler)

// 只匹配 GET "/"
app.Get("/", indexHandler)
// 只匹配 GET "/about"
app.Get("/about", aboutHandler)

// 匹配前缀为 "/profile/" 的所有 GET 请求
// 接着是其余部分的匹配
app.Get("/profile/{username:string}", userHandler)
// 只匹配 "/profile/me" GET 请求，
// 这和 /profile/{username:string} 
// 或跟通配符 {root:path} 不冲突
app.Get("/profile/me", userHandler)

// 匹配所有前缀为 /users/ 的 GET 请求
// 参数为数字，且 >= 1
app.Get("/user/{userid:int min(1)}", getUserHandler)
// 匹配所有前缀为 /users/ 的 DELETE 请求
// 参数为数字，且 >= 1
app.Delete("/user/{userid:int min(1)}", deleteUserHandler)

// 匹配所有 GET 请求，除了 "/", "/about", 或其他以 "/assets/" 开头
// 因为它不会与其他路线冲突。
app.Get("{root:path}", rootWildcardHandler)
```

可以浏览以下示例，以便更好理解

- [概览](routing/overview/main.go)
- [基本使用](routing/basic/main.go)
- [控制器](mvc)
- [自定义 HTTP 错误](routing/http-errors/main.go)
- [动态路径](routing/dynamic-path/main.go)
    * [根级通配符路径](routing/dynamic-path/root-wildcard/main.go)
- [反向路由](routing/reverse/main.go)
- [自定义包装](routing/custom-wrapper/main.go)
- 自定义上下文
    * [方法重写](routing/custom-context/method-overriding/main.go)
    * [新实现方式](routing/custom-context/new-implementation/main.go)
- [路由状态](routing/route-state/main.go)
- [中间件定义](routing/writing-a-middleware)
    * [路由前](routing/writing-a-middleware/per-route/main.go)
    * [全局](routing/writing-a-middleware/globally/main.go)

### hero (输出的一种高效包装模式)

- [基础](hero/basic/main.go)
- [概览](hero/overview)

### MVC 模式

![](mvc/web_mvc_diagram.png)

Iris **对 MVC (Model View Controller) 有一流的支持**, 在 Go 社区里是独一无二的。

Iris 支持快速的请求数据，模型，持久性数据和绑定。

**特点**

All HTTP Methods are supported, for example if want to serve `GET`
then the controller should have a function named `Get()`,
you can define more than one method function to serve in the same Controller.

Serve custom controller's struct's methods as handlers with custom paths(even with regex parametermized path) via the `BeforeActivation` custom event callback, per-controller. Example:

```go
import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/mvc"
)

func main() {
    app := iris.New()
    mvc.Configure(app.Party("/root"), myMVC)
    app.Run(iris.Addr(":8080"))
}

func myMVC(app *mvc.Application) {
    // app.Register(...)
    // app.Router.Use/UseGlobal/Done(...)
    app.Handle(new(MyController))
}

type MyController struct {}

func (m *MyController) BeforeActivation(b mvc.BeforeActivation) {
    // b.Dependencies().Add/Remove
    // b.Router().Use/UseGlobal/Done // and any standard API call you already know

    // 1-> Method
    // 2-> Path
    // 3-> The controller's function name to be parsed as handler
    // 4-> Any handlers that should run before the MyCustomHandler
    b.Handle("GET", "/something/{id:long}", "MyCustomHandler", anyMiddleware...)
}

// GET: http://localhost:8080/root
func (m *MyController) Get() string { return "Hey" }

// GET: http://localhost:8080/root/something/{id:long}
func (m *MyController) MyCustomHandler(id int64) string { return "MyCustomHandler says Hey" }
```

Persistence data inside your Controller struct (share data between requests)
by defining services to the Dependencies or have a `Singleton` controller scope.

Share the dependencies between controllers or register them on a parent MVC Application, and ability
to modify dependencies per-controller on the `BeforeActivation` optional event callback inside a Controller,
i.e `func(c *MyController) BeforeActivation(b mvc.BeforeActivation) { b.Dependencies().Add/Remove(...) }`.

Access to the `Context` as a controller's field(no manual binding is neede) i.e `Ctx iris.Context` or via a method's input argument, i.e `func(ctx iris.Context, otherArguments...)`.

Models inside your Controller struct (set-ed at the Method function and rendered by the View).
You can return models from a controller's method or set a field in the request lifecycle
and return that field to another method, in the same request lifecycle.

Flow as you used to, mvc application has its own `Router` which is a type of `iris/router.Party`, the standard iris api.
`Controllers` can be registered to any `Party`, including Subdomains, the Party's begin and done handlers work as expected.

Optional `BeginRequest(ctx)` function to perform any initialization before the method execution,
useful to call middlewares or when many methods use the same collection of data.

Optional `EndRequest(ctx)` function to perform any finalization after any method executed.

Inheritance, recursively, see for example our `mvc.SessionController`, it has the `Session *sessions.Session` and `Manager *sessions.Sessions` as embedded fields
which are filled by its `BeginRequest`, [here](https://github.com/kataras/iris/blob/master/mvc/session_controller.go).
This is just an example, you could use the `sessions.Session` which returned from the manager's `Start` as a dynamic dependency to the MVC Application, i.e
`mvcApp.Register(sessions.New(sessions.Config{Cookie: "iris_session_id"}).Start)`.

Access to the dynamic path parameters via the controller's methods' input arguments, no binding is needed.
When you use the Iris' default syntax to parse handlers from a controller, you need to suffix the methods
with the `By` word, uppercase is a new sub path. Example:

If `mvc.New(app.Party("/user")).Handle(new(user.Controller))`

- `func(*Controller) Get()` - `GET:/user`.
- `func(*Controller) Post()` - `POST:/user`.
- `func(*Controller) GetLogin()` - `GET:/user/login`
- `func(*Controller) PostLogin()` - `POST:/user/login`
- `func(*Controller) GetProfileFollowers()` - `GET:/user/profile/followers`
- `func(*Controller) PostProfileFollowers()` - `POST:/user/profile/followers`
- `func(*Controller) GetBy(id int64)` - `GET:/user/{param:long}`
- `func(*Controller) PostBy(id int64)` - `POST:/user/{param:long}`

If `mvc.New(app.Party("/profile")).Handle(new(profile.Controller))`

- `func(*Controller) GetBy(username string)` - `GET:/profile/{param:string}`

If `mvc.New(app.Party("/assets")).Handle(new(file.Controller))`

- `func(*Controller) GetByWildard(path string)` - `GET:/assets/{param:path}`

    Supported types for method functions receivers: int, int64, bool and string.

Response via output arguments, optionally, i.e

```go
func(c *ExampleController) Get() string |
                                (string, string) |
                                (string, int) |
                                int |
                                (int, string) |
                                (string, error) |
                                error |
                                (int, error) |
                                (any, bool) |
                                (customStruct, error) |
                                customStruct |
                                (customStruct, int) |
                                (customStruct, string) |
                                mvc.Result or (mvc.Result, error)
```

where [mvc.Result](https://github.com/kataras/iris/blob/master/mvc/func_result.go) is an interface which contains only that function: `Dispatch(ctx iris.Context)`.

## Iris MVC 模式代码复用

By creating components that are independent of one another, developers are able to reuse components quickly and easily in other applications. The same (or similar) view for one application can be refactored for another application with different data because the view is simply handling how the data is being displayed to the user.

If you're new to back-end web development read about the MVC architectural pattern first, a good start is that [wikipedia article](https://en.wikipedia.org/wiki/Model%E2%80%93view%E2%80%93controller).

参考下面的示例

- [Hello world](mvc/hello-world/main.go) **UPDATED**
- [Session Controller](mvc/session-controller/main.go) **UPDATED**
- [Overview - Plus Repository and Service layers](mvc/overview) **UPDATED**
- [Login showcase - Plus Repository and Service layers](mvc/login) **UPDATED**
- [Singleton](mvc/singleton) **NEW**
- [Websocket Controller](mvc/websocket) **NEW**
- [Register Middleware](mvc/middleware) **NEW**
- [Vue.js Todo MVC](tutorial/vuejs-todo-mvc) **NEW**

### 子域名

- [单域名](subdomains/single/main.go)
- [多域名](subdomains/multi/main.go)
- [通配符](subdomains/wildcard/main.go)
- [WWW](subdomains/www/main.go)
- [快速跳转](subdomains/redirect/main.go)

### 改造 `http.Handler/HandlerFunc`

- [From func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)](convert-handlers/negroni-like/main.go)
- [From http.Handler or http.HandlerFunc](convert-handlers/nethttp/main.go)
- [From func(http.HandlerFunc) http.HandlerFunc](convert-handlers/real-usecase-raven/writing-middleware/main.go)

### 视图

| 模板引擎 | 调用声明 |
| -----------|-------------|
| template/html | `iris.HTML(...)`       |
| django        | `iris.Django(...)`     |
| handlebars    | `iris.Handlebars(...)` |
| amber         | `iris.Amber(...)`      |
| pug(jade)     | `iris.Pug(...)`        |

- [Overview](view/overview/main.go)
- [Hi](view/template_html_0/main.go)
- [A simple Layout](view/template_html_1/main.go)
- [Layouts: `yield` and `render` tmpl funcs](view/template_html_2/main.go)
- [The `urlpath` tmpl func](view/template_html_3/main.go)
- [The `url` tmpl func](view/template_html_4/main.go)
- [Inject Data Between Handlers](view/context-view-data/main.go)
- [Embedding Templates Into App Executable File](view/embedding-templates-into-app/main.go)
- [Write to a custom `io.Writer`](view/write-to)
- [Greeting with Pug (Jade)`](view/template_pug_0)
- [Pug (Jade) Actions`](view/template_pug_1)
- [Pug (Jade) Includes`](view/template_pug_2)
- [Pug (Jade) Extends`](view/template_pug_3)

You can serve [quicktemplate](https://github.com/valyala/quicktemplate) and [hero templates](https://github.com/shiyanhui/hero/hero) files too, simply by using the `context#ResponseWriter`, take a look at the [http_responsewriter/quicktemplate](http_responsewriter/quicktemplate) and [http_responsewriter/herotemplate](http_responsewriter/herotemplate) examples.

### 认证

- [Basic Authentication](authentication/basicauth/main.go)
- [OAUth2](authentication/oauth2/main.go)
- [JWT](experimental-handlers/jwt/main.go)
- [Sessions](#sessions)

### 文件服务器

- [Favicon](file-server/favicon/main.go)
- [Basic](file-server/basic/main.go)
- [Embedding Files Into App Executable File](file-server/embedding-files-into-app/main.go)
- [Embedding Gziped Files Into App Executable File](file-server/embedding-gziped-files-into-app/main.go) **NEW**
- [Send/Force-Download Files](file-server/send-files/main.go)
- Single Page Applications
    * [single Page Application](file-server/single-page-application/basic/main.go)
    * [embedded Single Page Application](file-server/single-page-application/embedded-single-page-application/main.go)
    * [embedded Single Page Application with other routes](file-server/single-page-application/embedded-single-page-application-with-other-routes/main.go)

### How to Read from `context.Request() *http.Request`

- [Read JSON](http_request/read-json/main.go)
- [Read XML](http_request/read-xml/main.go)
- [Read Form](http_request/read-form/main.go)
- [Read Custom per type](http_request/read-custom-per-type/main.go)
- [Read Custom via Unmarshaler](http_request/read-custom-via-unmarshaler/main.go)
- [Upload/Read File](http_request/upload-file/main.go)
- [Upload multiple files with an easy way](http_request/upload-files/main.go)

> The `context.Request()` returns the same *http.Request you already know, these examples show some places where the  Context uses this object. Besides that you can use it as you did before iris.

### How to Write to `context.ResponseWriter() http.ResponseWriter`

- [Write `valyala/quicktemplate` templates](http_responsewriter/quicktemplate)
- [Write `shiyanhui/hero` templates](http_responsewriter/herotemplate)
- [Text, Markdown, HTML, JSON, JSONP, XML, Binary](http_responsewriter/write-rest/main.go)
- [Write Gzip](http_responsewriter/write-gzip/main.go)
- [Stream Writer](http_responsewriter/stream-writer/main.go)
- [Transactions](http_responsewriter/transactions/main.go)
- [SSE (third-party package usage for server-side events)](http_responsewriter/sse-third-party/main.go)

> The `context/context#ResponseWriter()` returns an enchament version of a http.ResponseWriter, these examples show some places where the Context uses this object. Besides that you can use it as you did before iris.

### ORM

- [Using xorm(Mysql, MyMysql, Postgres, Tidb, **SQLite**, MsSql, MsSql, Oracle)](orm/xorm/main.go)

### 其他

- [Request Logger](http_request/request-logger/main.go)
    * [log requests to a file](http_request/request-logger/request-logger-file/main.go)
- [Localization and Internationalization](miscellaneous/i18n/main.go)
- [Recovery](miscellaneous/recover/main.go)
- [Profiling (pprof)](miscellaneous/pprof/main.go)
- [Internal Application File Logger](miscellaneous/file-logger/main.go)
- [Google reCAPTCHA](miscellaneous/recaptcha/main.go) 

### 试验性质处理器

- [Casbin wrapper](experimental-handlers/casbin/wrapper/main.go)
- [Casbin middleware](experimental-handlers/casbin/middleware/main.go)
- [Cloudwatch](experimental-handlers/cloudwatch/simple/main.go)
- [CORS](experimental-handlers/cors/simple/main.go)
- [JWT](experimental-handlers/jwt/main.go)
- [Newrelic](experimental-handlers/newrelic/simple/main.go)
- [Prometheus](experimental-handlers/prometheus/simple/main.go)
- [Secure](experimental-handlers/secure/simple/main.go)
- [Tollboothic](experimental-handlers/tollboothic/limit-handler/main.go)
- [Cross-Site Request Forgery Protection](experimental-handlers/csrf/main.go)

#### 更多

https://github.com/kataras/iris/tree/master/middleware#third-party-handlers

### 自动 API 文档

- [yaag](apidoc/yaag/main.go)

### 测试

The `httptest` package is your way for end-to-end HTTP testing, it uses the httpexpect library created by our friend, [gavv](https://github.com/gavv).

[Example](testing/httptest/main_test.go)

### 缓存

Iris 独立缓存包 [package](https://github.com/kataras/iris/tree/master/cache).

- [简单示例](cache/simple/main.go)
- [客户端 (304)](cache/client-side/main.go) - context 方法

> 可以随意使用自定义的缓存包。

### Cookies

- [Basic](cookies/basic/main.go)
- [Encode/Decode (securecookie)](cookies/securecookie/main.go)

### Sessions

Iris session 管理独立包 [package](https://github.com/kataras/iris/tree/master/sessions).

- [Overview](sessions/overview/main.go)
- [Standalone](sessions/standalone/main.go)
- [Secure Cookie](sessions/securecookie/main.go)
- [Flash Messages](sessions/flash-messages/main.go)
- [Databases](sessions/database)
    * [Badger](sessions/database/badger/main.go)
    * [Redis](sessions/database/redis/main.go)

> 可以随意使用自定义的 Session 管理包。

### Websockets

iris websocket library lives on its own [package](https://github.com/kataras/iris/tree/master/websocket).

The package is designed to work with raw websockets although its API is similar to the famous [socket.io](https://socket.io). I have read an article recently and I felt very contented about my decision to design a **fast** websocket-**only** package for Iris and not a backwards socket.io-like package. You can read that article by following this link: https://medium.com/@ivanderbyl/why-you-don-t-need-socket-io-6848f1c871cd.

- [Chat](websocket/chat/main.go)
- [Native Messages](websocket/native-messages/main.go)
- [Connection List](websocket/connectionlist/main.go)
- [TLS Enabled](websocket/secure/main.go)
- [Custom Raw Go Client](websocket/custom-go-client/main.go)
- [Third-Party socket.io](websocket/third-party-socketio/main.go)

> 如果你愿意，你可以自由使用你自己喜欢的websockets包。

### Typescript 自动化工具

Typescript 自动化工具独立库： [https://github.com/kataras/iris/tree/master/typescript](https://github.com/kataras/iris/tree/master/typescript) **包含相关示例**

### 大兄弟

进一步学习可通过 [godocs](https://godoc.org/github.com/kataras/iris) 和 https://docs.iris-go.com

不要忘记点赞 [star or watch](https://github.com/kataras/iris/stargazers) 这个项目会一直跟进最新趋势。
