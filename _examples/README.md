# Examples

Please do learn how [net/http](https://golang.org/pkg/net/http/) std package works, first.

This folder provides easy to understand code snippets on how to get started with [iris](https://github.com/kataras/iris) web framework.

It doesn't always contain the "best ways" but it does cover each important feature that will make you so excited to GO with iris!

## Running the examples

1. Install the Go Programming Language, version 1.12+ from https://golang.org/dl.
2. [Install Iris](https://github.com/kataras/iris/wiki/installation)
3. Install any external packages that required by the examples

<details>
<summary>External packages</summary>

```sh
cd _examples && go get ./...
```

</details>

And run each example you wanna see, e.g.

```sh
$ cd $GOPATH/src/github.com/kataras/iris/_examples/overview
$ go run main.go
```

> Test the examples by opening a terminal window and execute: `go test -v ./...`

### Overview

- [Hello world!](hello-world/main.go)
- [Docker](docker/README.md)
- [Hello WebAssembly!](webassembly/basic/main.go)
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
- [Tutorial: API for Apache Kafka](tutorial/api-for-apache-kafka)

### Structuring

Nothing stops you from using your favorite folder structure. Iris is a low level web framework, it has got MVC first-class support but it doesn't limit your folder structure, this is your choice.

Structuring depends on your own needs. We can't tell you how to design your own application for sure but you're free to take a closer look to the examples below; you may find something useful that you can borrow for your app;

- [Bootstrapper](structuring/bootstrap)
- [MVC with Repository and Service layer Overview](structuring/mvc-plus-repository-and-service-layers)
- [Login (MVC with Single Responsibility package)](structuring/login-mvc-single-responsibility-package)
- [Login (MVC with Datamodels, Datasource, Repository and Service layer)](structuring/login-mvc)

### HTTP Listening

- [Common, with address](http-listening/listen-addr/main.go)
    * [public domain address](http-listening/listen-addr-public/main.go)
    * [omit server errors](http-listening/listen-addr/omit-server-errors/main.go)
- [UNIX socket file](http-listening/listen-unix/main.go)
- [TLS](http-listening/listen-tls/main.go)
- [Letsencrypt (Automatic Certifications)](http-listening/listen-letsencrypt/main.go)
- [Notify on shutdown](http-listening/notify-on-shutdown/main.go)
- Custom TCP Listener
    * [common net.Listener](http-listening/custom-listener/main.go)
    * [SO_REUSEPORT for unix systems](http-listening/custom-listener/unix-reuseport/main.go)
- Custom HTTP Server
    * [HTTP/3 Quic](http-listening/http3-quic)
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
- [Not Found - Suggest Closest Paths](routing/not-found-suggests/main.go) **NEW**
- [Dynamic Path](routing/dynamic-path/main.go)
    * [root level wildcard path](routing/dynamic-path/root-wildcard/main.go)
- [Write your own custom parameter types](routing/macros/main.go)
- [Reverse routing](routing/reverse/main.go)
- [Custom Router (high-level)](routing/custom-high-level-router/main.go)
- [Custom Wrapper](routing/custom-wrapper/main.go)
- Custom Context
    * [method overriding](routing/custom-context/method-overriding/main.go)
    * [new implementation](routing/custom-context/new-implementation/main.go)
- [Route State](routing/route-state/main.go)
- [Writing a middleware](routing/writing-a-middleware)
    * [per-route](routing/writing-a-middleware/per-route/main.go)
    * [globally](routing/writing-a-middleware/globally/main.go)

### Versioning

- [How it works](https://github.com/kataras/iris/blob/master/versioning/README.md)
- [Example](versioning/main.go)

### Dependency Injection

- [Basic](hero/basic/main.go)
- [Overview](hero/overview)
- [Sessions](hero/sessions)
- [Yet another dependency injection example and good practises at general](hero/smart-contract/main.go)

### MVC

- [Hello world](mvc/hello-world/main.go)
- [Regexp](mvc/regexp/main.go)
- [Session Controller](mvc/session-controller/main.go)
- [Overview - Plus Repository and Service layers](mvc/overview)
- [Login showcase - Plus Repository and Service layers](mvc/login)
- [Singleton](mvc/singleton)
- [Websocket Controller](mvc/websocket)
- [Register Middleware](mvc/middleware)
- [Vue.js Todo MVC](tutorial/vuejs-todo-mvc)

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
- [Jet](/view/template_jet_0)
- [Jet Embedded](view/template_jet_1_embedded)

You can serve [quicktemplate](https://github.com/valyala/quicktemplate) and [hero templates](https://github.com/shiyanhui/hero/hero) files too, simply by using the `context#ResponseWriter`, take a look at the [http_responsewriter/quicktemplate](http_responsewriter/quicktemplate) and [http_responsewriter/herotemplate](http_responsewriter/herotemplate) examples.

### Localization and Internationalization

- [I18n](i18n/main.go) **NEW**

### Sitemap

- [Sitemap](sitemap/main.go) **NEW**

### Desktop App

- [Using blink package](desktop-app/blink) **NEW**
- [Using lorca package](desktop-app/lorca) **NEW**
- [Using webview package](desktop-app/webview) **NEW**

### Authentication

- [Basic Authentication](authentication/basicauth/main.go)
- [OAUth2](authentication/oauth2/main.go)
- [Request Auth(JWT)](experimental-handlers/jwt/main.go)
- [Sessions](#sessions)

### File Server

- [Favicon](file-server/favicon/main.go)
- [Basic](file-server/basic/main.go)
- [Embedding Files Into App Executable File](file-server/embedding-files-into-app/main.go)
- [Embedding Gziped Files Into App Executable File](file-server/embedding-gziped-files-into-app/main.go)
- [Send/Force-Download Files](file-server/send-files/main.go)
- Single Page Applications
    * [single Page Application](file-server/single-page-application/basic/main.go)
    * [embedded Single Page Application](file-server/single-page-application/embedded-single-page-application/main.go)
    * [embedded Single Page Application with other routes](file-server/single-page-application/embedded-single-page-application-with-other-routes/main.go)

### How to Read from `context.Request() *http.Request`

- [Read JSON](http_request/read-json/main.go)
    * [Struct Validation](http_request/read-json-struct-validation/main.go)
- [Read XML](http_request/read-xml/main.go)
- [Read YAML](http_request/read-yaml/main.go)
- [Read Form](http_request/read-form/main.go)
- [Read Query](http_request/read-query/main.go)
- [Read Custom per type](http_request/read-custom-per-type/main.go)
- [Read Custom via Unmarshaler](http_request/read-custom-via-unmarshaler/main.go)
- [Read Many times](http_request/read-many/main.go)
- [Upload/Read File](http_request/upload-file/main.go)
- [Upload multiple files with an easy way](http_request/upload-files/main.go)
- [Extract referrer from "referer" header or URL query parameter](http_request/extract-referer/main.go)

> The `context.Request()` returns the same *http.Request you already know, these examples show some places where the  Context uses this object. Besides that you can use it as you did before iris.

### How to Write to `context.ResponseWriter() http.ResponseWriter`

- [Content Negotiation](http_responsewriter/content-negotiation)
- [Write `valyala/quicktemplate` templates](http_responsewriter/quicktemplate)
- [Write `shiyanhui/hero` templates](http_responsewriter/herotemplate)
- [Text, Markdown, HTML, JSON, JSONP, XML, Binary](http_responsewriter/write-rest/main.go)
- [Write Gzip](http_responsewriter/write-gzip/main.go)
- [Stream Writer](http_responsewriter/stream-writer/main.go)
- [Transactions](http_responsewriter/transactions/main.go)
- [SSE](http_responsewriter/sse/main.go)
- [SSE (third-party package usage for server sent events)](http_responsewriter/sse-third-party/main.go)

> The `context/context#ResponseWriter()` returns an enchament version of a http.ResponseWriter, these examples show some places where the Context uses this object. Besides that you can use it as you did before iris.

### ORM

- [Using xorm(Mysql, MyMysql, Postgres, Tidb, **SQLite**, MsSql, MsSql, Oracle)](orm/xorm/main.go)
- [Using gorm](orm/gorm/main.go)

### Miscellaneous

- [HTTP Method Override](https://github.com/kataras/iris/blob/master/middleware/methodoverride/methodoverride_test.go)
- [Request Logger](http_request/request-logger/main.go)
    * [log requests to a file](http_request/request-logger/request-logger-file/main.go)
- [Recovery](miscellaneous/recover/main.go)
- [Profiling (pprof)](miscellaneous/pprof/main.go)
- [Internal Application File Logger](miscellaneous/file-logger/main.go)
- [Google reCAPTCHA](miscellaneous/recaptcha/main.go) 

### Community-based Handlers

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
- [Middleware](sessions/middleware/main.go)
- [Secure Cookie](sessions/securecookie/main.go)
- [Flash Messages](sessions/flash-messages/main.go)
- [Databases](sessions/database)
    * [Badger](sessions/database/badger/main.go)
    * [BoltDB](sessions/database/boltdb/main.go)
    * [Redis](sessions/database/redis/main.go)

> You're free to use your own favourite sessions package if you'd like so.

### Websockets

- [Basic](websocket/basic)
    * [Server](websocket/basic/server.go)
    * [Go Client](websocket/basic/go-client/client.go)
    * [Browser Client](websocket/basic/browser/index.html)
    * [Browser NPM Client (browserify)](websocket/basic/browserify/app.js)
- [Native Messages](websocket/native-messages/main.go)
- [TLS Enabled](websocket/secure/README.md)

### Typescript Automation Tools

typescript automation tools have their own repository: [https://github.com/kataras/iris/tree/master/typescript](https://github.com/kataras/iris/tree/master/typescript) **it contains examples**

> I'd like to tell you that you can use your favourite but I don't think you will find such a thing anywhere else.

### Hey, You

Developers should read the [godocs](https://godoc.org/github.com/kataras/iris) and https://docs.iris-go.com for a better understanding.

Psst, I almost forgot; do not forget to [star or watch](https://github.com/kataras/iris/stargazers) the project in order to stay updated with the latest tech trends, it never takes more than a second!
