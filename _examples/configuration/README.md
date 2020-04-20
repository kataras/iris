# Configuration

All configuration's values have default values, things will work as you expected with `iris.New()`.

Configuration is useless before listen functions, so it should be passed on `Application#Run/2` (second argument(s)).

Iris has a type named `Configurator` which is a `func(*iris.Application)`, any function
which completes this can be passed at `Application#Configure` and/or `Application#Run/2`.

`Application#ConfigurationReadOnly()` returns the configuration values.

`.Run` **by `Configuration` struct**

```go
package main

import (
    "github.com/kataras/iris/v12"
)

func main() {
    app := iris.New()
    app.Get("/", func(ctx iris.Context) {
        ctx.HTML("<b>Hello!</b>")
    })
    // [...]

    // Good when you want to modify the whole configuration.
    app.Listen(":8080", iris.WithConfiguration(iris.Configuration{
        DisableStartupLog:                 false,
        DisableInterruptHandler:           false,
        DisablePathCorrection:             false,
        EnablePathEscape:                  false,
        FireMethodNotAllowed:              false,
        DisableBodyConsumptionOnUnmarshal: false,
        DisableAutoFireStatusCode:         false,
        TimeFormat:                        "Mon, 02 Jan 2006 15:04:05 GMT",
        Charset:                           "utf-8",
    }))
}
```

`.Run` **by options**

```go
package main

import (
    "github.com/kataras/iris/v12"
)

func main() {
    app := iris.New()
    app.Get("/", func(ctx iris.Context) {
        ctx.HTML("<b>Hello!</b>")
    })
    // [...]

    // Good when you want to change some of the configuration's field.
    // Prefix: "With", code editors will help you navigate through all
    // configuration options without even a glitch to the documentation.

    app.Listen(":8080", iris.WithoutStartupLog, iris.WithCharset("utf-8"))

    // or before run:
    // app.Configure(iris.WithoutStartupLog, iris.WithCharset("utf-8"))
    // app.Listen(":8080")
}
```

`.Run` **by TOML config file**

```tml
DisablePathCorrection = false
EnablePathEscape = false
FireMethodNotAllowed = true
DisableBodyConsumptionOnUnmarshal = false
TimeFormat = "Mon, 01 Jan 2006 15:04:05 GMT"
Charset = "utf-8"

[Other]
	MyServerName = "iris"

```

```go
package main

import (
    "github.com/kataras/iris/v12"
)

func main() {
    app := iris.New()

    app.Get("/", func(ctx iris.Context) {
        ctx.HTML("<b>Hello!</b>")
    })
    // [...]

    // Good when you have two configurations, one for development and a different one for production use.
    app.Listen(":8080", iris.WithConfiguration(iris.TOML("./configs/iris.tml")))
}
```


`.Run` **by YAML config file**

```yml
DisablePathCorrection: false
EnablePathEscape: false
FireMethodNotAllowed: true
DisableBodyConsumptionOnUnmarshal: true
TimeFormat: Mon, 01 Jan 2006 15:04:05 GMT
Charset: UTF-8
```

```go
package main

import (
    "github.com/kataras/iris/v12"
)

func main() {
    app := iris.New()
    app.Get("/", func(ctx iris.Context) {
        ctx.HTML("<b>Hello!</b>")
    })
    // [...]

    app.Listen(":8080", iris.WithConfiguration(iris.YAML("./configs/iris.yml")))
}

```

## Builtin Configurators

```go

// WithGlobalConfiguration will load the global yaml configuration file
// from the home directory and it will set/override the whole app's configuration
// to that file's contents. The global configuration file can be modified by user
// and be used by multiple iris instances.
//
// This is useful when we run multiple iris servers that share the same
// configuration, even with custom values at its "Other" field.
//
// Usage: `app.Configure(iris.WithGlobalConfiguration)` or `app.Run([iris.Runner], iris.WithGlobalConfiguration)`.
WithGlobalConfiguration

// variables for configurators don't need any receivers, functions
// for them that need (helps code editors to recognise as variables without parenthesis completion).

// WithoutServerError will cause to ignore the matched "errors"
// from the main application's `Run` function.
//
// Usage:
// err := app.Listen(":8080", iris.WithoutServerError(iris.ErrServerClosed))
// will return `nil` if the server's error was `http/iris#ErrServerClosed`.
//
// See `Configuration#IgnoreServerErrors []string` too.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/http-listening/listen-addr/omit-server-errors
WithoutServerError(errors ...error) Configurator

// WithoutStartupLog turns off the information send, once, to the terminal when the main server is open.
WithoutStartupLog

// WithoutInterruptHandler disables the automatic graceful server shutdown
// when control/cmd+C pressed.
WithoutInterruptHandle

// WithoutPathCorrection disables the PathCorrection setting.
//
// See `Configuration`.
WithoutPathCorrectio

