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


## Getting started

1. Install `$ go get -u github.com/kataras/iris`

2. Read the [Iris book](https://www.gitbook.com/book/kataras/iris/details)

3. Examples are [here](https://github.com/iris-contrib/examples)

4. Post an [issue](https://github.com/kataras/iris/issues) or [idea](https://github.com/kataras/iris/issues)

5. Chat with the [Community](https://gitter.im/kataras/iris)

## Versioning
Iris is in active development status, check for updates once per week. **Compatible only with go1.6+ **.

Current: **v2.0.0**


Read more about Semantic Versioning 2.0.0

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions


## Benchmarks


Benchmarks results taken [from external source](https://github.com/smallnest/go-web-framework-benchmark), created by [@smallnest](https://github.com/smallnest).

This is the most realistic benchmark suite than you will find for Go Web Frameworks. Give attention to its readme.md.

April 22 2016


![Benchmark Wizzard Concurenncy](http://kataras.github.io/iris/assets/benchmark_all_28April_2016.png)

[click here to view detailed tables of different benchmarks](https://github.com/smallnest/go-web-framework-benchmark)


### Q: What makes iris significantly [faster](#benchmarks)?
*    First of all Iris is builded on top of the [fasthttp](https://github.com/valyala/fasthttp)
*    Make use of the algorithm which the BSD's kernel uses for internal routing
*    Iris can detect what features are used and what don't and optimized itself before server run

## Contributors

Thanks goes to the people who have contributed code to this package, see the

- [Iris GitHub Contributors page](https://github.com/kataras/iris/graphs/contributors).
- [Iris Contrib GitHub Contributors page](https://github.com/orgs/iris-contrib/people).


## Community

If you'd like to discuss this package, or ask questions about it, feel free to

* **Chat**: https://gitter.im/kataras/iris
* **Post/Discuss an issue**: https://github.com/kataras/iris/issues

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
