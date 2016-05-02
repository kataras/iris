# Iris Web Framework
<img align="right" width="132" src="http://kataras.github.io/iris/assets/56e4b048f1ee49764ddd78fe_iris_favicon.ico">
[![Build Status](https://travis-ci.org/kataras/iris.svg?branch=development&style=flat-square)](https://travis-ci.org/kataras/iris)
[![Go Report Card](https://goreportcard.com/badge/github.com/kataras/iris?style=flat-square)](https://goreportcard.com/report/github.com/kataras/iris)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/kataras/iris?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![GoDoc](https://godoc.org/github.com/kataras/iris?status.svg)](https://godoc.org/github.com/kataras/iris)
[![License](https://img.shields.io/badge/license-BSD3-blue.svg?style=flat-square)](LICENSE)

A Community driven Web framework written in Go. Its performance is unique, seems to be the [fastest](#benchmarks) golang web framework was ever created.

Start using Iris Web Framework today. Iris is easy-to-learn while providing robust set of features for building modern & shiny web applications.

![Hi Iris GIF](http://kataras.github.io/iris/assets/hi_iris_may.gif)




> [Build a better web, together.](https://www.gitbook.com/read/book/kataras/iris)

[![https://www.gitbook.com/read/book/kataras/iris](https://raw.githubusercontent.com/kataras/iris/gh-pages/assets/book/cover_1.png)](https://www.gitbook.com/read/book/kataras/iris)

### Q: What makes iris significantly [faster](#benchmarks)?
*    First of all Iris is builded on top of the [fasthttp](https://github.com/valyala/fasthttp)
*    Iris uses the same algorithm as the BSD's kernel does for routing (call it Trie)
*    Iris can detect what features are used and what don't and optimized itself before server run.
*    Middlewares and Plugins are 'light' , that's a principle.



## Versioning

Current: **v2.0.0**

##### [Changelog v1.2.1 -> v2.0.0](https://github.com/kataras/iris/blob/development/HISTORY.md)


Read more about Semantic Versioning 2.0.0

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions

## Install
Iris is in active development status, check for updates once per week. **Compatible with go1.6+ **.
```sh
$ go get -u github.com/kataras/iris
```


- [Read the Iris book](https://www.gitbook.com/read/book/kataras/iris)
- [Examples](https://github.com/iris-contrib/examples)


## Benchmarks


Benchmarks results taken [from external source](https://github.com/smallnest/go-web-framework-benchmark), created by [@smallnest](https://github.com/smallnest).

This is the most realistic benchmark suite than you will find for Go Web Frameworks. Give attention to its readme.md.

April 22 2016


![Benchmark Wizzard Basic 22 April 2016](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/master/concurrency-pipeline.png)

[click here to view detailed tables from all kind of different benchmark results](https://github.com/smallnest/go-web-framework-benchmark)

## Third Party Middlewares


>Note: After v1.1 most of the third party middlewares are incompatible, I'm putting a big effort to convert all of these to work with Iris,
if you want to help please do so (pr).


| Middleware | Author | Description | Tested |
| -----------|--------|-------------|--------|
| [Graceful](https://github.com/iris-contrib/graceful) | [Ported to iris](https://github.com/iris-contrib/graceful) | Graceful HTTP Shutdown | [Yes](https://github.com/iris-contrib/examples/tree/master/graceful) |
| [gzip](https://github.com/kataras/iris/tree/development/middleware/gzip/) | [Iris](https://github.com/kataras/iris) | GZIP response compression | [Yes](https://github.com/kataras/iris/tree/development/middleware/gzip/README.md) |
| [RestGate](https://github.com/pjebs/restgate) | [Prasanga Siripala](https://github.com/pjebs) | Secure authentication for REST API endpoints | No |
| [secure](https://github.com/kataras/iris/tree/development/middleware/secure) | [Ported to Iris](https://github.com/kataras/iris/tree/development/middleware/secure) | Middleware that implements a few quick security wins | [Yes](https://github.com/iris-contrib/examples/tree/master/secure) |
| [JWT Middleware](https://github.com/auth0/go-jwt-middleware) | [Auth0](https://github.com/auth0) | Middleware checks for a JWT on the `Authorization` header on incoming requests and decodes it| No |
| [binding](https://github.com/mholt/binding) | [Matt Holt](https://github.com/mholt) | Data binding from HTTP requests into structs | No |
| [i18n](https://github.com/kataras/iris/tree/development/middleware/i18n) | [Iris](https://github.com/kataras/iris) | Internationalization and Localization | [Yes](https://github.com/iris-contrib/examples/tree/master/middleware_internationalization_i18n) |
| [logrus](https://github.com/meatballhat/negroni-logrus) | [Dan Buch](https://github.com/meatballhat) | Logrus-based logger | No |
| [render](https://github.com/iris-contrib/render) | [Ported to iris](https://github.com/kataras/iris) | Render JSON, XML and HTML templates | [Yes](https://github.com/iris-contrib/render) |
| [gorelic](https://github.com/jingweno/negroni-gorelic) | [Jingwen Owen Ou](https://github.com/jingweno) | New Relic agent for Go runtime | No |
| [pongo2](https://github.com/iris-contrib/examples/tree/master/middleware_pongo2) | [Iris](https://github.com/kataras/iris) | Middleware for [pongo2 templates](https://github.com/flosch/pongo2)| [Yes](https://github.com/iris-contrib/examples/tree/master/middleware_pongo2) |
| [oauth2](https://github.com/goincremental/negroni-oauth2) | [David Bochenski](https://github.com/bochenski) | oAuth2 middleware | No |
| [permissions2](https://github.com/xyproto/permissions2) | [Alexander Rødseth](https://github.com/xyproto) | Cookies, users and permissions | No |
| [onthefly](https://github.com/xyproto/onthefly) | [Alexander Rødseth](https://github.com/xyproto) | Generate TinySVG, HTML and CSS on the fly | No |
| [cors](https://github.com/kataras/iris/tree/development/middleware/cors) | [Keuller Magalhaes](https://github.com/keuller) | [Cross Origin Resource Sharing](http://www.w3.org/TR/cors/) (CORS) support | [Yes](https://github.com/kataras/iris/tree/development/middleware/cors) |
| [xrequestid](https://github.com/pilu/xrequestid) | [Andrea Franz](https://github.com/pilu) | Middleware that assigns a random X-Request-Id header to each request | No |
| [VanGoH](https://github.com/auroratechnologies/vangoh) | [Taylor Wrobel](https://github.com/twrobel3) | Configurable [AWS-Style](http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html) HMAC authentication middleware | No |
| [stats](https://github.com/thoas/stats) | [Florent Messa](https://github.com/thoas) | Store information about your web application (response time, etc.) | No |

## Contributors

Thanks goes to the people who have contributed code to this package, see the
[GitHub Contributors page][].

[GitHub Contributors page]: https://github.com/kataras/iris/graphs/contributors



## Community

If you'd like to discuss this package, or ask questions about it, feel free to

* **Chat**: https://gitter.im/kataras/iris

## Todo
- [x] [Provide a lighter, with less using bytes,  to save middleware for a route.](https://github.com/kataras/iris/tree/development/handler.go)
- [x] [Create examples.](https://github.com/iris-contrib/examples)
- [x] [Subdomains supports with the same syntax as iris.Get, iris.Post ...](https://github.com/iris-contrib/examples/tree/master/subdomains_simple)
- [x] [Provide a more detailed benchmark table](https://github.com/smallnest/go-web-framework-benchmark)
- [x] Convert useful middlewares out there into Iris middlewares, or contact with their authors to do so.
- [ ] Provide automatic HTTPS using https://letsencrypt.org/how-it-works/.
- [ ] Create administration web interface as plugin.
- [x] Create an easy websocket api.
- [x] [Create a mechanism that scan for Typescript files, compile them on server startup and serve them.](https://github.com/kataras/iris/tree/development/plugin/typescript)
- [x] Simplify the plugin mechanism.
- [ ] Implement an Iris updater and add the specific entry -bool on IrisConfig.
- [x] [Re-Implement the sessions from zero.](https://github.com/kataras/iris/tree/development/sessions)

## Articles

* [Ultra-wide framework Go Http routing performance comparison](https://translate.google.com/translate?sl=auto&tl=en&js=y&prev=_t&hl=el&ie=UTF-8&u=http%3A%2F%2Fcolobu.com%2F2016%2F03%2F23%2FGo-HTTP-request-router-and-web-framework-benchmark%2F&edit-text=&act=url)

> According to my  article ( comparative ultra wide frame Go Http routing performance ) on a variety of relatively Go http routing framework, Iris clear winner, its performance far exceeds other Golang http routing framework.


## License

This project is licensed under the [BSD 3-Clause License](https://opensource.org/licenses/BSD-3-Clause).
License can be found [here](https://github.com/kataras/iris/blob/master/LICENSE).