// WithoutPathCorrectionRedirection disables the PathCorrectionRedirection setting.
//
// See `Configuration`.
WithoutPathCorrectionRedirection

// WithoutBodyConsumptionOnUnmarshal disables BodyConsumptionOnUnmarshal setting.
//
// See `Configuration`.
WithoutBodyConsumptionOnUnmarshal

// WithoutAutoFireStatusCode disables the AutoFireStatusCode setting.
//
// See `Configuration`.
WithoutAutoFireStatusCode

// WithPathEscape enables the PathEscape setting.
//
// See `Configuration`.
WithPathEscape

// WithOptimizations can force the application to optimize for the best performance where is possible.
//
// See `Configuration`.
WithOptimizations

// WithFireMethodNotAllowed enables the FireMethodNotAllowed setting.
//
// See `Configuration`.
WithFireMethodNotAllowed

// WithTimeFormat sets the TimeFormat setting.
//
// See `Configuration`.
WithTimeFormat(timeformat string) Configurator

// WithCharset sets the Charset setting.
//
// See `Configuration`.
WithCharset(charset string) Configurator

// WithPostMaxMemory sets the maximum post data size
// that a client can send to the server, this differs
// from the overral request body size which can be modified
// by the `context#SetMaxRequestBodySize` or `iris#LimitRequestBodySize`.
//
// Defaults to 32MB or 32 << 20 if you prefer.
WithPostMaxMemory(limit int64) Configurator

// WithRemoteAddrHeader enables or adds a new or existing request header name
// that can be used to validate the client's real IP.
//
// By-default no "X-" header is consired safe to be used for retrieving the
// client's IP address, because those headers can manually change by
// the client. But sometimes are useful e.g., when behind a proxy
// you want to enable the "X-Forwarded-For" or when cloudflare
// you want to enable the "CF-Connecting-IP", inneed you
// can allow the `ctx.RemoteAddr()` to use any header
// that the client may sent.
//
// Defaults to an empty map but an example usage is:
// WithRemoteAddrHeader("X-Forwarded-For")
//
// Look `context.RemoteAddr()` for more.
WithRemoteAddrHeader(headerName string) Configurator

// WithoutRemoteAddrHeader disables an existing request header name
// that can be used to validate and parse the client's real IP.
//
//
// Keep note that RemoteAddrHeaders is already defaults to an empty map
// so you don't have to call this Configurator if you didn't
// add allowed headers via configuration or via `WithRemoteAddrHeader` before.
//
// Look `context.RemoteAddr()` for more.
WithoutRemoteAddrHeader(headerName string) Configurator

// WithRemoteAddrPrivateSubnet adds a new private sub-net to be excluded from `context.RemoteAddr`.
// See `WithRemoteAddrHeader` too.
WithRemoteAddrPrivateSubnet(startIP, endIP string) Configurator

// WithOtherValue adds a value based on a key to the Other setting.
//
// See `Configuration.Other`.
WithOtherValue(key string, val interface{}) Configurator

// WithSitemap enables the sitemap generator.
// Use the Route's `SetLastMod`, `SetChangeFreq` and `SetPriority` to modify
// the sitemap's URL child element properties.
//
// It accepts a "startURL" input argument which
// is the prefix for the registered routes that will be included in the sitemap.
//
// If more than 50,000 static routes are registered then sitemaps will be splitted and a sitemap index will be served in
// /sitemap.xml.
//
// If `Application.I18n.Load/LoadAssets` is called then the sitemap will contain translated links for each static route.
//
// If the result does not complete your needs you can take control
// and use the github.com/kataras/sitemap package to generate a customized one instead.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/sitemap.
WithSitemap(startURL string) Configurator

// WithTunneling is the `iris.Configurator` for the `iris.Configuration.Tunneling` field.
// It's used to enable http tunneling for an Iris Application, per registered host
//
// Alternatively use the `iris.WithConfiguration(iris.Configuration{Tunneling: iris.TunnelingConfiguration{ ...}}}`.
WithTunneling
```

## Custom Configurator

With the `Configurator` developers can modularize their applications with ease.

Example Code:

```go
// file counter/counter.go
package counter

import (
    "time"

    "github.com/kataras/iris/v12"
    "github.com/kataras/iris/v12/core/host"
)

func Configurator(app *iris.Application) {
    counterValue := 0

    go func() {
        ticker := time.NewTicker(time.Second)

        for range ticker.C {
            counterValue++
        }

        app.ConfigureHost(func(h *host.Supervisor) { // <- HERE: IMPORTANT
            h.RegisterOnShutdown(func() {
                ticker.Stop()
            })
        })
    }()

    app.Get("/counter", func(ctx iris.Context) {
        ctx.Writef("Counter value = %d", counterValue)
    })
}
```

```go
// file: main.go
package main

import (
    "counter"

    "github.com/kataras/iris/v12"
)

func main() {
    app := iris.New()
    app.Configure(counter.Configurator)

    app.Listen(":8080")
}
```