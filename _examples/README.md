# Examples

Please do learn how [net/http](https://golang.org/pkg/net/http/) std package works, first.

This folder provides easy to understand code snippets on how to get started with [iris](https://github.com/kataras/iris) micro web framework.

It doesn't always contain the "best ways" but it does cover each important feature that will make you so excited to GO with iris!

### Overview

- [Hello world!](hello-world/main.go)
- [Glimpse](overview/main.go)
- [Tutorial: Online Visitors](tutorial/online-visitors/main.go)
- [Tutorial: URL Shortener using BoltDB](https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7)
- [Tutorial: How to turn your Android Device into a fully featured Web Server (**MUST**)](https://twitter.com/ThePracticalDev/status/892022594031017988)

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
- [Controllers](routing/mvc)
- [Custom HTTP Errors](routing/http-errors/main.go)
- [Dynamic Path](routing/dynamic-path/main.go)
    * [root level wildcard path](routing/dynamic-path/root-wildcard/main.go)
- [Reverse routing](routing/reverse/main.go)
- [Custom wrapper](routing/custom-wrapper/main.go)
- Custom Context
    * [method overriding](routing/custom-context/method-overriding/main.go)
    * [new implementation](routing/custom-context/new-implementation/main.go)
- [Route State](routing/route-state/main.go)

### Subdomains

- [Single](subdomains/single/main.go)
- [Multi](subdomains/multi/main.go)
- [Wildcard](subdomains/wildcard/main.go)
- [WWW](subdomains/www/main.go) 

### Convert `http.Handler/HandlerFunc`

- [From func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)](convert-handlers/negroni-like/main.go)
- [From http.Handler or http.HandlerFunc](convert-handlers/nethttp/main.go)

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


You can serve [quicktemplate](https://github.com/valyala/quicktemplate) files too, simply by using the `context#ResponseWriter`, take a look at the [http_responsewriter/quicktemplate](http_responsewriter/quicktemplate) example.

### Authentication

- [Basic Authentication](authentication/basicauth/main.go)
- [OAUth2](authentication/oauth2/main.go)
- [JWT](https://github.com/iris-contrib/middleware/blob/master/jwt/_example/main.go)
- [Sessions](#sessions)

### File Server

- [Favicon](file-server/favicon/main.go)
- [Basic](file-server/basic/main.go)
- [Embedding Files Into App Executable File](file-server/embedding-files-into-app/main.go)
- [Send/Force-Download Files](file-server/send-files/main.go)
- Single Page Applications
    * [single Page Application](file-server/single-page-application/basic/main.go)
    * [embedded Single Page Application](file-server/single-page-application/embedded-single-page-application/main.go)

### How to Read from `context.Request() *http.Request`

- [Bind JSON](http_request/read-json/main.go)
- [Bind Form](http_request/read-form/main.go)
- [Upload/Read Files](http_request/upload-files/main.go)

> The `context.Request()` returns the same *http.Request you already know, these examples show some places where the  Context uses this object. Besides that you can use it as you did before iris.

### How to Write to `context.ResponseWriter() http.ResponseWriter`

- [Write `valyala/quicktemplate` templates](http_responsewriter/quicktemplate)
- [Text, Markdown, HTML, JSON, JSONP, XML, Binary](http_responsewriter/write-rest/main.go)
- [Stream Writer](http_responsewriter/stream-writer/main.go)
- [Transactions](http_responsewriter/transactions/main.go)

> The `context.ResponseWriter()` returns an enchament version of a http.ResponseWriter, these examples show some places where the Context uses this object. Besides that you can use it as you did before iris.

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

#### More

https://github.com/kataras/iris/tree/master/middleware#third-party-handlers

### Testing

The `httptest` package is your way for end-to-end HTTP testing, it uses the httpexpect library created by our friend, [gavv](https://github.com/gavv).

[Example](testing/httptest/main_test.go)

### Caching

iris cache library lives on its own [package](https://github.com/kataras/iris/tree/master/cache).

- [Simple](cache/simple/main.go)

> You're free to use your own favourite caching package if you'd like so.

### Sessions

iris session manager lives on its own [package](https://github.com/kataras/iris/tree/master/sessions).

- [Overview](sessions/overview/main.go)
- [Standalone](sessions/standalone/main.go)
- [Secure Cookie](sessions/securecookie/main.go)
- [Flash Messages](sessions/flash-messages/main.go)
- [Databases](sessions/database)
    * [File](sessions/database/file/main.go)
    * [BoltDB](sessions/database/boltdb/main.go)
    * [Redis](sessions/database/redis/main.go)

> You're free to use your own favourite sessions package if you'd like so.

### Websockets

iris websocket library lives on its own [package](https://github.com/kataras/iris/tree/master/websocket).

- [Chat](websocket/chat/main.go)
- [Native Messages](websocket/native-messages/main.go)
- [Connection List](websocket/connectionlist/main.go)
- [TLS Enabled](websocket/secure/main.go)
- [Custom Raw Go Client](websocket/custom-go-client/main.go)

> You're free to use your own favourite websockets package if you'd like so.

### Typescript Automation Tools

typescript automation tools have their own repository: [https://github.com/kataras/iris/tree/master/typescript](https://github.com/kataras/iris/tree/master/typescript) **it contains examples**

> I'd like to tell you that you can use your favourite but I don't think you will find such a thing anywhere else.

### Hey, You!

Developers should read the [godocs](https://godoc.org/github.com/kataras/iris) for a better understanding.

Psst, I almost forgot; do not forget to [star or watch](https://github.com/kataras/iris/stargazers) the project in order to stay updated with the latest tech trends, it never takes more than a second!


