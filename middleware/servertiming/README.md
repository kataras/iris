# Server-Timing Middleware for Iris

The `Server-Timing` response header reports how long the server spent on each part of a request, and browser devtools display those numbers next to the network timings. This middleware records the metrics and writes the header. The header itself is specified in the [Server Timing RFC](https://www.w3.org/TR/server-timing/#the-server-timing-header-field).

### What is Server-Timing?

The `Server-Timing` header provides a mechanism for the server to communicate performance metrics about the request-response cycle to the client. This header can include multiple metrics, each with a name, description, and duration. These metrics can help developers understand where time is being spent on the server and identify potential bottlenecks.

#### Example of a Server-Timing Header

```
Server-Timing: db;desc="Database";dur=53.2, app;desc="Application";dur=47.8
```

In this example, two metrics are provided:
- `db`: Describes the time spent on database operations, with a duration of 53.2 milliseconds.
- `app`: Describes the time spent on application processing, with a duration of 47.8 milliseconds.

### Middleware Overview

The package writes `Server-Timing` headers for you. `Handler` measures the total processing time of every request on its own, and `Start`/`End` time whichever individual operations you care about.

### Installation

To use this middleware, you need to import it into your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/servertiming"
```

### Usage

#### Middleware Handler

The `Handler` function is a middleware that automatically measures the total processing time of each request and adds it to the `Server-Timing` header.

##### Example

```go
app := iris.New()
app.Use(servertiming.Handler)

app.Get("/", func(ctx iris.Context) {
    time.Sleep(100 * time.Millisecond) // Simulate some work
    ctx.WriteString("Hello, World!")
})

app.Listen(":8080")
```

In this example, the `Handler` middleware is used to measure the total processing time of the request and add it to the `Server-Timing` header.

#### Start and End Functions

Use `Start` and `End` to time one specific operation inside a handler.

##### Example

```go
app := iris.New()

app.Get("/", func(ctx iris.Context) {
    servertiming.Start(ctx, "db", "Database processing")
    time.Sleep(50 * time.Millisecond) // Simulate database work
    servertiming.End(ctx, "db")

    servertiming.Start(ctx, "cache", "Cache processing")
    time.Sleep(30 * time.Millisecond) // Simulate cache work
    servertiming.End(ctx, "cache")

    ctx.WriteString("Hello, World!")
})

app.Listen(":8080")
```

In this example, the `Start` and `End` functions are used to measure the time spent on database and cache operations. These metrics are then added to the `Server-Timing` header.

#### WriteMetrics Function

`WriteMetrics` writes the `Server-Timing` header yourself, at the exact point in the handler where you want it flushed.

##### Example

```go
app := iris.New()

app.Get("/", func(ctx iris.Context) {
    // Manually create and calculate metrics
    startDB := time.Now()
    time.Sleep(50 * time.Millisecond) // Simulate database work
    elapsedDB := time.Since(startDB)

    startCache := time.Now()
    time.Sleep(30 * time.Millisecond) // Simulate cache work
    elapsedCache := time.Since(startCache)

    metrics := []*servertiming.Metric{
        {
            Name:        "db",
            Description: "Database processing",
            Duration:    elapsedDB,
        },
        {
            Name:        "cache",
            Description: "Cache processing",
            Duration:    elapsedCache,
        },
        {
            Name:        "custom",
            Description: "Custom metric",
            Duration:    20 * time.Millisecond,
        },
    }
    servertiming.WriteMetrics(ctx, metrics)

    ctx.WriteString("Hello, World!")
})

app.Listen(":8080")
```

In this example, the elapsed time for the database and cache operations is manually calculated inside the request handler. The `WriteMetrics` function is then used to add these metrics to the `Server-Timing` header.
