<p align="center">


 <a href="https://www.gitbook.com/book/kataras/iris/details"><img  width="600" src="https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/book/cover_6_flat_alpha.png"></a>

<br/>

<a href="https://travis-ci.org/kataras/iris"><img src="https://img.shields.io/travis/kataras/iris.svg?style=flat-square" alt="Build Status"></a>

<a href="https://github.com/avelino/awesome-go"><img src="https://img.shields.io/badge/awesome-%E2%9C%93-ff69b4.svg?style=flat-square" alt="https://cdn.rawgit.com/sindresorhus/awesome/d7305f38d29fed78fa85652e3a63e154dd8e8829/media/badge.svg"></a>

<a href="#"><img src="https://img.shields.io/badge/platform-Any-ec2eb4.svg?style=flat-square" alt="Platforms"></a>

<a href="https://github.com/kataras/iris/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0%20%20-E91E63.svg?style=flat-square" alt="License"></a>


<a href="https://golang.org"><img src="https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square" alt="Built with GoLang"></a>

<br/>


<a href="https://github.com/kataras/iris/releases"><img src="https://img.shields.io/badge/%20version%20-%204.2.7%20-blue.svg?style=flat-square" alt="Releases"></a>

<a href="https://github.com/iris-contrib/examples"><img src="https://img.shields.io/badge/%20examples-repository-3362c2.svg?style=flat-square" alt="Examples"></a>

<a href="https://www.gitbook.com/book/kataras/iris/details"><img src="https://img.shields.io/badge/%20docs-reference-5272B4.svg?style=flat-square" alt="Practical Guide/Docs"></a>

<a href="https://kataras.rocket.chat/channel/iris"><img src="https://img.shields.io/badge/%20community-chat-00BCD4.svg?style=flat-square" alt="Chat"></a><br/><br/>


The <a href="https://github.com/kataras/iris#benchmarks">fastest</a> backend web framework for Go.
<br/>
Easy to <a href="https://www.gitbook.com/book/kataras/iris/details">learn</a>,  while it's highly customizable. <br/>
Ideally suited for both experienced and novice <b>Developers</b>.
<br/>
<br/>

<img src="https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png" href="#benchmarks" alt="Benchmark Wizzard July 21, 2016- Processing Time Horizontal Graph" />

</p>




Quick Look
------------

```go
package main

import "github.com/kataras/iris"

func main() {
  // serve static files, just a fav here
  iris.Favicon("./favicon.ico")

  // handle "/" - HTTP METHOD: "GET"
  iris.Get("/", func(ctx *iris.Context) {
    ctx.Render("index.html")
  })

  iris.Get("/login", func(ctx *iris.Context) {
    ctx.Render("login.html", iris.Map{"Title": "Login Page"})
  })

  // handle "/login" - HTTP METHOD: "POST"
  iris.Post("/login", func(ctx *iris.Context) {
    secret := ctx.PostValue("secret")
    ctx.Session().Set("secret", secret)

    ctx.Redirect("/user")
  })

  // handle websocket connections
  iris.Config.Websocket.Endpoint = "/mychat"
  iris.Websocket.OnConnection(func(c iris.WebsocketConnection) {
    c.Join("myroom")

    c.On("chat", func(message string){
      c.To("myroom").Emit("chat", "From "+c.ID()+": "+message)
    })
  })

  // serve requests at http://localhost:8080
  iris.Listen(":8080")
}
```

