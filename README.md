# iris: sometimes the simplicity is the best solution
[![Travis Widget]][Travis]
[![Report Widget]][Report]
[![Awesome Widget]][Awesome]
[![License Widget]][License]
[![Release Widget]][Release]
[![Examples Widget]][Examples]
[![Documentation Widget]][Documentation]
[![Chat Widget]][Chat]  
[![Iris Logo](https://github.com/kataras/tolk/raw/master/logo.jpg)](http://iris-go.com)

The fastest web framework for Go. Easy to learn, while it's highly customizable.
Ideally suited for both experienced and novice Developers.



## Features

- Focus on high performance
- Automatically install TLS certificates from https://letsencrypt.org
- Proxy HTTP and WebSocket requests
- Robust routing and middleware ecosystem
- Define virtual hosts and (wildcard) subdomains with path level routing
- Graceful shutdown
- Limit request body
- I18N
- Serve static files
- Log requests
- Gzip response
- Authentication
 - OAuth, OAuth2 supporting 27+ popular websites
 - JWT
 - Basic Authentication
 - HTTP Sessions
- Add / Remove trailing slash from the URL with option to redirect
- Redirect requests
 - HTTP to HTTPS
 - HTTP to HTTPS WWW
 - HTTP to HTTPS non WWW
 - Non WWW to WWW
 - WWW to non WWW
- View system supporting more than six template engines
- Highly scalable rich render (Markdown, JSON, JSONP, XML...)
- Websocket API similar to socket.io  
- Hot Reload
- Typescript integration + Web IDE
- Checks for updates at startup

Getting Started
------------

### Installation

The only requirement is the [Go Programming Language](https://golang.org/dl), at least v1.7.

```bash
$ go get -u github.com/kataras/iris/iris
```

> If you have installation issues or you are connected to the Internet through China please, [click here](https://kataras.gitbooks.io/iris/content/install.html).

### Need help?

 <a href="https://www.gitbook.com/book/kataras/iris/details"><img align="right" width="125" src="https://raw.githubusercontent.com/iris-contrib/website/gh-pages/assets/book/cover_4.jpg"></a>


 - The most important is to read [the practical guide](https://www.gitbook.com/book/kataras/iris/details).

 - Explore & download the [examples](https://github.com/iris-contrib/examples).

 - [HISTORY.md](https://github.com//kataras/iris/tree/master/HISTORY.md) file is your best friend.

#### FAQ

Explore [these questions](https://github.com/kataras/iris/issues?q=label%3Aquestion) or navigate to the [community chat][Chat].


Support
------------

Hi, my name is Gerasimos Maropoulos and I'm the author of this project, let me put a few words about me.

I started to design iris the night of the 13 March 2016, some weeks later, iris started to became famous and I have to fix many issues and implement new features, but I didn't have time to work on Iris because I had a part time job and the (software engineering) colleague which I studied.

I wanted to make iris' users proud of the framework they're using, so I decided to interupt my studies and colleague, two days later I left from my part time job also.

Today I spend all my days and nights coding for Iris, and I'm happy about this, therefore I have zero incoming value.

- :star: the project
- [Donate](https://github.com/kataras/iris/blob/master/DONATIONS.md)
- :earth_americas: spread the word
- [Contribute](#contributing) to the project



Philosophy
------------

The Iris philosophy is to provide robust tooling for HTTP, making it a great solution for single page applications, web sites, hybrids, or public HTTP APIs. Keep note that, today, iris has the clostest performance to the nginx.

Iris does not force you to use any specific ORM or template engine. With support for the most used template engines, you can quickly craft the perfect application.



Benchmarks
------------

This Benchmark test aims to compare the whole HTTP request processing between Go web frameworks.


![Benchmark Wizzard July 21, 2016- Processing Time Horizontal Graph](https://raw.githubusercontent.com/smallnest/go-web-framework-benchmark/4db507a22c964c9bc9774c5b31afdc199a0fe8b7/benchmark.png)

**The results have been updated on July 21, 2016**

Testing
------------

I recommend writing your API tests using this new library, [httpexpect](https://github.com/gavv/httpexpect) which supports Iris and fasthttp now, after my request [here](https://github.com/gavv/httpexpect/issues/2). You can find Iris examples [here](https://github.com/gavv/httpexpect/blob/master/example/iris_test.go), [here](https://github.com/kataras/iris/blob/master/http_test.go) and [here](https://github.com/kataras/iris/blob/master/context_test.go).

Versioning
------------

Current: **v4.4.2**

>  Iris is an active project

Read more about Semantic Versioning 2.0.0

 - http://semver.org/
 - https://en.wikipedia.org/wiki/Software_versioning
 - https://wiki.debian.org/UpstreamGuide#Releases_and_Versions


Contributing
------------
If you are interested in contributing to the Iris project, please make sure that you read the document [CONTRIBUTING](https://github.com/kataras/iris/blob/master/CONTRIBUTING.md) first.

- Report issues
- Suggest new features or enhancements

People
------------
The big thanks goes to [all people](https://github.com/kataras/iris/issues?utf8=%E2%9C%93&q=label%3A%22feature+request%22) who help building this framework with feature-requests & bug reports!

The author of Iris is [@kataras](https://github.com/kataras). If **you**'re willing to donate, feel **free** to navigate to the [DONATIONS PAGE](https://github.com/kataras/iris/blob/master/DONATIONS.md).





License
------------

This project is licensed under the [MIT License](LICENSE), Copyright (c) 2016 Gerasimos Maropoulos.


[Travis Widget]: https://img.shields.io/travis/kataras/iris.svg?style=flat-square
[Travis]: http://travis-ci.org/kataras/iris
[License Widget]: https://img.shields.io/badge/license-MIT%20%20License%20-E91E63.svg?style=flat-square
[License]: https://github.com/kataras/iris/blob/master/LICENSE
[Release Widget]: https://img.shields.io/badge/release-4.4.2%20-blue.svg?style=flat-square
[Release]: https://github.com/kataras/iris/releases
[Chat Widget]: https://img.shields.io/badge/community-chat%20-00BCD4.svg?style=flat-square
[Chat]: https://kataras.rocket.chat/channel/iris
[ChatMain]: https://kataras.rocket.chat/channel/iris
[ChatAlternative]: https://gitter.im/kataras/iris
[Report Widget]: https://img.shields.io/badge/report%20card%20-A%2B-F44336.svg?style=flat-square
[Report]: http://goreportcard.com/report/kataras/iris
[Documentation Widget]: https://img.shields.io/badge/docs-reference%20-5272B4.svg?style=flat-square
[Documentation]: https://www.gitbook.com/book/kataras/iris/details
[Examples Widget]: https://img.shields.io/badge/examples-repository%20-3362c2.svg?style=flat-square
[Examples]: https://github.com/iris-contrib/examples
[Language Widget]: https://img.shields.io/badge/powered_by-Go-3362c2.svg?style=flat-square
[Language]: http://golang.org
[Platform Widget]: https://img.shields.io/badge/platform-Any--OS-gray.svg?style=flat-square
[Awesome]: https://github.com/avelino/awesome-go
[Awesome Widget]: https://img.shields.io/badge/awesome%20go-%E2%9C%93-ff69b4.svg?style=flat-square
