# Iris Web Framework
<img align="right" width="132" src="http://kataras.github.io/iris/assets/56e4b048f1ee49764ddd78fe_iris_favicon.ico">
[![Build Status](https://travis-ci.org/kataras/iris.svg?branch=development&style=flat-square)](https://travis-ci.org/kataras/iris)
[![Go Report Card](https://goreportcard.com/badge/github.com/kataras/iris?style=flat-square)](https://goreportcard.com/report/github.com/kataras/iris)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/kataras/iris?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)
[![GoDoc](https://godoc.org/github.com/kataras/iris?status.svg)](https://godoc.org/github.com/kataras/iris)
[![License](https://img.shields.io/badge/license-BSD3-blue.svg?style=flat-square)](LICENSE)

**Iris** is a modern, **community-driven** **web** application **framework** for **Go**. Comes with the [highest performance](#benchmarks) which ever achieved.

![Benchmark Wizzard Concurenncy](http://kataras.github.io/iris/assets/benchmark_all_28April_2016.png)


**Easy to learn** while providing robust set of features for building **modern & shiny web applications**.

<a href="https://www.gitbook.com/read/book/kataras/iris"><img align="left" width="185" src="https://raw.githubusercontent.com/kataras/iris/gh-pages/assets/book/cover_1.png"></a>

![Hi Iris GIF](http://kataras.github.io/iris/assets/hi_iris_may.gif)

## Getting started

1. Install `$ go get -u github.com/kataras/iris`

2. Read the [Iris book](https://www.gitbook.com/book/kataras/iris/details)

3. Examples are [here](https://github.com/iris-contrib/examples)

4. Post an [issue](https://github.com/kataras/iris/issues) or [idea](https://github.com/kataras/iris/issues)

5. Chat with the [Community](https://gitter.im/kataras/iris)

## Community

**Open debates**

 - [E-book Cover - Which one you suggest?](https://github.com/kataras/iris/issues/67)
 - [Be able to append a Data object on the Context](https://github.com/kataras/iris/issues/72)
 - [Form Validation](https://github.com/kataras/iris/issues/84)

**If you'd like to discuss this package, or ask questions about it, feel free to**

* Post: https://github.com/kataras/iris/issues
* Chat: https://gitter.im/kataras/iris




## Benchmarks


Benchmarks results taken [from external source](https://github.com/smallnest/go-web-framework-benchmark), created by [@smallnest](https://github.com/smallnest).

This is the most realistic benchmark suite than you will find for Go Web Frameworks. Give attention to its readme.md.

April 22 2016


![Benchmark Wizzard Concurenncy](http://kataras.github.io/iris/assets/benchmark_all_28April_2016.png)

[click here to view detailed tables of different benchmarks](https://github.com/smallnest/go-web-framework-benchmark)


### Q: What makes iris significantly faster?
*    Follows the path of the extensible [fasthttp](https://github.com/valyala/fasthttp)
*    Makes use of a part of the algorithm which the [FreeBSD](https://github.com/freebsd/freebsd) uses
*    Detects what features are used and what don't and optimize itself
*    ...
*    ...

## Versioning
Iris is in active development status, check for updates once per week. Compatible only with go1.6+

Current: **v2.2.0**


Read more about Semantic Versioning 2.0.0

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions
 -
## Contributors

Thanks goes to the people who have contributed code to this package, see the

- [Iris GitHub Contributors page](https://github.com/kataras/iris/graphs/contributors).
- [Iris Contrib GitHub Contributors page](https://github.com/orgs/iris-contrib/people).



## Donations

I am a student at the [University of Central Macedonia](http://teiser.gr/), therefore I have no income value.

From a younger age I have dedicated all my time to various open source projects, including, my lovely child, Iris.

**If and only if** you think that any information you obtained here is worth something  and you are willing to pay for it(and you can afford it),  feel free to send any amount through paypal

[![](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://www.paypal.com/cgi-bin/webscr?cmd=_donations&business=makis%40ideopod%2ecom&lc=GR&item_name=Iris%20web%20framework&item_number=iriswebframeworkdonationid2016&amount=2%2e00&currency_code=EUR&bn=PP%2dDonationsBF%3abtn_donateCC_LG%2egif%3aNonHosted)

## License

This project is licensed under the [BSD 3-Clause License](https://opensource.org/licenses/BSD-3-Clause).
License can be found [here](https://github.com/kataras/iris/blob/master/LICENSE).

