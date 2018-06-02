# Examples

Please do learn how [net/http](https://golang.org/pkg/net/http/) std package works, first.

This folder provides easy to understand code snippets on how to get started with [iris](https://github.com/kataras/iris) micro web framework.

It doesn't always contain the "best ways" but it does cover each important feature that will make you so excited to GO with iris!

### Overview

- [Hello world!](hello-world/main.go)
- [Glimpse](overview/main.go)
- [Tutorial: Online Visitors](tutorial/online-visitors/main.go)
- [Tutorial: A Todo MVC Application using Iris and Vue.js](https://hackernoon.com/a-todo-mvc-application-using-iris-and-vue-js-5019ff870064)
- [Tutorial: URL Shortener using BoltDB](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)
- [Tutorial: How to turn your Android Device into a fully featured Web Server (**MUST**)](https://twitter.com/ThePracticalDev/status/892022594031017988)
- [POC: Convert the medium-sized project "Parrot" from native to Iris](https://github.com/iris-contrib/parrot)
- [POC: Isomorphic react/hot reloadable/redux/css-modules starter kit](https://github.com/kataras/iris-starter-kit)
- [Tutorial: DropzoneJS Uploader](tutorial/dropzonejs)
- [Tutorial: Caddy](tutorial/caddy)
- [Tutorial:Iris Go Framework + MongoDB](https://medium.com/go-language/iris-go-framework-mongodb-552e349eab9c)

### Structuring

Nothing stops you from using your favorite folder structure. Iris is a low level web framework, it has got MVC first-class support but it doesn't limit your folder structure, this is your choice.

Structuring depends on your own needs. We can't tell you how to design your own application for sure but you're free to take a closer look to the examples below; you may find something useful that you can borrow for your app;

- [Bootstrapper](structuring/bootstrap)
- [MVC with Repository and Service layer Overview](structuring/mvc-plus-repository-and-service-layers)
- [Login (MVC with Single Responsibility package)](structuring/login-mvc-single-responsibility-package)
- [Login (MVC with Datamodels, Datasource, Repository and Service layer)](structuring/login-mvc)

### HTTP Listening

- [Common, with address](http-listening/listen-addr/main.go)
    * [omit server errors](http-listening/listen-addr/omit-server-errors/main.go)
- [UNIX socket file](http-listening/listen-unix/main.go)
- [TLS](http-listening/listen-tls/main.go)
- [Letsencrypt (Automatic Certifications)](http-listening/listen-letsencrypt/main.go)
- [Notify on shutdown](http-listening/notify-on-shutdown/main.go)
- Custom TCP Listener
    * [common net.Listener](http-listening/custom-listener/main.go)
    * [SO_REUSEPORT for unix systems](http-listening/custom-listener/unix-reuseport/main.go)
- Custom HTTP Server
    * [easy way](http-listening/custom-httpserver/easy-way/main.go)
    * [std way](http-listening/custom-httpserver/std-way/main.go)
    * [multi server instances](http-listening/custom-httpserver/multi/main.go)
- Graceful Shutdown
    * [using the `RegisterOnInterrupt`](http-listening/graceful-shutdown/default-notifier/main.go)
    * [using a custom notifier](http-listening/graceful-shutdown/custom-notifier/main.go)

### Configuration

- [Functional](configuration/functional/main.go)
- [From Configuration Struct](configuration/from-configuration-structure/main.go)
- [Import from YAML file](configuration/from-yaml-file/main.go)
    * [Share Configuration between multiple instances](configuration/from-yaml-file/shared-configuration/main.go)
- [Import from TOML file](configuration/from-toml-file/main.go)

### Routing, Grouping, Dynamic Path Parameters, "Macros" and Custom Context

* `app.Get("{userid:int min(1)}", myHandler)`
* `app.Post("{asset:path}", myHandler)`
* `app.Put("{custom:string regexp([a-z]+)}", myHandler)`

Note: unlike other routers you'd seen, iris' router can handle things like these:
```go
// Matches all GET requests prefixed with "/assets/"
app.Get("/assets/{asset:path}", assetsWildcardHandler)

// Matches only GET "/"
app.Get("/", indexHandler)
// Matches only GET "/about"
app.Get("/about", aboutHandler)

// Matches all GET requests prefixed with "/profile/"
// and followed by a single path part
app.Get("/profile/{username:string}", userHandler)
// Matches only GET "/profile/me" because 
// it does not conflict with /profile/{username:string}
// or the root wildcard {root:path}
app.Get("/profile/me", userHandler)

// Matches all GET requests prefixed with /users/
// and followed by a number which should be equal or bigger than 1
app.Get("/user/{userid:int min(1)}", getUserHandler)
// Matches all requests DELETE prefixed with /users/
// and following by a number which should be equal or bigger than 1
app.Delete("/user/{userid:int min(1)}", deleteUserHandler)

// Matches all GET requests except "/", "/about", anything starts with "/assets/" etc...
// because it does not conflict with the rest of the routes.
app.Get("{root:path}", rootWildcardHandler)
```

Navigate through examples for a better understanding.

- [Overview](routing/overview/main.go)
- [Basic](routing/basic/main.go)
- [Controllers](mvc)
- [Custom HTTP Errors](routing/http-errors/main.go)
- [Dynamic Path](routing/dynamic-path/main.go)
    * [root level wildcard path](routing/dynamic-path/root-wildcard/main.go)
- [Reverse routing](routing/reverse/main.go)
- [Custom wrapper](routing/custom-wrapper/main.go)
- Custom Context
    * [method overriding](routing/custom-context/method-overriding/main.go)
    * [new implementation](routing/custom-context/new-implementation/main.go)
- [Route State](routing/route-state/main.go)
- [Writing a middleware](routing/writing-a-middleware)
    * [per-route](routing/writing-a-middleware/per-route/main.go)
    * [globally](routing/writing-a-middleware/globally/main.go)

### hero

- [Basic](hero/basic/main.go)
- [Overview](hero/overview)

### MVC

![](mvc/web_mvc_diagram.png)

Iris has **first-class support for the MVC (Model View Controller) pattern**, you'll not find
these stuff anywhere else in the Go world.

Iris web framework supports Request data, Models, Persistence Data and Binding
with the fastest possible execution.

**Characteristics**

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

where [mvc.Result](https://github.com/kataras/iris/blob/master/mvc/go19.go#L10) is an [interface](https://github.com/kataras/iris/blob/master/hero/func_result.go#L18) which contains only that function: `Dispatch(ctx iris.Context)`.

## Using Iris MVC for code reuse

By creating components that are independent of one another, developers are able to reuse components quickly and easily in other applications. The same (or similar) view for one application can be refactored for another application with different data because the view is simply handling how the data is being displayed to the user.

If you're new to back-end web development read about the MVC architectural pattern first, a good start is that [wikipedia article](https://en.wikipedia.org/wiki/Model%E2%80%93view%E2%80%93controller).

Follow the examples below,

- [Hello world](mvc/hello-world/main.go) **UPDATED**
- [Session Controller](mvc/session-controller/main.go) **UPDATED**
- [Overview - Plus Repository and Service layers](mvc/overview) **UPDATED**
- [Login showcase - Plus Repository and Service layers](mvc/login) **UPDATED**
- [Singleton](mvc/singleton) **NEW**
- [Websocket Controller](mvc/websocket) **NEW**
- [Register Middleware](mvc/middleware) **NEW**
- [Vue.js Todo MVC](tutorial/vuejs-todo-mvc) **NEW**

### Subdomains

- [Single](subdomains/single/main.go)
- [Multi](subdomains/multi/main.go)
- [Wildcard](subdomains/wildcard/main.go)
- [WWW](subdomains/www/main.go)
- [Redirect fast](subdomains/redirect/main.go)

### Convert `http.Handler/HandlerFunc`

- [From func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)](convert-handlers/negroni-like/main.go)
- [From http.Handler or http.HandlerFunc](convert-handlers/nethttp/main.go)
- [From func(http.HandlerFunc) http.HandlerFunc](convert-handlers/real-usecase-raven/writing-middleware/main.go)

### View

| Engine | Declaration |
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

### Authentication

- [Basic Authentication](authentication/basicauth/main.go)
- [OAUth2](authentication/oauth2/main.go)
- [JWT](experimental-handlers/jwt/main.go)
- [Sessions](#sessions)

### File Server

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

### Miscellaneous

- [Request Logger](http_request/request-logger/main.go)
    * [log requests to a file](http_request/request-logger/request-logger-file/main.go)
- [Localization and Internationalization](miscellaneous/i18n/main.go)
- [Recovery](miscellaneous/recover/main.go)
- [Profiling (pprof)](miscellaneous/pprof/main.go)
- [Internal Application File Logger](miscellaneous/file-logger/main.go)
- [Google reCAPTCHA](miscellaneous/recaptcha/main.go) 

### Experimental Handlers

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

#### More

https://github.com/kataras/iris/tree/master/middleware#third-party-handlers

### Automated API Documentation

- [yaag](apidoc/yaag/main.go)

### Testing

The `httptest` package is your way for end-to-end HTTP testing, it uses the httpexpect library created by our friend, [gavv](https://github.com/gavv).

[Example](testing/httptest/main_test.go)

### Caching

iris cache library lives on its own [package](https://github.com/kataras/iris/tree/master/cache).

- [Simple](cache/simple/main.go)
- [Client-Side (304)](cache/client-side/main.go) - part of the iris context core

> You're free to use your own favourite caching package if you'd like so.

### Cookies

- [Basic](cookies/basic/main.go)
- [Encode/Decode (securecookie)](cookies/securecookie/main.go)

### Sessions

iris session manager lives on its own [package](https://github.com/kataras/iris/tree/master/sessions).

- [Overview](sessions/overview/main.go)
- [Standalone](sessions/standalone/main.go)
- [Secure Cookie](sessions/securecookie/main.go)
- [Flash Messages](sessions/flash-messages/main.go)
- [Databases](sessions/database)
    * [Badger](sessions/database/badger/main.go)
    * [BoltDB](sessions/database/boltdb/main.go)
    * [Redis](sessions/database/redis/main.go)

> You're free to use your own favourite sessions package if you'd like so.

### Websockets

iris websocket library lives on its own [package](https://github.com/kataras/iris/tree/master/websocket).

The package is designed to work with raw websockets although its API is similar to the famous [socket.io](https://socket.io). I have read an article recently and I felt very contented about my decision to design a **fast** websocket-**only** package for Iris and not a backwards socket.io-like package. You can read that article by following this link: https://medium.com/@ivanderbyl/why-you-don-t-need-socket-io-6848f1c871cd.

- [Chat](websocket/chat/main.go)
- [Native Messages](websocket/native-messages/main.go)
- [Connection List](websocket/connectionlist/main.go)
- [TLS Enabled](websocket/secure/main.go)
- [Custom Raw Go Client](websocket/custom-go-client/main.go)
- [Third-Party socket.io](websocket/third-party-socketio/main.go)

> You're free to use your own favourite websockets package if you'd like so.

### Typescript Automation Tools

typescript automation tools have their own repository: [https://github.com/kataras/iris/tree/master/typescript](https://github.com/kataras/iris/tree/master/typescript) **it contains examples**

> I'd like to tell you that you can use your favourite but I don't think you will find such a thing anywhere else.

### Hey, You

Developers should read the [godocs](https://godoc.org/github.com/kataras/iris) and https://docs.iris-go.com for a better understanding.

Psst, I almost forgot; do not forget to [star or watch](https://github.com/kataras/iris/stargazers) the project in order to stay updated with the latest tech trends, it never takes more than a second!
