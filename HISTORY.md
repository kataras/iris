# FAQ

### Looking for free support?

	http://support.iris-go.com
    https://kataras.rocket.chat/channel/iris

### Looking for previous versions?

    https://github.com/kataras/iris#-version


### Should I upgrade my Iris?

Developers are not forced to upgrade if they don't really need it. Upgrade whenever you feel ready.

> Iris uses the [vendor directory](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo) feature, so you get truly reproducible builds, as this method guards against upstream renames and deletes.

**How to upgrade**: Open your command-line and execute this command: `go get -u github.com/kataras/iris`.

# Tu, 01 August 2017 | v8.1.3

- Add `Option` function to the `html view engine`: https://github.com/kataras/iris/issues/694
- Fix sessions backend databases restore expiration: https://github.com/kataras/iris/issues/692 by @corebreaker
- Add `PartyFunc`, same as `Party` but receives a function with the sub router as its argument instead [GO1.9 Users-ONLY]

# Mo, 31 July 2017 | v8.1.2

Add a `ConfigureHost` function as an alternative way to customize the hosts via `host.Configurator`.
The first way was to pass `host.Configurator` as optional arguments on `iris.Runner`s built'n functions (`iris#Server, iris#Listener, iris#Addr, iris#TLS, iris#AutoTLS`), example of this can be found [there](https://github.com/kataras/iris/blob/master/_examples/http-listening/notify-on-shutdown).

Example Code:

```go
package main

import (
	stdContext "context"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/host"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx context.Context) {
		ctx.HTML("<h1>Hello, try to refresh the page after ~10 secs</h1>")
	})

    app.ConfigureHost(configureHost) // or pass "configureHost" as `app.Addr` argument, same result.

	app.Logger().Info("Wait 10 seconds and check your terminal again")
	// simulate a shutdown action here...
	go func() {
		<-time.After(10 * time.Second)
		timeout := 5 * time.Second
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
		defer cancel()
		// close all hosts, this will notify the callback we had register
		// inside the `configureHost` func.
		app.Shutdown(ctx)
	}()

	// http://localhost:8080
	// wait 10 seconds and check your terminal.
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}

func configureHost(su *host.Supervisor) {
	// here we have full access to the host that will be created
	// inside the `app.Run` or `app.NewHost` function .
	//
	// we're registering a shutdown "event" callback here:
	su.RegisterOnShutdown(func() {
		println("server is closed")
	})
	// su.RegisterOnError
	// su.RegisterOnServe
}
```

# Su, 30 July 2017

Greetings my friends, nothing special today, no version number yet.

We just improve the, external, Iris Logging library and the `Columns` config field from `middleware/logger` defaults to `false` now. Upgrade with `go get -u github.com/kataras/iris` and have fun!

# Sa, 29 July 2017 | v8.1.1

No breaking changes, just an addition to make your life easier.

This feature has been implemented after @corebreaker 's request, posted at: https://github.com/kataras/iris/issues/688. He was also tried to fix that by a [PR](https://github.com/kataras/iris/pull/689), we thanks him but the problem with that PR was the duplication and the separation of concepts, however we thanks him for pushing for a solution. The current feature's implementation gives a permant solution to host supervisor access issues.

Optional host configurators added to all common serve and listen functions.

Below you'll find how to gain access to the host, **the second way is the new feature.**

### Hosts

Access to all hosts that serve your application can be provided by
the `Application#Hosts` field, after the `Run` method.

But the most common scenario is that you may need access to the host before the `Run` method,
there are two ways of gain access to the host supervisor, read below.

First way is to use the `app.NewHost` to create a new host
and use one of its `Serve` or `Listen` functions
to start the application via the `iris#Raw` Runner.
Note that this way needs an extra import of the `net/http` package.

Example Code:

```go
h := app.NewHost(&http.Server{Addr:":8080"})
h.RegisterOnShutdown(func(){
    println("server was closed!")
})

app.Run(iris.Raw(h.ListenAndServe))
```

Second, and probably easier way is to use the `host.Configurator`.

Note that this method requires an extra import statement of
"github.com/kataras/iris/core/host" when using go < 1.9,
if you're targeting on go1.9 then you can use the `iris#Supervisor`
and omit the extra host import.

All common `Runners` we saw earlier (`iris#Addr, iris#Listener, iris#Server, iris#TLS, iris#AutoTLS`)
accept a variadic argument of `host.Configurator`, there are just `func(*host.Supervisor)`.
Therefore the `Application` gives you the rights to modify the auto-created host supervisor through these.