Installation
------------
The only requirement is the [Go Programming Language](https://golang.org/dl), at least v1.7.

```bash
$ go get -u github.com/kataras/iris/iris
```

> If you have installation issues or you are connected to the Internet through China please, [click here](https://kataras.gitbooks.io/iris/content/install.html).



Docs & Community
------------

 <a href="https://www.gitbook.com/book/kataras/iris/details"><img align="right" width="185" src="https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/book/cover_4.jpg"></a>


 - The most important is to read [the practical guide](https://www.gitbook.com/book/kataras/iris/details).

 - Explore & download the [examples](https://github.com/iris-contrib/examples).

 - [HISTORY.md](https://github.com//kataras/iris/tree/master/HISTORY.md) file is your best friend.


If you'd like to discuss this package, or ask questions about it, feel free to

 * Post an issue or  idea [here](https://github.com/kataras/iris/issues).
 * [Chat][Chat].


New website-docs & logo have been designed by the community[*](https://github.com/kataras/iris/issues/153)

- Website created by [@kujtimiihoxha](https://github.com/kujtimiihoxha)
- Logo designed by [@OneebMalik](https://github.com/OneebMalik)


Features
------------
- Focus on high performance
- Robust routing, static, wildcard subdomains and routes.
- [Websocket API](https://github.com/kataras/go-websocket), [Sessions](https://github.com/kataras/go-sessions) support out of the box
- Remote control through [SSH](https://github.com/iris-contrib/examples/blob/master/ssh/main.go)
- View system supporting [6+](https://github.com/kataras/go-template) template engines.[*](https://kataras.gitbooks.io/iris/content/template-engines.html)
- Highly scalable response engines with pre-defined [serializers](https://github.com/kataras/go-serializer)
- Live reload
- Typescript integration + Online editor
- OAuth, OAuth2 supporting  27+ API providers, JWT, BasicAuth
- and many other surprises

<img src="https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/arrowdown.png" width="72"/>


| Name        | Description           | Usage  |
| ------------------|:---------------------:|-------:|
| [JSON ](https://github.com/kataras/go-serializer/tree/master/json)      | JSON Serializer (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/serialize_engines/json_1/main.go),[example 2](https://github.com/iris-contrib/examples/blob/master/serialize_engines/json_2/main.go), [book section](https://kataras.gitbooks.io/iris/content/serialize-engines.html)
| [JSONP ](https://github.com/kataras/go-serializer/tree/master/jsonp)      | JSONP Serializer (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/serialize_engines/jsonp_1/main.go),[example 2](https://github.com/iris-contrib/examples/blob/master/serialize_engines/jsonp_2/main.go), [book section](https://kataras.gitbooks.io/iris/content/serialize-engines.html)
| [XML ](https://github.com/kataras/go-serializer/tree/master/xml)      | XML Serializer (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/serialize_engines/xml_1/main.go),[example 2](https://github.com/iris-contrib/examples/blob/master/serialize_engines/xml_2/main.go), [book section](https://kataras.gitbooks.io/iris/content/serialize-engines.html)
| [Markdown ](https://github.com/kataras/go-serializer/tree/master/markdown)      | Markdown Serializer (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/serialize_engines/markdown_1/main.go),[example 2](https://github.com/iris-contrib/examples/blob/master/serialize_engines/markdown_2/main.go), [book section](https://kataras.gitbooks.io/iris/content/serialize-engines.html)
| [Text](https://github.com/kataras/go-serializer/tree/master/text)      | Text Serializer (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/serialize_engines/text_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/serialize-engines.html)
| [Binary Data ](https://github.com/kataras/go-serializer/tree/master/data)      | Binary Data Serializer (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/serialize_engines/data_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/serialize-engines.html)
| [HTML/Default Engine ](https://github.com/kataras/go-template/tree/master/html)      | HTML Template Engine (Default)                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_html_0/main.go), [book section](https://kataras.gitbooks.io/iris/content/template-engines.html)
| [Django Engine ](https://github.com/kataras/go-template/tree/master/django)      | Django Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_django_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/template-engines.html)
| [Pug/Jade Engine ](https://github.com/kataras/go-template/tree/master/pug)      | Pug Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_pug_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/template-engines.html)
| [Handlebars Engine ](https://github.com/kataras/go-template/tree/master/handlebars)      | Handlebars Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_handlebars_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/template-engines.html)
| [Amber Engine ](https://github.com/kataras/go-template/tree/master/amber)      | Amber Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_amber_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/template-engines.html)
| [Markdown Engine ](https://github.com/kataras/go-template/tree/master/markdown)      | Markdown Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_markdown_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/template-engines.html)
| [Basicauth Middleware ](https://github.com/iris-contrib/middleware/tree/master/basicauth)      | HTTP Basic authentication                  |[example 1](https://github.com/iris-contrib/examples/blob/master/middleware_basicauth_1/main.go), [example 2](https://github.com/iris-contrib/examples/blob/master/middleware_basicauth_2/main.go), [book section](https://kataras.gitbooks.io/iris/content/basic-authentication.html)  |
| [JWT Middleware ](https://github.com/iris-contrib/middleware/tree/master/jwt)      | JSON Web Tokens                  |[example ](https://github.com/iris-contrib/examples/blob/master/middleware_jwt/main.go), [book section](https://kataras.gitbooks.io/iris/content/jwt.html)  |
| [Cors Middleware ](https://github.com/iris-contrib/middleware/tree/master/cors)      | Cross Origin Resource Sharing W3 specification   | [how to use ](https://github.com/iris-contrib/middleware/tree/master/cors#how-to-use)  |
| [Secure Middleware ](https://github.com/iris-contrib/middleware/tree/master/secure) |  Facilitates some quick security wins      | [example](https://github.com/iris-contrib/examples/blob/master/middleware_secure/main.go)  |
| [I18n Middleware ](https://github.com/iris-contrib/middleware/tree/master/i18n)      | Simple internationalization       | [example](https://github.com/iris-contrib/examples/tree/master/middleware_internationalization_i18n), [book section](https://kataras.gitbooks.io/iris/content/middleware-internationalization-and-localization.html)  |
| [Recovery Middleware ](https://github.com/iris-contrib/middleware/tree/master/recovery) | Safety recover the station from panic       | [example](https://github.com/iris-contrib/examples/blob/master/middleware_recovery/main.go)  |
| [Logger Middleware ](https://github.com/iris-contrib/middleware/tree/master/logger)      | Logs every request       | [example](https://github.com/iris-contrib/examples/blob/master/middleware_logger/main.go), [book section](https://kataras.gitbooks.io/iris/content/logger.html)  |
| [Editor Plugin](https://github.com/iris-contrib/plugin/tree/master/editor)      | Alm-tools, a typescript online IDE/Editor | [book section](https://kataras.gitbooks.io/iris/content/plugin-editor.html) |
| [Typescript Plugin](https://github.com/iris-contrib/plugin/tree/master/typescript)      | Auto-compile client-side typescript files      |   [book section](https://kataras.gitbooks.io/iris/content/plugin-typescript.html) |
| [OAuth,OAuth2 Plugin](https://github.com/iris-contrib/plugin/tree/master/oauth) |  User Authentication was never be easier, supports >27 providers |    [example](https://github.com/iris-contrib/examples/tree/master/plugin_oauth_oauth2), [book section](https://kataras.gitbooks.io/iris/content/plugin-oauth.html) |
| [Iris control Plugin](https://github.com/iris-contrib/plugin/tree/master/iriscontrol) |   Basic (browser-based) control over your Iris station |    [example](https://github.com/iris-contrib/examples/blob/master/plugin_iriscontrol/main.go), [book section](https://kataras.gitbooks.io/iris/content/plugin-iriscontrol.html) |


FAQ
------------
Explore [these questions](https://github.com/kataras/iris/issues?q=label%3Aquestion) or navigate to the [community chat][Chat].

Philosophy
------------

The Iris philosophy is to provide robust tooling for HTTP, making it a great solution for single page applications, web sites, hybrids, or public HTTP APIs.

Iris does not force you to use any specific ORM or template engine. With support for the most used template engines, you can quickly craft the perfect application.

Iris is built on top of fasthttp (http basic layer), net/http middleware will not work by default on Iris, but you can convert any net/http middleware to Iris, see [middleware](https://github.com/iris-contrib/middleware) repository to see how.

If for any personal reasons you think that Iris+fasthttp is not suitable for you, but you don't want to miss the unique features that Iris provides, you can take a look at the HTTP2 [Q web framework](https://github.com/kataras/q).

## Benchmarks

This Benchmark test aims to compare the whole HTTP request processing between Go web frameworks.


![Benchmark Wizzard July 21, 2016- Processing Time Horizontal Graph](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

**The results have been updated on July 21, 2016**

Testing
------------

Community should write third-party or iris base tests to the [iris-contrib/tests repository](https://github.com/iris-contrib/tests).
I recommend writing your API tests using this new library, [httpexpect](https://github.com/gavv/httpexpect) which supports Iris and fasthttp now, after my request [here](https://github.com/gavv/httpexpect/issues/2).

Versioning
------------

Current: **v4.2.7**

>  Iris is an active project

Read more about Semantic Versioning 2.0.0

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions

Todo
------------
- [x] Use of the standard `log.Logger` instead of the `iris-contrib/logger`(colorful logger), make these changes to all middleware, examples and plugins.
- [x] Implement, even, a better way to manage configuration/options, devs will be able to set their own custom options inside there. ` I'm thinking of something the last days, but it will have breaking changes. `
- [x] Implement an internal updater, as requested [here](https://github.com/kataras/iris/issues/401).

Iris is a **Community-Driven** Project, waiting for your suggestions and [feature requests](https://github.com/kataras/iris/issues?utf8=%E2%9C%93&q=label%3A%22feature%20request%22)!

I, as the author of this package, am working full time on this package, no time to any other job, so
if you're **willing to donate** and you can **afford it** please click [here](DONATIONS.md), thank you!

People
------------
The big thanks goes to [all people](https://github.com/kataras/iris/issues?utf8=%E2%9C%93&q=label%3A%22feature+request%22) who help building this framework with feature-requests & bug reports!

The author of Iris is [@kataras](https://github.com/kataras).


Contributing
------------
If you are interested in contributing to the Iris project, please see the document [CONTRIBUTING](https://github.com/kataras/iris/blob/master/CONTRIBUTING.md).

License
------------

This project is licensed under the Apache License, Version 2.0.

License can be found [here](LICENSE).

[Travis Widget]: https://img.shields.io/travis/kataras/iris.svg?style=flat-square
[Travis]: http://travis-ci.org/kataras/iris
[License Widget]: https://img.shields.io/badge/license-Apache%202.0%20%20-E91E63.svg?style=flat-square
[License]: https://github.com/kataras/iris/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/release-v4.2.7-blue.svg?style=flat-square
[Release]: https://github.com/kataras/iris/releases
[Chat Widget]: https://img.shields.io/badge/community-chat-00BCD4.svg?style=flat-square
[Chat]: https://kataras.rocket.chat/channel/iris
[ChatMain]: https://kataras.rocket.chat/channel/iris
[ChatAlternative]: https://gitter.im/kataras/iris
[Report Widget]: https://img.shields.io/badge/report%20card-A%2B-F44336.svg?style=flat-square
[Report]: http://goreportcard.com/report/kataras/iris
[Documentation Widget]: https://img.shields.io/badge/documentation-reference-5272B4.svg?style=flat-square
[Documentation]: https://www.gitbook.com/book/kataras/iris/details
[Examples Widget]: https://img.shields.io/badge/examples-repository-3362c2.svg?style=flat-square
[Examples]: https://github.com/iris-contrib/examples
[Language Widget]: https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square
[Language]: http://golang.org
[Platform Widget]: https://img.shields.io/badge/platform-Any--OS-gray.svg?style=flat-square
