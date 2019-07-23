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
    "github.com/kataras/iris"
)

func main() {
    app := iris.New()
    app.Get("/", func(ctx iris.Context) {
        ctx.HTML("<b>Hello!</b>")
    })
    // [...]

    // Good when you want to modify the whole configuration.
    app.Run(iris.Addr(":8080"), iris.WithConfiguration(iris.Configuration{
        DisableStartupLog:                 false,
        DisableInterruptHandler:           false,
        DisablePathCorrection:             false,
        EnablePathEscape:                  false,
        FireMethodNotAllowed:              false,
        DisableBodyConsumptionOnUnmarshal: false,
        DisableAutoFireStatusCode:         false,
        TimeFormat:                        "Mon, 02 Jan 2006 15:04:05 GMT",
        Charset:                           "UTF-8",
    }))
}
```

`.Run` **by options**

```go
package main

import (
    "github.com/kataras/iris"
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

    app.Run(iris.Addr(":8080"), iris.WithoutStartupLog, iris.WithCharset("UTF-8"))

    // or before run:
    // app.Configure(iris.WithoutStartupLog, iris.WithCharset("UTF-8"))
    // app.Run(iris.Addr(":8080"))
}
```

`.Run` **by TOML config file**

```tml
DisablePathCorrection = false
EnablePathEscape = false
FireMethodNotAllowed = true
DisableBodyConsumptionOnUnmarshal = false
TimeFormat = "Mon, 01 Jan 2006 15:04:05 GMT"
Charset = "UTF-8"

[Other]
	MyServerName = "iris"

```

```go
package main

import (
    "github.com/kataras/iris"
)

func main() {
    app := iris.New()

    app.Get("/", func(ctx iris.Context) {
        ctx.HTML("<b>Hello!</b>")
    })
    // [...]

    // Good when you have two configurations, one for development and a different one for production use.
    app.Run(iris.Addr(":8080"), iris.WithConfiguration(iris.TOML("./configs/iris.tml")))
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
    "github.com/kataras/iris"
)

func main() {
    app := iris.New()
    app.Get("/", func(ctx iris.Context) {
        ctx.HTML("<b>Hello!</b>")
    })
    // [...]

    app.Run(iris.Addr(":8080"), iris.WithConfiguration(iris.YAML("./configs/iris.yml")))
}

```

## Builtin Configurators

```go
// WithoutServerError will cause to ignore the matched "errors"
// from the main application's `Run` function.
//
// Usage:
// err := app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
// will return `nil` if the server's error was `http/iris#ErrServerClosed`.
//
// See `Configuration#IgnoreServerErrors []string` too.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/http-listening/listen-addr/omit-server-errors
func WithoutServerError(errors ...error) Configurator

// WithoutStartupLog turns off the information send, once, to the terminal when the main server is open.
var WithoutStartupLog

// WithoutInterruptHandler disables the automatic graceful server shutdown
// when control/cmd+C pressed.
var WithoutInterruptHandler

// WithoutPathCorrection disables the PathCorrection setting.
//
// See `Configuration`.
var WithoutPathCorrection

// WithoutBodyConsumptionOnUnmarshal disables BodyConsumptionOnUnmarshal setting.
//
// See `Configuration`.
var WithoutBodyConsumptionOnUnmarshal

// WithoutAutoFireStatusCode disables the AutoFireStatusCode setting.
//
// See `Configuration`.
var WithoutAutoFireStatusCode

// WithPathEscape enanbles the PathEscape setting.
//
// See `Configuration`.
var WithPathEscape

// WithOptimizations can force the application to optimize for the best performance where is possible.
//
// See `Configuration`.
var WithOptimizations

// WithFireMethodNotAllowed enanbles the FireMethodNotAllowed setting.
//
// See `Configuration`.
var WithFireMethodNotAllowed

// WithTimeFormat sets the TimeFormat setting.
//
// See `Configuration`.
func WithTimeFormat(timeformat string) Configurator

// WithCharset sets the Charset setting.
//
// See `Configuration`.
func WithCharset(charset string) Configurator

// WithRemoteAddrHeader enables or adds a new or existing request header name
// that can be used to validate the client's real IP.
//
// Existing values are:
// "X-Real-Ip":             false,
// "X-Forwarded-For":       false,
// "CF-Connecting-IP": false
//
// Look `context.RemoteAddr()` for more.
func WithRemoteAddrHeader(headerName string) Configurator

// WithoutRemoteAddrHeader disables an existing request header name
// that can be used to validate the client's real IP.
//
// Existing values are:
// "X-Real-Ip":             false,
// "X-Forwarded-For":       false,
// "CF-Connecting-IP": false
//
// Look `context.RemoteAddr()` for more.
func WithoutRemoteAddrHeader(headerName string) Configurator

// WithOtherValue adds a value based on a key to the Other setting.
//
// See `Configuration`.
func WithOtherValue(key string, val interface{}) Configurator
```

## Custom Configurator

With the `Configurator` developers can modularize their applications with ease.

Example Code:

```go
// file counter/counter.go
package counter

import (
    "time"

    "github.com/kataras/iris"
    "github.com/kataras/iris/core/host"
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

    "github.com/kataras/iris"
)

func main() {
    app := iris.New()
    app.Configure(counter.Configurator)

    app.Run(iris.Addr(":8080"))
}
```