Example Code:

```go
package main

import (
    stdContext "context"
    "time"

    "github.com/kataras/iris"
    "github.com/kataras/iris/context"
    "github.com/kataras/iris/core/host"
)

func main() {
    app := iris.New()

    app.Get("/", func(ctx context.Context) {
        ctx.HTML("<h1>Hello, try to refresh the page after ~10 secs</h1>")
    })

    app.Logger().Info("Wait 10 seconds and check your terminal again")
    // simulate a shutdown action here...
    go func() {
        <-time.After(10 * time.Second)
        timeout := 5 * time.Second
        ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
        defer cancel()
        // close all hosts, this will notify the callback we had register
        // inside the `configureHost` func.
        app.Shutdown(ctx)
    }()

    // start the server as usual, the only difference is that
    // we're adding a second (optional) function
    // to configure the just-created host supervisor.
    //
    // http://localhost:8080
    // wait 10 seconds and check your terminal.
    app.Run(iris.Addr(":8080", configureHost), iris.WithoutServerError(iris.ErrServerClosed))

}

func configureHost(su *host.Supervisor) {
    // here we have full access to the host that will be created
    // inside the `Run` function.
    //
    // we register a shutdown "event" callback
    su.RegisterOnShutdown(func() {
        println("server is closed")
    })
    // su.RegisterOnError
    // su.RegisterOnServe
}
```

Read more about listening and gracefully shutdown by navigating to: https://github.com/kataras/iris/tree/master/_examples/#http-listening

# We, 26 July 2017 | v8.1.0

