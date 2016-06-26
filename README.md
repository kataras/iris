[![Iris Logo](http://iris-go.com/assets/iris_full_logo_2.png)](http://iris-go.com)

[![Travis Widget]][Travis] [![Release Widget]][Release] [![Report Widget]][Report] [![License Widget]][License] [![Gitter Widget]][Gitter] [![Documentation Widget]][Documentation]

[Travis Widget]: https://img.shields.io/travis/tmrts/boilr.svg?style=flat-square
[Travis]: http://travis-ci.org/kataras/iris
[License Widget]: https://img.shields.io/badge/license-MIT%20%20License%20-E91E63.svg?style=flat-square
[License]: https://github.com/kataras/iris/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/release-v3.0.0--rc.3-blue.svg?style=flat-square
[Release]: https://github.com/kataras/iris/releases
[Gitter Widget]: https://img.shields.io/badge/chat-on%20gitter-00BCD4.svg?style=flat-square
[Gitter]: https://gitter.im/kataras/iris
[Report Widget]: https://img.shields.io/badge/report%20card-A%2B-F44336.svg?style=flat-square
[Report]: http://goreportcard.com/report/kataras/iris
[Documentation Widget]: https://img.shields.io/badge/documentation-reference-5272B4.svg?style=flat-square
[Documentation]: https://www.gitbook.com/book/kataras/iris/details
[Language Widget]: https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square
[Language]: http://golang.org
[Platform Widget]: https://img.shields.io/badge/platform-Any--OS-gray.svg?style=flat-square

The fastest web framework for Go. Easy to [learn](https://www.gitbook.com/book/kataras/iris/details) while providing robust set of features for building modern web applications.

[![Benchmark Wizzard Processing Time Horizontal Graph](https://raw.githubusercontent.com/iris-contrib/website/cf71811e6acb2f9bf1e715e25660392bf090b923/assets/benchmark_horizontal_transparent.png)](https://github.com/smallnest/go-web-framework-benchmark)


```sh
$ cat test_json.go
```
```go
package main

import (
	"github.com/kataras/iris"
)

func main() {

	// render JSON
	iris.Get("/hi_json", func(c *iris.Context) {
		c.JSON(iris.StatusOK, iris.Map{
			"Name":  "Iris",
			"Born":  "13 March 2016",
			"Stars": 3533,
		})
	})
	iris.Listen(":8080")
}
```

> Learn more about [render](https://kataras.gitbooks.io/iris/content/render.html).

Installation
------------
 The only requirement is Go 1.6

`$ go get -u github.com/kataras/iris/iris`

 >If you are connected to the Internet through China [click here](https://kataras.gitbooks.io/iris/content/install.html)

FAQ
------------
You can find answers by exploring [these questions](https://github.com/kataras/iris/issues?q=label%3Aquestion).


Features
------------
- Focus on high performance
- Robust routing supports static and wildcard subdomains
- View system supporting [5+](https://kataras.gitbooks.io/iris/content/render_templates.html) template engines
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

<a href="https://www.gitbook.com/book/kataras/iris/details"><img align="right" width="185" src="http://iris-go.com/assets/book/cover_1.png"></a>


- Read the [book](https://www.gitbook.com/book/kataras/iris/details) or [wiki](https://github.com/kataras/iris/wiki)

- Take a look at the [examples](https://github.com/iris-contrib/examples)

- [HISTORY](https://github.com//kataras/iris/tree/master/HISTORY.md) file is your best friend.


If you'd like to discuss this package, or ask questions about it, feel free to

* Post an issue or  idea [here](https://github.com/kataras/iris/issues)
* [Chat]( https://gitter.im/kataras/iris) with us

Open debates

 - [Contribute: New website and logo for Iris](https://github.com/kataras/iris/issues/153)
 - [E-book Cover - Which one you suggest?](https://github.com/kataras/iris/issues/67)

Philosophy
------------

The Iris philosophy is to provide robust tooling for HTTP, making it a great solution for single page applications, web sites, hybrids, or public HTTP APIs.

Iris does not force you to use any specific ORM or template engine. With support for the most used template engines, you can quickly craft the perfect application.

Benchmarks
------------

[This Benchmark suite](https://github.com/smallnest/go-web-framework-benchmark) aims to compare the whole HTTP request processing between Go web frameworks.

![Benchmark Wizzard Processing Time Horizontal Graph](https://raw.githubusercontent.com/iris-contrib/website/cf71811e6acb2f9bf1e715e25660392bf090b923/assets/benchmark_horizontal_transparent.png)

[Please click here to view all detailed benchmarks.](https://github.com/smallnest/go-web-framework-benchmark)

Testing
------------

Tests are located to the [iris-contrib/tests repository](https://github.com/iris-contrib/tests), community should write some code there!
I recommend writing your API tests using this new library, [httpexpect](https://github.com/gavv/httpexpect) which supports Iris and fasthttp now, after my request [here](https://github.com/gavv/httpexpect/issues/2).

Versioning
------------

Current: **v3.0.0-rc.3**
>  Iris is an active project


Read more about Semantic Versioning 2.0.0

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions


Todo
------------
> for the next release 'v3'

- [x] [Dynamic/Wildcard subdomains](https://kataras.gitbooks.io/iris/content/subdomains.html).
- [x] Create server & client side (js) library for .on('event', func action(...)) / .emit('event')... (like socket.io but supports only websocket).
- [x] Create a view system supporting different types of template engines.
- [x] Extend, test and publish to the public the [Iris' cmd](https://github.com/kataras/iris/tree/master/iris).

If you're willing to donate click [here](DONATIONS.md)!

People
------------
A big thanks goes to [all people](https://github.com/kataras/iris/issues?utf8=%E2%9C%93&q=label%3A%22feature+request%22) who help building this framework with feature-requests & bug reports!

The author of Iris is [@kataras](https://github.com/kataras).


Contributing
------------
If you are interested in contributing to the Iris project, please see the document [CONTRIBUTING](https://github.com/kataras/iris/blob/master/CONTRIBUTING.md).


Third-Party Licenses
------------
This project is using third-party libraries to achieve this result.

Third-Party Licenses can be found [here](THIRDPARTY-LICENSE.md)


License
------------

This project is licensed under the MIT License.

License can be found [here](LICENSE).
