# Examples

Please do learn how [net/http](https://golang.org/pkg/net/http/) std package works, first.

This folder provides easy to understand code snippets on how to get started with [iris](https://github.com/kataras/iris) micro web framework.

It doesn't always contain the "best ways" but it does cover each important feature that will make you so excited to GO with iris!

### Overview

- [Hello world!](hello-world/main.go)
- [Glimpse](overview/main.go)
- [Tutorial: Online Visitors](tutorial/online-visitors/main.go)
- [Tutorial: URL Shortener using BoltDB](tutorial/url-shortener/main.go)

### HTTP Listening 

- [Common, with address](http-listening/listen-addr/main.go)
    * [omit server errors](http-listening/listen-addr/omit-server-errors/main.go)
- [UNIX socket file](http-listening/listen-unix/main.go)
- [TLS](http-listening/listen-tls/main.go)
- [Letsencrypt (Automatic Certifications)](http-listening/listen-letsencrypt/main.go)
- Custom TCP Listener
    * [common net.Listener](http-listening/custom-listener/main.go)
    * [SO_REUSEPORT for unix systems](http-listening/custom-listener/unix-reuseport/main.go)
- Custom HTTP Server
    * [iris way](http-listening/custom-httpserver/easy-way/main.go)
    * [std way](http-listening/custom-httpserver/std-way/main.go)
    * [multi server instances](http-listening/custom-httpserver/multi/main.go)
- Graceful Shutdown
    * [using the `RegisterOnInterrupt`](http-listening/graceful-shutdown/default-notifier/main.go)
    * [using a custom notifier](http-listening/graceful-shutdown/custom-notifier/main.go)
   
### Configuration

- [Functional](configuration/functional/main.go)
* [From Configuration Struct](configuration/from-configuration-structure/main.go)
- [Import from YAML file](configuration/from-yaml-file/main.go)
- [Import from TOML file](configuration/from-toml-file/main.go)


### Routing, Grouping, Dynamic Path Parameters, "Macros" and Custom Context

- [Overview](routing/overview/main.go)
- [Basic](routing/basic/main.go)
- [Custom HTTP Errors](routing/http-errors/main.go)
- [Dynamic Path](routing/dynamic-path/main.go)
- [Reverse routing](routing/reverse/main.go)
- [Custom wrapper](routing/custom-wrapper/main.go)
- Custom Context
    * [Method Overriding](routing/custom-context/method-overriding/main.go)
    * [New Implementation](routing/custom-context/new-implementation/main.go)
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
    * [Single Page Application](file-server/single-page-application/basic/main.go)
    * [Embedded Single Page Application](file-server/single-page-application/embedded-single-page-application/main.go)

### How to Read from `context.Request() *http.Request`

- [Bind JSON](http_request/read-json/main.go)
- [Bind Form](http_request/read-form/main.go)
- [Upload/Read Files](http_request/upload-files/main.go)

> The `context.Request()` returns the same *http.Request you already know, these examples show some places where the  Context uses this object. Besides that you can use it as you did before iris.

### How to Write to `context.ResponseWriter() http.ResponseWriter`

- [Text, Markdown, HTML, JSON, JSONP, XML, Binary](http_responsewriter/write-rest/main.go)
- [Stream Writer](http_responsewriter/stream-writer/main.go)
- [Transactions](http_responsewriter/transactions/main.go)

> The `context.ResponseWriter()` returns an enchament version of a http.ResponseWriter, these examples show some places where the Context uses this object. Besides that you can use it as you did before iris.

### Miscellaneous

- [Request Logger](http_request/request-logger/main.go)
- [Localization and Internationalization](miscellaneous/i18n/main.go)
- [Recovery](miscellaneous/recover/main.go)
- [Profiling (pprof)](miscellaneous/pprof/main.go)
- [Internal Application File Logger](miscellaneous/file-logger/main.go)

#### More

https://github.com/kataras/iris/tree/master/middleware#third-party-handlers

### Testing

The `httptest` package is your way for end-to-end HTTP testing, it uses the httpexpect library created by our friend, [gavv](https://github.com/gavv).

[Example](testing/httptest/main_test.go)

### Caching

iris cache library lives on its own package: [https://github.com/kataras/iris/tree/master/cache](https://github.com/kataras/iris/tree/master/cache) **it contains examples**

### Sessions

iris session manager lives on its own package: [https://github.com/kataras/iris/tree/master/sessions](https://github.com/kataras/iris/tree/master/sessions) **it contains examples**

> You're free to use your own favourite sessions package if you'd like so.

### Websockets

iris websocket library lives on its own package: [https://github.com/kataras/iris/tree/master/websocket](https://github.com/kataras/iris/tree/master/websocket) **it contains examples**

> You're free to use your own favourite websockets package if you'd like so.

### Typescript Automation Tools

typescript automation tools have their own repository: [https://github.com/kataras/iris/tree/master/typescript](https://github.com/kataras/iris/tree/master/typescript) **it contains examples**

> I'd like to tell you that you can use your favourite but I don't think you will find such a thing anywhere else.

### Hey, You!

Developers should read the [godocs](https://godoc.org/github.com/kataras/iris) for a better understanding.

Psst, I almost forgot; do not forget to [star or watch](https://github.com/kataras/iris/stargazers) the project in order to stay updated with the latest tech trends, it never takes more than a second!


