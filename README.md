[![Iris Logo](https://github.com/iris-contrib/logo/raw/contribs/OneebMalik/Iris%20Logo%20with%20txt%20and%20bg.jpg)](http://iris-go.com)

[![Travis Widget]][Travis] [![Release Widget]][Release] [![Report Widget]][Report] [![License Widget]][License] [![Chat Widget]][Chat] [![Documentation Widget]][Documentation] [![Examples Widget]][Examples]

The fastest web framework for Go.

[![Alt: Benchmark Wizzard July 21, 2016- Processing Time Horizontal Graph](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)](#benchmarks)


```bash
$ cat render_json.go
```
```go
package main
import "github.com/kataras/iris"

func main() {
	iris.Get("/hi_json", func(c *iris.Context) {
		c.JSON(iris.StatusOK, iris.Map{
			"Name":  "Iris",
			"Born":  "13 March 2016",
			"Stars": 4420,
		})
	})
	iris.Listen(":8080")
}
```
> Learn more about [render](https://kataras.gitbooks.io/iris/content/render.html)

Installation
------------
The only requirement is the [Go Programming Language](https://golang.org/dl), at least v1.6

```bash
$ go get -u github.com/kataras/iris/iris
```

 >If you have installation issues or you are connected to the Internet through China please, [click here](https://kataras.gitbooks.io/iris/content/install.html).

FAQ
------------
Explore [these questions](https://github.com/kataras/iris/issues?q=label%3Aquestion) or navigate to the [community chat][Chat].


Features
------------
- Focus on high performance
- Robust routing supports static and wildcard subdomains
- View system supporting [6+](https://kataras.gitbooks.io/iris/content/template-engines.html) template engines
- Add custom [renders/responses](https://github.com/iris-contrib/examples/tree/master/response_engines/my_custom_response/main.go) or customize & replace the [defaults](https://github.com/iris-contrib/response/) (`context.JSON/JSONP/XML/Markdown/Text/Binary(...)`)
- Highly scalable Websocket API with custom events
- Sessions support with GC, memory & redis providers
- Middlewares & Plugins were never be easier
- Full REST API
- Custom HTTP Errors
- Typescript compiler + Browser-based editor
- Content negotiation & streaming
- Transport Layer Security
- [Reload](https://github.com/kataras/iris/tree/master/iris#run) on source code changes
- OAuth, OAuth2 supporting  27+ API providers
- JSON Web Tokens
- and more

<img src="https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/arrowdown.png" width="72"/>


| Name        | Description           | Usage  |
| ------------------|:---------------------:|-------:|
| [JSON ](https://github.com/iris-contrib/response/tree/master/json)      | JSON Response Engine (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/response_engines/json_1/main.go),[example 2](https://github.com/iris-contrib/examples/blob/master/response_engines/json_2/main.go), [book section](https://kataras.gitbooks.io/iris/content/response-engines.html)
| [JSONP ](https://github.com/iris-contrib/response/tree/master/jsonp)      | JSONP Response Engine (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/response_engines/jsonp_1/main.go),[example 2](https://github.com/iris-contrib/examples/blob/master/response_engines/jsonp_2/main.go), [book section](https://kataras.gitbooks.io/iris/content/response-engines.html)
| [XML ](https://github.com/iris-contrib/response/tree/master/xml)      | XML Response Engine (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/response_engines/xml_1/main.go),[example 2](https://github.com/iris-contrib/examples/blob/master/response_engines/xml_2/main.go), [book section](https://kataras.gitbooks.io/iris/content/response-engines.html)
| [Markdown ](https://github.com/iris-contrib/response/tree/master/markdown)      | Markdown Response Engine (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/response_engines/markdown_1/main.go),[example 2](https://github.com/iris-contrib/examples/blob/master/response_engines/markdown_2/main.go), [book section](https://kataras.gitbooks.io/iris/content/response-engines.html)
| [Text](https://github.com/iris-contrib/response/tree/master/text)      | Text Response Engine (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/response_engines/text_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/response-engines.html)
| [Binary Data ](https://github.com/iris-contrib/response/tree/master/data)      | Binary Data Response Engine (Default)                  |[example 1](https://github.com/iris-contrib/examples/blob/master/response_engines/data_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/response-engines.html)
| [HTML/Default Engine ](https://github.com/iris-contrib/template/tree/master/html)      | HTML Template Engine (Default)                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_html_0/main.go), [book section](https://kataras.gitbooks.io/iris/content/template-engines.html)
| [Django Engine ](https://github.com/iris-contrib/template/tree/master/django)      | Django Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_django_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/template-engines.html)
| [Pug/Jade Engine ](https://github.com/iris-contrib/template/tree/master/pug)      | Pug Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_pug_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/template-engines.html)
| [Handlebars Engine ](https://github.com/iris-contrib/template/tree/master/handlebars)      | Handlebars Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_handlebars_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/template-engines.html)
| [Amber Engine ](https://github.com/iris-contrib/template/tree/master/amber)      | Amber Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_amber_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/template-engines.html)
| [Markdown Engine ](https://github.com/iris-contrib/template/tree/master/markdown)      | Markdown Template Engine                  |[example ](https://github.com/iris-contrib/examples/blob/master/template_engines/template_markdown_1/main.go), [book section](https://kataras.gitbooks.io/iris/content/template-engines.html)
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

Docs & Community
------------

<a href="https://www.gitbook.com/book/kataras/iris/details"><img align="right" width="185" src="https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/book/cover_1.png"></a>


- Read the [book](https://www.gitbook.com/book/kataras/iris/details)

- Take a look at the [examples](https://github.com/iris-contrib/examples)

- [HISTORY](https://github.com//kataras/iris/tree/master/HISTORY.md) file is your best friend.


If you'd like to discuss this package, or ask questions about it, feel free to

* Post an issue or  idea [here](https://github.com/kataras/iris/issues)
* [Chat][Chat]

Open debates

 - [Contribute: New website and logo for Iris](https://github.com/kataras/iris/issues/153)
 - [E-book Cover - Which one you suggest?](https://github.com/kataras/iris/issues/67)

Philosophy
------------

The Iris philosophy is to provide robust tooling for HTTP, making it a great solution for single page applications, web sites, hybrids, or public HTTP APIs.

Iris does not force you to use any specific ORM or template engine. With support for the most used template engines, you can quickly craft the perfect application.

## Benchmarks

[This Benchmark suite](https://github.com/smallnest/go-web-framework-benchmark) aims to compare the whole HTTP request processing between Go web frameworks.

**New results just came out:**

![Benchmark Wizzard July 21, 2016- Processing Time Horizontal Graph](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

**The results have been updated on July 21, 2016**

[Please click here to view all detailed benchmarks.](https://github.com/smallnest/go-web-framework-benchmark/commit/4db507a22c964c9bc9774c5b31afdc199a0fe8b7)

Testing
------------

Community should write third-party or iris base tests to the [iris-contrib/tests repository](https://github.com/iris-contrib/tests).
I recommend writing your API tests using this new library, [httpexpect](https://github.com/gavv/httpexpect) which supports Iris and fasthttp now, after my request [here](https://github.com/gavv/httpexpect/issues/2).

Versioning
------------

Current: **v4.0.0-alpha.4**

>  Iris is an active project


Todo
------------
> for 'v4'

- [x] Refactor & extend view engine, separate the engines from the main code base, easier for the community to create new view engines[*](https://github.com/iris-contrib/template)
- [x] Refactor & extend sessions, split the different databases functionality to the iris-contrib[*](https://github.com/iris-contrib/sessiondb)
- [x] Refactor & extends the rest render engine in order to be able to developer to use their own implemention for rendering restful types, like, for example a  custom JSON implementation using no-standard go package for encode/decode[*](https://github.com/iris-contrib/response)
- [x] Move the iris/websocket package's source code inside iris/websocket.go one file, to be easier to use by users without import a new package[*](https://github.com/kataras/iris/blob/master/websocket.go)
- [ ] configs package should be removed after all these, we will not need big configurations because of different packages splitted & moved to the iris-contrib, we will keep interfaces and all required things inside kataras/iris.go.
- [ ] Implement all [opened community's feature requests](https://github.com/kataras/iris/issues?q=is%3Aissue+is%3Aopen+label%3A%22feature+request%22)
- [ ] Extend i18n middleware for easier and better internalization support
- [ ] Create a router as optional plugin, for optional path parts. Its name, 'ryan', taken from the community-member and donator who requested this feature
- [ ] Extend the iris control plugin
- [x] Remove deprecated functions
- [ ] Will think more :)

If you're **willing to donate** click [here](DONATIONS.md)!

People
------------
The big thanks goes to [all people](https://github.com/kataras/iris/issues?utf8=%E2%9C%93&q=label%3A%22feature+request%22) who help building this framework with feature-requests & bug reports!

The author of Iris is [@kataras](https://github.com/kataras).


Contributing
------------
If you are interested in contributing to the Iris project, please see the document [CONTRIBUTING](https://github.com/kataras/iris/blob/master/CONTRIBUTING.md).

License
------------

This project is licensed under the MIT License.

License can be found [here](LICENSE).

[Travis Widget]: https://img.shields.io/travis/kataras/iris.svg?style=flat-square
[Travis]: http://travis-ci.org/kataras/iris
[License Widget]: https://img.shields.io/badge/license-MIT%20%20License%20-E91E63.svg?style=flat-square
[License]: https://github.com/kataras/iris/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/release-v4.0.0--alpha.4-blue.svg?style=flat-square
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
