Built'n Handlers
------------

| Middleware | Example |
| -----------|-------------|
| [basic authentication](basicauth) | [iris/_examples/authentication/basicauth](https://github.com/kataras/iris/tree/master/_examples/authentication/basicauth) |
| [Google reCAPTCHA](recaptcha) | [iris/_examples/miscellaneous/recaptcha](https://github.com/kataras/iris/tree/master/_examples/miscellaneous/recaptcha) |
| [localization and internationalization](i18n) | [iris/_examples/miscellaneous/i81n](https://github.com/kataras/iris/tree/master/_examples/miscellaneous/i18n) |
| [request logger](logger) | [iris/_examples/http_request/request-logger](https://github.com/kataras/iris/tree/master/_examples/http_request/request-logger) |
| [profiling (pprof)](pprof) | [iris/_examples/miscellaneous/pprof](https://github.com/kataras/iris/tree/master/_examples/miscellaneous/pprof) |
| [recovery](recover) | [iris/_examples/miscellaneous/recover](https://github.com/kataras/iris/tree/master/_examples/miscellaneous/recover) |

Experimental Handlers
------------

Most of the experimental handlers are ported to work with _iris_'s handler form, from third-party sources.

| Middleware | Description | Example |
| -----------|--------|-------------|
| [jwt](https://github.com/iris-contrib/middleware/tree/master/jwt) | Middleware checks for a JWT on the `Authorization` header on incoming requests and decodes it. | [iris-contrib/middleware/jwt/_example](https://github.com/iris-contrib/middleware/tree/master/jwt/_example) |
| [cors](https://github.com/iris-contrib/middleware/tree/master/cors) | HTTP Access Control. | [iris-contrib/middleware/cors/_example](https://github.com/iris-contrib/middleware/tree/master/cors/_example) |
| [secure](https://github.com/iris-contrib/middleware/tree/master/secure) | Middleware that implements a few quick security wins. | [iris-contrib/middleware/secure/_example](https://github.com/iris-contrib/middleware/tree/master/secure/_example/main.go) |
| [tollbooth](https://github.com/iris-contrib/middleware/tree/master/tollboothic) | Generic middleware to rate-limit HTTP requests. | [iris-contrib/middleware/tollbooth/_examples/limit-handler](https://github.com/iris-contrib/middleware/tree/master/tollbooth/_examples/limit-handler) |
| [cloudwatch](https://github.com/iris-contrib/middleware/tree/master/cloudwatch) |  AWS cloudwatch metrics middleware. |[iris-contrib/middleware/cloudwatch/_example](https://github.com/iris-contrib/middleware/tree/master/cloudwatch/_example) |
| [new relic](https://github.com/iris-contrib/middleware/tree/master/newrelic) | Official [New Relic Go Agent](https://github.com/newrelic/go-agent). | [iris-contrib/middleware/newrelic/_example](https://github.com/iris-contrib/middleware/tree/master/newrelic/_example) |
| [prometheus](https://github.com/iris-contrib/middleware/tree/master/prometheus)| Easily create metrics endpoint for the [prometheus](http://prometheus.io) instrumentation tool | [iris-contrib/middleware/prometheus/_example](https://github.com/iris-contrib/middleware/tree/master/prometheus/_example) |
| [casbin](https://github.com/iris-contrib/middleware/tree/master/casbin)| An authorization library that supports access control models like ACL, RBAC, ABAC | [iris-contrib/middleware/casbin/_examples](https://github.com/iris-contrib/middleware/tree/master/casbin/_examples) |
| [raven](https://github.com/iris-contrib/middleware/tree/master/raven)| Sentry client in Go | [raven/_example](https://github.com/iris-contrib/middleware/blob/master/raven/_example/main.go) |
| [csrf](https://github.com/iris-contrib/middleware/tree/master/csrf)| Cross-Site Request Forgery Protection | [csrf/_example](https://github.com/iris-contrib/middleware/blob/master/csrf/_example/main.go) (hard-tested for Iris) **NEW** |
Third-Party Handlers
------------

iris has its own middleware form of `func(ctx context.Context)` but it's also compatible with all `net/http` middleware forms. See [here](https://github.com/kataras/iris/tree/master/_examples/convert-handlers).

Here's a small list of useful third-party handlers:

| Middleware | Description |
| -----------|-------------|
| [goth](https://github.com/markbates/goth) | OAuth, OAuth2 authentication. [Example](https://github.com/kataras/iris/tree/master/_examples/authentication/oauth2) |
| [binding](https://github.com/mholt/binding) | Data binding from HTTP requests into structs |
| [csp](https://github.com/awakenetworks/csp) | [Content Security Policy](https://www.w3.org/TR/CSP2/) (CSP) support |
| [delay](https://github.com/jeffbmartinez/delay) | Add delays/latency to endpoints. Useful when testing effects of high latency |
| [onthefly](https://github.com/xyproto/onthefly) | Generate TinySVG, HTML and CSS on the fly |
| [permissions2](https://github.com/xyproto/permissions2) | Cookies, users and permissions |
| [RestGate](https://github.com/pjebs/restgate) | Secure authentication for REST API endpoints |
| [stats](https://github.com/thoas/stats) | Store information about your web application (response time, etc.) |
| [VanGoH](https://github.com/auroratechnologies/vangoh) | Configurable [AWS-Style](http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html) HMAC authentication middleware |
| [xrequestid](https://github.com/pilu/xrequestid) | Middleware that assigns a random X-Request-Id header to each request |
| [digits](https://github.com/bamarni/digits) | Middleware that handles [Twitter Digits](https://get.digits.com/) authentication |

> Feel free to put up your own middleware in this list!