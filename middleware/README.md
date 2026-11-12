# Middleware

The Iris web framework provides a variety of built-in middleware handlers to extend and enhance your application. Middleware handlers are functions that can intercept and process incoming HTTP requests before they reach your application's routes. They can perform tasks such as logging, authentication, rate limiting, and more. Middleware is just a `func(iris.Context)` which runs before (or after) the main route's handler. Example of a middleware handler:

```go
package main

import (
    "github.com/kataras/iris/v14"
)

func main() {
    app := iris.New()

    // Register a custom middleware handler.
    app.Use(func(ctx iris.Context) {
        ctx.Application().Logger().Infof("Request received: %s", ctx.Path())
        ctx.Next()
    })

    app.Get("/", func(ctx iris.Context) {
        ctx.HTML("Hello, world!")
    })

    app.Listen(":8080")
}
```

Use `app.Use` to apply middleware for all routes, including 404s and 405s. Use `childRouter.Use` to apply it for sub-routes only, excluding 404s and 405s. Use `app.UseRouter` to register middleware before the routing process (where you can modify request parts such as the URL path), or attach it directly to specific routes with `app.Get`, `app.Post`, and so on.

## Built-in Middleware

| Middleware | Description | Documentation |
| -----------|--------|-------------|
| [AccessLog](accesslog) | The **AccessLog** middleware provides configurable access logging for your application. It records incoming HTTP requests and responses for traffic monitoring, debugging and audits. The log format and output destination are configurable. | [accesslog/README](https://github.com/kataras/iris/blob/main/middleware/accesslog/README.md) |
| [API Key](apikey) | The **API Key** middleware guards routes with a shared secret that the caller presents on every request, which is how services, cron jobs and webhooks authenticate when there is no user to sign in. Keys are compared in constant time, may be stored as hashes or resolved from a database, and the principal that matched is available to the handlers. | [apikey/README](https://github.com/kataras/iris/blob/main/middleware/apikey/README.md) |
| [Basic Authentication](basicauth) | The **Basic Authentication** middleware implements HTTP Basic Authentication as defined in [RFC 7617](https://datatracker.ietf.org/doc/html/rfc7617). It secures your routes by requiring clients to provide a username and password with each request. This is a simple way to protect endpoints or restrict access to certain parts of your application. | [basicauth/README](https://github.com/kataras/iris/blob/main/middleware/basicauth/README.md) |
| [Request Body Limit](bodylimit) | The **Request Body Limit** middleware caps the size of incoming request bodies. It helps prevent resource exhaustion attacks by rejecting requests that exceed the specified limit. | [bodylimit/README](https://github.com/kataras/iris/blob/main/middleware/bodylimit/README.md) |
| [Cache](cache) | The **Cache** middleware enables caching of HTTP responses. It helps improve performance by storing responses for repeated requests on the client-side, reducing server load and response times. It uses headers for clients to cache the response. | [cache/README](https://github.com/kataras/iris/blob/main/middleware/cache/README.md) |
| [HTTP Cache](cache/httpcache) | The **HTTP Cache** middleware enables caching of HTTP responses. It's server-side caching mechanism. | [cache/httpcache/README](https://github.com/kataras/iris/blob/main/middleware/cache/httpcache/README.md) |
| [HTTP Compression](compress) | The **Compress** middleware applies gzip or brotli compression to HTTP responses and client requests decompression. It reduces the size of data transmitted to clients, improving load times and bandwidth utilization. | [compress/README](https://github.com/kataras/iris/blob/main/middleware/compress/README.md) |
| [CORS](cors) | The **CORS** middleware provides Cross-Origin Resource Sharing (CORS) support, allowing your application to handle requests from different origins securely. It sets the appropriate HTTP headers based on your configuration. | [cors/README](https://github.com/kataras/iris/blob/main/middleware/cors/README.md) |
| [Counter](counter) | The **Counter** middleware tracks the number of requests or custom tasks processed by your application. It can be used for monitoring, analytics, or rate-limiting purposes. | [counter/README](https://github.com/kataras/iris/blob/main/middleware/counter/README.md) |
| [Geolocation](geolocation) | The **Geolocation** middleware enriches incoming requests with geolocation data based on the client's IP address. This can be useful for analytics, content customization, or access control. | [geolocation/README](https://github.com/kataras/iris/blob/main/middleware/geolocation/README.md) |
| [gRPC](grpc) | The **gRPC** middleware integrates gRPC services with your Iris application. It serves gRPC requests alongside ordinary HTTP requests on the same port. | [grpc/README](https://github.com/kataras/iris/blob/main/middleware/grpc/README.md) |
| [hCAPTCHA](hcaptcha) | The **hCAPTCHA** middleware provides integration with the [hCaptcha](https://www.hcaptcha.com/) service, adding CAPTCHA verification to your forms to protect against bots and spam. | [hcaptcha/README](https://github.com/kataras/iris/blob/main/middleware/hcaptcha/README.md) |
| [HTTP Cost](httpcost) | The **HTTP Cost** middleware monitors requests' resource usage (time, memory, CPU) to produce a cost metric for performance analysis or billing. | [httpcost/README](https://github.com/kataras/iris/blob/main/middleware/httpcost/README.md) |
| [Localization](i18n) | The **Localization** middleware serves each client in its own language. It matches the request against the loaded locales through the `Accept-Language` header, a URL parameter or the subdomain, and exposes the translations to handlers and templates. | [i18n/README](https://github.com/kataras/iris/blob/main/middleware/i18n/README.md) |
| [IP Access Control](ipaccess) | The **IP Access Control** middleware decides who reaches your application based on the client's IP address. Allowed and blocked patterns can be exact addresses, CIDR ranges, or regular expressions. | [ipaccess/README](https://github.com/kataras/iris/blob/main/middleware/ipaccess/README.md) |
| [JWT](jwt) | The **JWT** middleware handles JSON Web Token (JWT) authentication. It validates the token a client presents and rejects the request when it does not hold up. | [jwt/README](https://github.com/kataras/iris/blob/main/middleware/jwt/README.md) |
| [Method Override](methodoverride) | The **Method Override** middleware allows clients to change the HTTP method of a request via a header or form field. This is useful when clients or proxies do not support certain HTTP methods like PUT or DELETE. | [methodoverride/README](https://github.com/kataras/iris/blob/main/middleware/methodoverride/README.md) |
| [Modrevision](modrevision) | The **Modrevision** middleware embeds versioning information into your application at build time. It helps in tracking and displaying build revisions, useful for debugging and monitoring. | [modrevision/README](https://github.com/kataras/iris/blob/main/middleware/modrevision/README.md) |
| [Simple Monitor](monitor) | The **Simple Monitor** middleware provides basic application health monitoring. It can be used to check the availability and performance of your application endpoints. | [monitor/README](https://github.com/kataras/iris/blob/main/middleware/monitor/README.md) |
| [Profiling](pprof) | The **Profiling** middleware integrates Go's built-in profiling tool `pprof` into your application. It assists in performance tuning by providing runtime profiling data. | [pprof/README](https://github.com/kataras/iris/blob/main/middleware/pprof/README.md) |
| [Rate Limit](rate) | The **Rate Limit** middleware restricts the number of requests a client can make in a given time period. It helps protect your application against abuse and denial-of-service attacks. |  [rate/README](https://github.com/kataras/iris/blob/main/middleware/rate/README.md) |
| [reCAPTCHA](recaptcha) | The **reCAPTCHA** middleware integrates Google's [reCAPTCHA](https://www.google.com/recaptcha/) service into your forms. It helps prevent spam and automated abuse by verifying that a user is human. | [recaptcha/README](https://github.com/kataras/iris/blob/main/middleware/recaptcha/README.md) |
| [Request & Response Recorder](recorder) | The **Request & Response Recorder** middleware captures and records HTTP requests and responses. It is useful for debugging, logging, or replaying requests. | [recorder/README](https://github.com/kataras/iris/blob/main/middleware/recorder/README.md) |
| [Panic Recovery](recovery) | The **Panic Recovery** middleware recovers from panics that occur during request handling, preventing the application from crashing. It sends a configurable HTTP response or executes a custom recovery function. | [recovery/README](https://github.com/kataras/iris/blob/main/middleware/recovery/README.md) |
| [Referrer](referrer) | The **Referrer** middleware provides utilities for parsing and validating the HTTP `Referer` header. It can be used for analytics or security measures like CSRF protection. | [referrer/README](https://github.com/kataras/iris/blob/main/middleware/referrer/README.md) |
| [Request ID](requestid) | The **Request ID** middleware assigns a unique identifier to each incoming HTTP request. This is useful for tracing requests in logs and debugging. | [requestid/README](https://github.com/kataras/iris/blob/main/middleware/requestid/README.md) |
| [URL Rewrite](rewrite) | The **Rewrite** middleware modifies incoming request URLs based on defined rules. It enables URL rewriting, redirection, or proxying without changing application logic. |  [rewrite/README](https://github.com/kataras/iris/blob/main/middleware/rewrite/README.md) |
| [Server Timing](servertiming) | The **Server Timing** middleware adds server timing metrics to HTTP responses using the `Server-Timing` header. This helps in measuring and communicating backend performance to clients. |  [servertiming/README](https://github.com/kataras/iris/blob/main/middleware/servertiming/README.md) |
| [Sessions](sessions) | The **Sessions** middleware provides session management using cookies and server-side storage. It keeps user data across requests, with signed and encrypted cookies by default. |  [sessions/README](https://github.com/kataras/iris/blob/main/middleware/sessions/README.md) |
| [API Versioning](versioning) | The **API Versioning** middleware runs several versions of your API side by side. It routes requests to the appropriate handlers based on versioning strategies like URL prefixes or headers. | [versioning/README](https://github.com/kataras/iris/blob/main/middleware/versioning/README.md) |
| [View](view) | The **View** middleware integrates template rendering engines into your application. It supports layouts, partials, and dynamic data binding for server-side rendered pages. | [view/README](https://github.com/kataras/iris/blob/main/middleware/view/README.md) |
| [WebSocket](websocket) | The **WebSocket** middleware provides real-time, bidirectional communication between clients and the server using the WebSocket protocol. It enables the development of interactive applications like chat systems, live feeds, and multiplayer games. | [websocket/README](https://github.com/kataras/iris/blob/main/middleware/websocket/README.md) |

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

Iris has its own middleware form of `func(ctx iris.Context)` but it's also compatible with all `net/http` middleware forms. See [here](https://github.com/kataras/iris/tree/main/_examples/16-advanced/std-interop).

Here's a small list of useful third-party handlers:

| Middleware | Description |
| -----------|-------------|
| [goth](https://github.com/markbates/goth) | OAuth, OAuth2 authentication. |
| [permissions2](https://github.com/xyproto/permissions2) | Cookies, users and permissions. |
| [csp](https://github.com/awakenetworks/csp) | [Content Security Policy](https://www.w3.org/TR/CSP2/) (CSP) support |
| [delay](https://github.com/jeffbmartinez/delay) | Add delays/latency to endpoints. Useful when testing effects of high latency |
| [onthefly](https://github.com/xyproto/onthefly) | Generate TinySVG, HTML and CSS on the fly |
| [RestGate](https://github.com/pjebs/restgate) | Secure authentication for REST API endpoints |
| [stats](https://github.com/thoas/stats) | Store information about your web application (response time, etc.) |
| [VanGoH](https://github.com/auroratechnologies/vangoh) | Configurable [AWS-Style](http://docs.aws.amazon.com/AmazonS3/latest/dev/RESTAuthentication.html) HMAC authentication middleware |
| [digits](https://github.com/bamarni/digits) | Middleware that handles [Twitter Digits](https://get.digits.com/) authentication |

> Add your own middleware to this list with a pull request.