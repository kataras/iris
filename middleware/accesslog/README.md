# AccessLog Middleware for Iris

The `accesslog` middleware for the [Iris web framework](https://www.iris-go.com) provides detailed logging for incoming HTTP requests. This middleware is highly configurable and can log various aspects of the request and response, including custom fields.

## Features

- Logs request and response details.
- Supports multiple output formats (JSON, CSV, custom templates).
- Can log to multiple destinations (files, stdout, etc.).
- Asynchronous logging support.
- Customizable log fields.
- Middleware can be conditionally applied.

## Installation

To use the `accesslog` middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/accesslog"
```

## Basic Usage

Here is a basic example of how to use the `accesslog` middleware in an Iris application:

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/accesslog"
)

func makeAccessLog() *accesslog.AccessLog {
    ac := accesslog.File("./access.log")

    ac.Delim = '|'
    ac.TimeFormat = "2006-01-02 15:04:05"
    ac.Async = false
    ac.IP = true
    ac.BytesReceivedBody = true
    ac.BytesSentBody = true
    ac.BytesReceived = false
    ac.BytesSent = false
    ac.RequestBody = true
    ac.ResponseBody = false
    ac.KeepMultiLineError = true
    ac.PanicLog = accesslog.LogHandler

    ac.SetFormatter(&accesslog.JSON{
        Indent:    "  ",
        HumanTime: true,
    })

    return ac
}

func main() {
    ac := makeAccessLog()
    defer ac.Close()

    app := iris.New()
    app.UseRouter(ac.Handler)

    app.Get("/", func(ctx iris.Context) {
        ctx.WriteString("OK")
    })

    app.Listen(":8080")
}
```

## Configuration Options

### Output Destination

You can set the output destination for the logs using the `File` or `New` functions:

```go
ac := accesslog.File("./access.log")
// or
ac := accesslog.New(os.Stdout)
```

### Log Format

The default log format is:

```
Time|Latency|Code|Method|Path|IP|Path Params Query Fields|Bytes Received|Bytes Sent|Request|Response|
```

You can customize the log format using different formatters:

#### JSON Formatter

```go
ac.SetFormatter(&accesslog.JSON{
    Indent:    "  ",
    HumanTime: true,
})
```

#### CSV Formatter

```go
ac.SetFormatter(&accesslog.CSV{})
```

#### Custom Template Formatter

```go
ac.SetFormatter(&accesslog.Template{Text: "{{.Code}}"})
```

### Custom Fields

You can add custom fields to the log entries:

```go
ac.AddFields(func(ctx iris.Context, f *accesslog.Fields) {
    for k, v := range ctx.Request().Header {
        value := strings.Join(v, ", ")
        f.Set("request.header."+k, value)
    }
})
```

### Asynchronous Logging

Enable asynchronous logging to improve performance:

```go
ac.Async = true
```

### Conditional Logging

You can skip logging for specific routes or conditions:

```go
app.UseRouter(accesslog.SkipHandler)
```

### Response Body and Compression

With `ac.ResponseBody = true` the response body is logged, except for a response that carries a `Content-Encoding` other than `identity`: that line has an empty response field, and its status, bytes sent and every other field are logged as usual. The access log is registered outside the compression middleware (the order `iris.NewBuilder` fixes, so that a panic is still logged), which means what it recorded for a compressed response is the compressed bytes. Those are binary in a log file, and a `Redact` function cannot match anything in them, so it is not called for that line.

## Advanced Usage

### Logging to Multiple Destinations

`SetOutput` and `AddOutput` are variadic: pass the destinations one by one and the middleware combines them itself.

```go
f, _ := os.OpenFile("./access.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
ac.SetOutput(os.Stdout, f)
```

Pass them separately rather than pre-combining them with `io.MultiWriter`. The middleware inspects each writer it is given to find the flushers, closers and truncaters among them; a `MultiWriter` is one opaque writer, so `f` above would never be registered and `Close` would leave its descriptor open.

`Close` closes the destinations this package opened for you (`accesslog.File` and `accesslog.FileUnbuffered`) and every writer you passed that is an `io.Closer` (`f` above). It never closes `os.Stdout`, `os.Stderr` or `os.Stdin`: it did not open them, and closing the process's standard output on shutdown would break every other writer in the program.

### Custom Clock

You can set a custom clock for the log timestamps, useful for testing:

```go
ac.Clock = accesslog.TClock(time.Now())
```

### Middleware Integration

Integrate the `accesslog` middleware with other middlewares:

```go
app.UseRouter(ac.Handler)
app.UseRouter(otherMiddleware)
```

## Examples

### Log Requests to a JSON File

```go
ac := accesslog.File("access_log.json")
ac.SetFormatter(&accesslog.JSON{
    Indent:    "  ",
    HumanTime: true,
})
app.UseRouter(ac.Handler)
```

### Custom Fields and Template

Refer to the [custom fields and template example](https://github.com/kataras/iris/tree/main/_examples/06-middleware/accesslog) for more details.