The `app.Logger() *logrus.Logger` was replaced with a custom implementation [[golog](https://github.com/kataras/golog)], it's compatible with the [logrus](https://github.com/sirupsen/logrus) package and other open-source golang loggers as well, because of that: https://github.com/kataras/iris/issues/680#issuecomment-316184570. 

The API didn't change much except these:

-  the new implementation does not recognise `Fatal` and `Panic` because, actually, iris never panics
- the old `app.Logger().Out = io.Writer` should be written as `app.Logger().SetOutput(io.Writer)`

The new implementation, [golog](https://github.com/kataras/golog) is featured, **[three times faster than logrus](https://github.com/kataras/golog/tree/master/_benchmarks)**
and it completes every common usage.

### Integration

I understand that many of you may use logrus outside of Iris too. To integrate an external `logrus` logger just 
`Install` it-- all print operations will be handled by the provided `logrus instance`.

```go
import (
    "github.com/kataras/iris"
    "github.com/sirupsen/logrus"
)

package main(){
    app := iris.New()
    app.Logger().Install(logrus.StandardLogger()) // the package-level logrus instance
    // [...]
}
```

For more information about our new logger please navigate to: https://github.com/kataras/golog -  contributions are welcomed as well!

# Sa, 23 July 2017 | v8.0.7

Fix [It's true that with UseGlobal the "/path1.txt" route call the middleware but cause the prepend, the order is inversed](https://github.com/kataras/iris/issues/683#issuecomment-317229068)

# Sa, 22 July 2017 | v8.0.5 & v8.0.6

No API Changes.

### Performance

Add an experimental [Configuration#EnableOptimizations](https://github.com/kataras/iris/blob/master/configuration.go#L170) option.

```go
type Configuration {
    // [...]

    // EnableOptimization when this field is true
    // then the application tries to optimize for the best performance where is possible.
    //
    // Defaults to false.
    EnableOptimizations bool `yaml:"EnableOptimizations" toml:"EnableOptimizations"`

    // [...]
}
```

Usage:

```go
app.Run(iris.Addr(":8080"), iris.WithOptimizations)
```

### Django view engine

@corebreaker pushed a [PR](https://github.com/kataras/iris/pull/682) to solve the [Problem for {%extends%} in Django Engine with embedded files](https://github.com/kataras/iris/issues/681).

### Logger

Remove the `vendor/github.com/sirupsen/logrus` folder, as a temporary solution for the https://github.com/kataras/iris/issues/680#issuecomment-316196126.

#### Future versions

The logrus will be replaced with a custom implementation, because of that: https://github.com/kataras/iris/issues/680#issuecomment-316184570. 

As far as we know, @kataras is working on this new implementation, see [here](https://github.com/kataras/iris/issues/680#issuecomment-316544906), 
which will be compatible with the logrus package and other open-source golang loggers as well.


# Mo, 17 July 2017 | v8.0.4

No API changes.

### HTTP Errors

Fix a rare behavior: error handlers are not executed correctly
when a before-handler by-passes the order of execution, relative to the [previous feature](https://github.com/kataras/iris/blob/master/HISTORY.md#su-16-july-2017--v803). 

### Request Logger

Add `Configuration#MessageContextKey`. Example can be found at [_examples/http_request/request-logger/main.go](https://github.com/kataras/iris/blob/master/_examples/http_request/request-logger/main.go#L48).

# Su, 16 July 2017 | v8.0.3

No API changes.

Relative issues: 

- https://github.com/kataras/iris/issues/674
- https://github.com/kataras/iris/issues/675
- https://github.com/kataras/iris/issues/676

### HTTP Errors

Able to register a chain of Handlers (and middleware with `ctx.Next()` support like routes) for a specific error code, read more at [issues/674](https://github.com/kataras/iris/issues/674). Usage example can be found at [_examples/http_request/request-logger/main.go](https://github.com/kataras/iris/blob/master/_examples/http_request/request-logger/main.go#L41).


New function to register a Handler or a chain of Handlers for all official http error codes, by calling the new `app.OnAnyErrorCode(func(ctx context.Context){})`, read more at [issues/675](https://github.com/kataras/iris/issues/675). Usage example can be found at [_examples/http_request/request-logger/main.go](https://github.com/kataras/iris/blob/master/_examples/http_request/request-logger/main.go#L42).

### Request Logger

Add `Configuration#LogFunc` and `Configuration#Columns` fields, read more at [issues/676](https://github.com/kataras/iris/issues/676). Example can be found at [_examples/http_request/request-logger/request-logger-file/main.go](https://github.com/kataras/iris/blob/master/_examples/http_request/request-logger/request-logger-file/main.go).


Have fun and don't forget to [star](https://github.com/kataras/iris/stargazers) the github repository, it gives me power to continue publishing my work!

# Sa, 15 July 2017 | v8.0.2

Okay my friends, this is a good time to upgrade, I did implement a feature that you were asking many times at the past.

Iris' router can now handle root-level wildcard paths `app.Get("/{paramName:path})`.

In case you're wondering: no it does not conflict with other static or dynamic routes, meaning that you can code something like this:

```go
// it isn't conflicts with the rest of the static routes or dynamic routes with a path prefix.
app.Get("/{pathParamName:path}", myHandler) 
```

Or even like this:

```go
package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()

	// this works as expected now,
	// will handle all GET requests
	// except:
	// /                     -> because of app.Get("/", ...)
	// /other/anything/here  -> because of app.Get("/other/{paramother:path}", ...)
	// /other2/anything/here -> because of app.Get("/other2/{paramothersecond:path}", ...)
	// /other2/static        -> because of app.Get("/other2/static", ...)
	//
	// It isn't conflicts with the rest of the routes, without routing performance cost!
	//
	// i.e /something/here/that/cannot/be/found/by/other/registered/routes/order/not/matters
	app.Get("/{p:path}", h)

	// this will handle only GET /
	app.Get("/", staticPath)

	// this will handle all GET requests starting with "/other/"
	//
	// i.e /other/more/than/one/path/parts
	app.Get("/other/{paramother:path}", other)

	// this will handle all GET requests starting with "/other2/"
	// except /other2/static (because of the next static route)
	//
	// i.e /other2/more/than/one/path/parts
	app.Get("/other2/{paramothersecond:path}", other2)

	// this will handle only GET /other2/static
	app.Get("/other2/static", staticPath)

	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}

func h(ctx context.Context) {
	param := ctx.Params().Get("p")
	ctx.WriteString(param)
}

func other(ctx context.Context) {
	param := ctx.Params().Get("paramother")
	ctx.Writef("from other: %s", param)
}

func other2(ctx context.Context) {
	param := ctx.Params().Get("paramothersecond")
	ctx.Writef("from other2: %s", param)
}

func staticPath(ctx context.Context) {
	ctx.Writef("from the static path: %s", ctx.Path())
}
``` 

If you find any bugs with this change please send me a [chat message](https://kataras.rocket.chat/channel/iris) in order to investigate it, I'm totally free at weekends.

Have fun and don't forget to [star](https://github.com/kataras/iris/stargazers) the github repository, it gives me power to continue publishing my work!

# Th, 13 July 2017 | v8.0.1

Nothing tremendous at this minor version.

We've just added a configuration field in order to ignore errors received by the `Run` function, see below.

[Configuration#IgnoreServerErrors](https://github.com/kataras/iris/blob/master/configuration.go#L255)
```go
type Configuration struct {
    // [...]

    // IgnoreServerErrors will cause to ignore the matched "errors"
    // from the main application's `Run` function.
    // This is a slice of string, not a slice of error
    // users can register these errors using yaml or toml configuration file
    // like the rest of the configuration fields.
    //
    // See `WithoutServerError(...)` function too.
    //
    // Defaults to an empty slice.
    IgnoreServerErrors []string `yaml:"IgnoreServerErrors" toml:"IgnoreServerErrors"`

    // [...]
}
```
[Configuration#WithoutServerError](https://github.com/kataras/iris/blob/master/configuration.go#L106)
```go
// WithoutServerError will cause to ignore the matched "errors"
// from the main application's `Run` function.
//
// Usage:
// err := app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
// will return `nil` if the server's error was `http/iris#ErrServerClosed`.
//
// See `Configuration#IgnoreServerErrors []string` too.
WithoutServerError(errors ...error) Configurator
```

By default no error is being ignored, of course.

Example code:
[_examples/http-listening/listen-addr/omit-server-errors](https://github.com/kataras/iris/tree/master/_examples/http-listening/listen-addr/omit-server-errors)
```go
package main

import (
    "github.com/kataras/iris"
    "github.com/kataras/iris/context"
)

func main() {
    app := iris.New()

    app.Get("/", func(ctx context.Context) {
    	ctx.HTML("<h1>Hello World!/</h1>")
    })

    err := app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
    if err != nil {
        // do something
    }
    // same as:
    // err := app.Run(iris.Addr(":8080"))
    // if err != nil && (err != iris.ErrServerClosed || err.Error() != iris.ErrServerClosed.Error()) {
    //     [...]
    // }
}
```

At first we didn't want to implement something like that because it's ridiculous easy to do it manually but a second thought came to us,
that many applications are based on configuration, therefore it would be nice to have something to ignore errors
by simply string values that can be passed to the application's configuration via `toml` or `yaml` files too.

This feature has been implemented after a request of ignoring the `iris/http#ErrServerClosed` from the `Run` function: 
https://github.com/kataras/iris/issues/668

# Mo, 10 July 2017 | v8.0.0

## 📈 One and a half years with Iris and You...

Despite the deflamations, the clickbait articles, the removed posts of mine at reddit/r/golang, the unexpected and inadequate ban from the gophers slack room by @dlsniper alone the previous week without any reason or inform, Iris is still here and will be.

- 7070 github stars
- 749 github forks
- 1m total views at its documentation
- ~800$ at donations (there're a lot for a golang open-source project, thanks to you)
- ~550 reported bugs fixed
- ~30 community feature requests have been implemented

## 🔥 Reborn

As you may have heard I have huge responsibilities on my new position at Dubai nowadays, therefore I don't have the needed time to work on this project anymore.

After a month of negotiations and searching I succeed to find a decent software engineer to continue my work on the open source community.

The leadership of this, open-source, repository was transferred to [hiveminded](https://github.com/hiveminded), the author of iris-based [get-ion/ion](https://github.com/get-ion/ion), he actually did an excellent job on the framework, he kept the code as minimal as possible and at the same time added more features, examples and middleware(s).

These types of projects need heart and sacrifices to continue offer the best developer experience like a paid software, please do support him as you did with me!

## 📰 Changelog

> app. = `app := iris.New();` **app.**

> ctx. = `func(ctx context.Context) {` **ctx.** `}`

### Docker

Docker and kubernetes integration showcase, see the [iris-contrib/cloud-native-go](https://github.com/iris-contrib/cloud-native-go) repository as an example.

### Logger

* Logger which was an `io.Writer` was replaced with the pluggable `logrus`.
    * which you still attach an `io.Writer` with `app.Logger().Out = an io.Writer`.
    * iris as always logs only critical errors, you can disable them with `app.Logger().Level = iris.NoLog`
    * the request logger outputs the incoming requests as INFO level.

### Sessions

Remove `ctx.Session()` and `app.AttachSessionManager`, devs should import and use the `sessions` package as standalone, it's totally optional, devs can use any other session manager too. [Examples here](sessions#table-of-contents).

### Websockets

The `github.com/kataras/iris/websocket` package does not handle the endpoint and client side automatically anymore. Example code:

```go
func setupWebsocket(app *iris.Application) {
    // create our echo websocket server
    ws := websocket.New(websocket.Config{
    	ReadBufferSize:  1024,
    	WriteBufferSize: 1024,
    })
    ws.OnConnection(handleConnection)
    // serve the javascript built'n client-side library,
    // see weboskcets.html script tags, this path is used.
    app.Any("/iris-ws.js", func(ctx context.Context) {
    	ctx.Write(websocket.ClientSource)
    })

    // register the server on an endpoint.
    // see the inline javascript code in the websockets.html, this endpoint is used to connect to the server.
    app.Get("/echo", ws.Handler())
}
```

> More examples [here](websocket#table-of-contents)

### View

Rename `app.AttachView(...)` to `app.RegisterView(...)`.

Users can omit the import of `github.com/kataras/iris/view` and use the `github.com/kataras/iris` package to
refer to the view engines, i.e: `app.RegisterView(iris.HTML("./templates", ".html"))` is the same as `import "github.com/kataras/iris/view" [...] app.RegisterView(view.HTML("./templates" ,".html"))`.

> Examples [here](_examples/#view)

### Security

At previous versions, when you called `ctx.Remoteaddr()` Iris could parse and return the client's IP from the "X-Real-IP", "X-Forwarded-For" headers. This was a security leak as you can imagine, because the user can modify them. So we've disabled these headers by-default and add an option to add/remove request headers that are responsible to parse and return the client's real IP.

```go
// WithRemoteAddrHeader enables or adds a new or existing request header name
// that can be used to validate the client's real IP.
//
// Existing values are:
// "X-Real-Ip":             false,
// "X-Forwarded-For":       false,
// "CF-Connecting-IP": false
//
// Look `context.RemoteAddr()` for more.
WithRemoteAddrHeader(headerName string) Configurator // enables a header.
WithoutRemoteAddrHeader(headerName string) Configurator // disables a header.
```
For example, if you want to enable the "CF-Connecting-IP" header (cloudflare) 
you have to add the `WithRemoteAddrHeader` option to the `app.Run` function, at the end of your program.

```go
app.Run(iris.Addr(":8080"), iris.WithRemoteAddrHeader("CF-Connecting-IP"))
// This header name will be checked when ctx.RemoteAddr() called and if exists
// it will return the client's IP, otherwise it will return the default *http.Request's `RemoteAddr` field.
```

### Miscellaneous

Fix [typescript tools](typescript).

[_examples](_examples/) folder has been ordered by feature and usage:
    - contains tests on some examples
    - new examples added, one of them shows how the `reuseport` feature on UNIX and BSD systems can be used to listen for incoming connections, [see here](_examples/#http-listening)


Replace supervisor's tasks with events, like `RegisterOnShutdown`, `RegisterOnError`, `RegisterOnServe` and fix the (unharmful) race condition when output the banner to the console. Global notifier for interrupt signals which can be disabled via `app.Run([...], iris.WithoutInterruptHandler)`, look [graceful-shutdown](_examples/http-listening/graceful-shutdown/main.go) example for more.


More handlers are ported to Iris (they can be used as they are without `iris.FromStd`), these handlers can be found at [iris-contrib/middleware](https://github.com/iris-contrib/middleware). Feel free to put your own there.


| Middleware | Description | Example |
| -----------|--------|-------------|
| [jwt](https://github.com/iris-contrib/middleware/tree/master/jwt) | Middleware checks for a JWT on the `Authorization` header on incoming requests and decodes it. | [iris-contrib/middleware/jwt/_example](https://github.com/iris-contrib/middleware/tree/master/jwt/_example) |
| [cors](https://github.com/iris-contrib/middleware/tree/master/cors) | HTTP Access Control. | [iris-contrib/middleware/cors/_example](https://github.com/iris-contrib/middleware/tree/master/cors/_example) |
| [secure](https://github.com/iris-contrib/middleware/tree/master/secure) | Middleware that implements a few quick security wins. | [iris-contrib/middleware/secure/_example](https://github.com/iris-contrib/middleware/tree/master/secure/_example/main.go) |
| [tollbooth](https://github.com/iris-contrib/middleware/tree/master/tollboothic) | Generic middleware to rate-limit HTTP requests. | [iris-contrib/middleware/tollbooth/_examples/limit-handler](https://github.com/iris-contrib/middleware/tree/master/tollbooth/_examples/limit-handler) |
| [cloudwatch](https://github.com/iris-contrib/middleware/tree/master/cloudwatch) |  AWS cloudwatch metrics middleware. |[iris-contrib/middleware/cloudwatch/_example](https://github.com/iris-contrib/middleware/tree/master/cloudwatch/_example) |
| [new relic](https://github.com/iris-contrib/middleware/tree/master/newrelic) | Official [New Relic Go Agent](https://github.com/newrelic/go-agent). | [iris-contrib/middleware/newrelic/_example](https://github.com/iris-contrib/middleware/tree/master/newrelic/_example) |
| [prometheus](https://github.com/iris-contrib/middleware/tree/master/prometheus)| Easily create metrics endpoint for the [prometheus](http://prometheus.io) instrumentation tool | [iris-contrib/middleware/prometheus/_example](https://github.com/iris-contrib/middleware/tree/master/prometheus/_example) |


v7.x is deprecated because it sold as it is and it is not part of the public, stable `gopkg.in` iris versions. Developers/users of this library should upgrade their apps to v8.x, the refactor process will cost nothing for most of you, as the most common API remains as it was. The changelog history from that are being presented below.


# Th, 15 June 2017 | v7.2.0

### About our new home page
    http://iris-go.com

Thanks to [Santosh Anand](https://github.com/santoshanand) the http://iris-go.com has been upgraded and it's really awesome!

[Santosh](https://github.com/santoshanand) is a freelancer, he has a great knowledge of nodejs and express js, Android, iOS, React Native, Vue.js etc, if you need a developer to find or create a solution for your problem or task, please contact with him.


The amount of the next two or three donations you'll send they will be immediately transferred to his own account balance, so be generous please!

### Cache

Declare the `iris.Cache alias` to the new, improved and most-suited for common usage, `cache.Handler function`.

`iris.Cache` be used as middleware in the chain now, example [here](_examples/intermediate/cache-markdown/main.go). However [you can still use the cache as a wrapper](cache/cache_test.go) by importing the `github.com/kataras/iris/cache` package. 


### File server

- **Fix** [that](https://github.com/iris-contrib/community-board/issues/12).

- `app.StaticHandler(requestPath string, systemPath string, showList bool, gzip bool)` -> `app.StaticHandler(systemPath,showList bool, gzip bool)`

- **New** feature for Single Page Applications, `app.SPA(assetHandler context.Handler)` implemented.

- **New** `app.StaticEmbeddedHandler(vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string)` added in order to be able to pass that on `app.SPA(app.StaticEmbeddedHandler("./public", Asset, AssetNames))`.

- **Fix** `app.StaticEmbedded(requestPath string, vdir string, assetFn func(name string) ([]byte, error), namesFn func() []string)`.

Examples: 
- [Embedding Files Into Executable App](_examples/file-server/embedding-files-into-app)
- [Single Page Application](_examples/file-server/single-page-application)
- [Embedding Single Page Application](_examples/file-server/embedding-single-page-application)

> [app.StaticWeb](_examples/file-server/basic/main.go) doesn't works for root request path "/"  anymore, use the new `app.SPA` instead.   

### WWW subdomain entry

- [Example](_examples/subdomains/www/main.go) added to copy all application's routes, including parties, to the `www.mydomain.com`


### Wrapping the Router

- [Example](_examples/routing/custom-wrapper/main.go) added to show you how you can use the `app.WrapRouter` 
to implement a similar to `app.SPA` functionality, don't panic, it's easier than it sounds.


### Testing

- `httptest.New(app *iris.Application, t *testing.T)` -> `httptest.New(t *testing.T, app *iris.Application)`.

- **New** `httptest.NewLocalListener() net.Listener` added.
- **New** `httptest.NewLocalTLSListener(tcpListener net.Listener) net.Listener` added.

Useful for testing tls-enabled servers: 

Proxies are trying to understand local addresses in order to allow `InsecureSkipVerify`.

-  `host.ProxyHandler(target *url.URL) *httputil.ReverseProxy`.
-  `host.NewProxy(hostAddr string, target *url.URL) *Supervisor`.
        
    Tests [here](core/host/proxy_test.go).

# Tu, 13 June 2017 | v7.1.1

Fix [that](https://github.com/iris-contrib/community-board/issues/11).

# Mo, 12 June 2017 | v7.1.0

Fix [that](https://github.com/iris-contrib/community-board/issues/10).


# Su, 11 June 2017 | v7.0.5

Iris now supports static paths and dynamic paths for the same path prefix with zero performance cost:

`app.Get("/profile/{id:int}", handler)` and `app.Get("/profile/create", createHandler)` are not in conflict anymore.


The rest of the special Iris' routing features, including static & wildcard subdomains are still work like a charm.

> This was one of the most popular community's feature requests. Click [here](https://github.com/kataras/iris/blob/master/_examples/beginner/routing/overview/main.go) to see a trivial example.

# Sa, 10 June 2017 | v7.0.4

- Simplify and add a test for the [basicauth middleware](https://github.com/kataras/iris/tree/master/middleware/basicauth), no need to be
stored inside the Context anymore, developers can get the validated user(username and password) via `context.Request().BasicAuth()`. `basicauth.Config.ContextKey` was removed, just remove that field from your configuration, it's useless now. 

# Sa, 10 June 2017 | v7.0.3

- New `context.Session().PeekFlash("key")` added, unlike `GetFlash` this will return the flash value but keep the message valid for the next requests too.
- Complete the [httptest example](https://github.com/iris-contrib/examples/tree/master/httptest).
- Fix the (marked as deprecated) `ListenLETSENCRYPT` function.
- Upgrade the [iris-contrib/middleware](https://github.com/iris-contrib/middleware) including JWT, CORS and Secure handlers.
- Add [OAuth2 example](https://github.com/iris-contrib/examples/tree/master/oauth2) -- showcases the third-party package [goth](https://github.com/markbates/goth) integration with Iris.

### Community

 - Add github integration on https://kataras.rocket.chat/channel/iris , so users can login with their github accounts instead of creating new for the chat only.

# Th, 08 June 2017 | v7.0.2

- Able to set **immutable** data on sessions and context's storage. Aligned to fix an issue on slices and maps as reported [here](https://github.com/iris-contrib/community-board/issues/5).

# We, 07 June 2017 | v7.0.1

- Proof of concept of an internal release generator, navigate [here](https://github.com/iris-contrib/community-board/issues/2) to read more. 
- Remove tray icon "feature", click [here](https://github.com/iris-contrib/community-board/issues/1) to learn why.

# Sa, 03 June 2017 

After 2+ months of hard work and collaborations, Iris [version 7](https://github.com/kataras/iris) was published earlier today.

If you're new to Iris you don't have to read all these, just navigate to the [updated examples](https://github.com/kataras/iris/tree/master/_examples) and you should be fine:)

Note that this section will not
cover the internal changes, the difference is so big that anybody can see them with a glimpse, even the code structure itself.


## Changes from [v6](https://github.com/kataras/iris/tree/v6)

The whole framework was re-written from zero but I tried to keep the most common public API that iris developers use.

Vendoring /w update 

The previous vendor action for v6 was done by-hand, now I'm using the [go dep](https://github.com/golang/dep) tool, I had to do
some small steps:

- remove files like testdata to reduce the folder size
- rollback some of the "golang/x/net/ipv4" and "ipv6" source files because they are downloaded to their latest versions
by go dep, but they had lines with the `typealias` feature, which is not ready by current golang version (it will be on August)
- fix "cannot use internal package" at golang/x/net/ipv4 and ipv6 packages
	- rename the interal folder to was-internal, everywhere and fix its references.
- fix "main redeclared in this block"
	- remove all examples folders.
- remove main.go files on jsondiff lib, used by gavv/httpexpect, produces errors on `test -v ./...` while jd and jp folders are not used at all.

The go dep tool does what is says, as expected, don't be afraid of it now.
I am totally recommending this tool for package authors, even if it's in its alpha state.
I remember when Iris was in its alpha state and it had 4k stars on its first weeks/or month and that helped me a lot to fix reported bugs by users and make the framework even better, so give love to go dep from today!

General

- Several enhancements for the typescript transpiler, view engine, websocket server and sessions manager
- All `Listen` methods replaced with a single `Run` method, see [here](https://github.com/kataras/iris/tree/master/_examples/beginner/listening)
- Configuration, easier to modify the defaults, see [here](https://github.com/kataras/iris/tree/master/_examples/beginner/cofiguration)
- `HandlerFunc` removed, just `Handler` of `func(context.Context)` where context.Context derives from `import "github.com/kataras/iris/context"` (on August this import path will be optional)
    - Simplify API, i.e: instead of `Handle,HandleFunc,Use,UseFunc,Done,DoneFunc,UseGlobal,UseGlobalFunc` use `Handle,Use,Done,UseGlobal`.
- Response time decreased even more (9-35%, depends on the application)
- The `Adaptors` idea replaced with a more structural design pattern, but you have to apply these changes: 
    - `app.Adapt(view.HTML/Pug/Amber/Django/Handlebars...)` -> `app.AttachView(view.HTML/Pug/Amber/Django/Handlebars...)` 
    - `app.Adapt(sessions.New(...))` -> `app.AttachSessionManager(sessions.New(...))`
    - `app.Adapt(iris.LoggerPolicy(...))` -> `app.AttachLogger(io.Writer)`
    - `app.Adapt(iris.RenderPolicy(...))` -> removed and replaced with the ability to replace the whole context with a custom one or override some methods of it, see below.

Routing
- Remove of multiple routers, now we have the fresh Iris router which is based on top of the julien's [httprouter](https://github.com/julienschmidt/httprouter).
    > Update 11 June 2017: As of 7.0.5 this is changed, read [here](https://github.com/kataras/iris/blob/master/HISTORY.md#su-11-june-2017--v705).
- Subdomains routing algorithm has been improved.
- Iris router is using a custom interpreter with parser and path evaluator to achieve the best expressiveness, with zero performance loss, you ever seen so far, i.e: 
    - `app.Get("/", "/users/{userid:int min(1)}", handler)`,
        - `{username:string}` or just `{username}`
        - `{asset:path}`,
        - `{firstname:alphabetical}`,
        - `{requestfile:file}` ,
        - `{mylowercaseParam regexp([a-z]+)}`.
        - The previous syntax of `:param` and `*param` still working as expected. Previous rules for paths confliction remain as they were.
            - Also, path parameter names should be only alphabetical now, numbers and symbols are not allowed (for your own good, I have seen a lot the last year...).

Click [here](https://github.com/kataras/iris/tree/master/_examples/beginner/routing) for details.
> It was my first attempt/experience on the interpreters field, so be good with it :)

Context
- `iris.Context pointer` replaced with `context.Context interface` as we already mention
    - in order to be able to use a custom context and/or catch lifetime like `BeginRequest` and `EndRequest` from context itself, see below
- `context.JSON, context.JSONP, context.XML, context.Markdown, context.HTML` work faster
- `context.Render("filename.ext", bindingViewData{}, options) ` -> `context.View("filename.ext")`
    - `View` renders only templates, it will not try to search if you have a restful renderer adapted, because, now, you can do it via method overriding using a custom Context.
    - Able to set `context.ViewData` and `context.ViewLayout` via middleware when executing a template.
- `context.SetStatusCode(statusCode)` -> `context.StatusCode(statusCode)`
    - which is equivalent with the old `EmitError` too:
        - if status code >=400 given can automatically fire a custom http error handler if response wasn't written already.
    - `context.StatusCode()` -> `context.GetStatusCode()`
    - `app.OnError` -> `app.OnErrorCode`
    - Errors per party are removed by-default, you can just use one global error handler with logic like "if path starts with 'prefix' fire this error handler, else...". 
- Easy way to change Iris' default `Context` with a custom one, see [here](https://github.com/kataras/iris/tree/master/_examples/intermediate/custom-context)
- `context.ResponseWriter().SetBeforeFlush(...)` works for Flush and HTTP/2 Push, respectfully
- Several improvements under the `Request transactions` 
- Remember that you had to set a status code on each of the render-relative methods? Now it's not required, it just renders
with the status code that user gave with `context.StatusCode` or with `200 OK`, i.e:
    -`context.JSON(iris.StatusOK, myJSON{})` -> `context.JSON(myJSON{})`.
    - Each one of the context's render methods has optional per-call settings,
    - **the new API is even more easier to read, understand and use.**

Server
- Able to set custom underline *http.Server(s) with new Host (aka Server Supervisor) feature 
    - `Done` and `Err` channels to catch shutdown or any errors on custom hosts,
    - Schedule custom tasks(with cancelation) when server is running, see [here](https://github.com/kataras/iris/tree/master/_examples/intermediate/graceful-shutdown)
- Interrupt handler task for gracefully shutdown (when `CTRL/CMD+C`) are enabled by-default, you can disable its via configuration: `app.Run(iris.Addr(":8080"), iris.WithoutInterruptHandler)`

Future plans
- Future Go1.9's [ServeTLS](https://go-review.googlesource.com/c/38114/2/src/net/http/server.go) is ready when 1.9 released
- Future Go1.9's typealias feature is ready when 1.9 released, i.e `context.Context` -> `iris.Context` just one import path instead of todays' two.