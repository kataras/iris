Builtin Handlers
------------

| Middleware | Example |
| -----------|-------------|
| [rewrite](rewrite) | [iris/_examples/routing/rewrite](https://github.com/kataras/iris/tree/main/_examples/routing/rewrite) |
| [basic authentication](basicauth) | [iris/_examples/auth/basicauth](https://github.com/kataras/iris/tree/main/_examples/auth/basicauth) |
| [request logger](logger) | [iris/_examples/logging/request-logger](https://github.com/kataras/iris/tree/main/_examples/logging/request-logger) |
| [HTTP method override](methodoverride) | [iris/middleware/methodoverride/methodoverride_test.go](https://github.com/kataras/iris/blob/main/middleware/methodoverride/methodoverride_test.go) |
| [profiling (pprof)](pprof) | [iris/_examples/pprof](https://github.com/kataras/iris/tree/main/_examples/pprof) |
| [Google reCAPTCHA](recaptcha) | [iris/_examples/auth/recaptcha](https://github.com/kataras/iris/tree/main/_examples/auth/recaptcha) |
| [hCaptcha](hcaptcha) | [iris/_examples/auth/recaptcha](https://github.com/kataras/iris/tree/main/_examples/auth/hcaptcha) |
| [recovery](recover) | [iris/_examples/recover](https://github.com/kataras/iris/tree/main/_examples/recover) |
| [rate](rate) | [iris/_examples/request-ratelimit](https://github.com/kataras/iris/tree/main/_examples/request-ratelimit) |
| [jwt](jwt) | [iris/_examples/auth/jwt](https://github.com/kataras/iris/tree/main/_examples/auth/jwt) |
| [requestid](requestid) | [iris/middleware/requestid/requestid_test.go](https://github.com/kataras/iris/blob/main/_examples/middleware/requestid/requestid_test.go) |

Community made
------------

Most of the experimental handlers are ported to work with _iris_'s handler form, from third-party sources.

| Middleware | Description | Example |
| -----------|--------|-------------|
| [pg](https://github.com/iris-contrib/middleware/tree/master/pg) | Middleware that provides easy and type-safe access to PostgreSQL database | [iris-contrib/middleware/pg/_examples](https://github.com/iris-contrib/middleware/tree/master/pg/_examples) |
| [jwt](https://github.com/iris-contrib/middleware/tree/master/jwt) | Middleware checks for a JWT on the `Authorization` header on incoming requests and decodes it | [iris-contrib/middleware/jwt/_example](https://github.com/iris-contrib/middleware/tree/master/jwt/_example) |
| [cors](https://github.com/iris-contrib/middleware/tree/master/cors) | HTTP Access Control | [iris-contrib/middleware/cors/_example](https://github.com/iris-contrib/middleware/tree/master/cors/_example) |
| [secure](https://github.com/iris-contrib/middleware/tree/master/secure) | Middleware that implements a few quick security wins | [iris-contrib/middleware/secure/_example](https://github.com/iris-contrib/middleware/tree/master/secure/_example/main.go) |
| [tollbooth](https://github.com/iris-contrib/middleware/tree/master/tollboothic) | Generic middleware to rate-limit HTTP requests | [iris-contrib/middleware/tollboothic/_examples/limit-handler](https://github.com/iris-contrib/middleware/tree/master/tollboothic/_examples/limit-handler) |
| [cloudwatch](https://github.com/iris-contrib/middleware/tree/master/cloudwatch) |  AWS cloudwatch metrics middleware |[iris-contrib/middleware/cloudwatch/_example](https://github.com/iris-contrib/middleware/tree/master/cloudwatch/_example) |
| [new relic](https://github.com/iris-contrib/middleware/tree/master/newrelic) | Official [New Relic Go Agent](https://github.com/newrelic/go-agent) | [iris-contrib/middleware/newrelic/_example](https://github.com/iris-contrib/middleware/tree/master/newrelic/_example) |
| [prometheus](https://github.com/iris-contrib/middleware/tree/master/prometheus)| Easily create metrics endpoint for the [prometheus](http://prometheus.io) instrumentation tool | [iris-contrib/middleware/prometheus/_example](https://github.com/iris-contrib/middleware/tree/master/prometheus/_example) |
| [casbin](https://github.com/iris-contrib/middleware/tree/master/casbin)| An authorization library that supports access control models like ACL, RBAC, ABAC | [iris-contrib/middleware/casbin/_examples](https://github.com/iris-contrib/middleware/tree/master/casbin/_examples) |
| [sentry-go (ex. raven)](https://github.com/getsentry/sentry-go/tree/master/iris)| Sentry client in Go | [sentry-go/example/iris](https://github.com/getsentry/sentry-go/blob/master/example/iris/main.go) | <!-- raven was deprecated by its company, the successor is sentry-go, they contain an Iris middleware. -->
| [csrf](https://github.com/iris-contrib/middleware/tree/master/csrf)| Cross-Site Request Forgery Protection | [iris-contrib/middleware/csrf/_example](https://github.com/iris-contrib/middleware/blob/master/csrf/_example/main.go) |
| [throttler](https://github.com/iris-contrib/middleware/tree/master/throttler)| Rate limiting access to HTTP endpoints | [iris-contrib/middleware/throttler/_example](https://github.com/iris-contrib/middleware/blob/master/throttler/_example/main.go) |

Third-Party Handlers
------------

Iris has its own middleware form of `func(ctx iris.Context)` but it's also compatible with all `net/http` middleware forms. See [here](https://github.com/kataras/iris/tree/main/_examples/convert-handlers).

Here's a small list of useful third-party handlers:

| Middleware | Description |
| -----------|-------------|
| [goth](https://github.com/markbates/goth) | OAuth, OAuth2 authentication. [Example](https://github.com/kataras/iris/tree/main/_examples/auth/goth) |
| [permissions2](https://github.com/xyproto/permissions2) | Cookies, users and permissions. [Example](https://github.com/kataras/iris/tree/main/_examples/auth/permissions) |
| [csp](https://github.com/awakenetworks/csp) | [Content Security Policy](https://www.w3.org/TR/CSP2/) (CSP) support |
| [delay](https://github.com/jeffbmartinez/delay) | Add delays/latency to endpoints. Useful when testing effects of high latency |
| [onthefly](https://github.com/xyproto/onthefly) | Generate TinySVG, HTML and CSS on the fly |
| [RestGate](https://github.com/pjebs/restgate) | Secure authentication for REST API endpoints |
| [stats](https://github.com/thoas/stats) | Store information about your web application (response time, etc.) |
| [VanGoH](https://github.com/auroratechnologies/vangoh) | Configurable [AWS-Style](http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html) HMAC authentication middleware |
| [digits](https://github.com/bamarni/digits) | Middleware that handles [Twitter Digits](https://get.digits.com/) authentication |

> Feel free to put up your own middleware in this list!