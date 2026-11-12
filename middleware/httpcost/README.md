# Iris HTTP Cost Middleware

This middleware measures the resource usage (time, memory, CPU, and network bytes) of incoming requests or custom jobs, providing a unified “cost” metric. It can help you track performance and resource consumption across your Iris-powered applications.

## Table of Contents
1. Overview  
2. Installation  
3. Usage  
   - 1. Quick Start  
   - 2. Using Compute for Custom Jobs
4. Configuration  
5. Where to Use httpcost.Metric  
6. Tips & Ideas

## Overview

The **httpcost** middleware calculates how “expensive” a request or process is, in terms of:
- Time duration (seconds)
- Memory usage (RAM + Network, in MB)
- CPU time (in seconds)

This allows you, as a developer, to log or bill resource usage in multi-tenant environments, monitor suspiciously long requests, or simply analyze performance bottlenecks in your application.

## Installation

```bash
go get github.com/kataras/iris/v14@latest
```

## Usage

### 1. Quick Start

Register the HTTP cost middleware to your Iris app:

```go
import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/context"

    "github.com/kataras/iris/v14/middleware/httpcost"
)

func main() {
    app := iris.New()

    // Get the final metric.
    app.Use(func(ctx iris.Context) {
        // Fire the next handler(s) first,
        // in this case the httpcost middleware itself and the main handler.
        ctx.Next()
        // Get the cost metric for this request.
        metric, ok := httpcost.Get(ctx) 
        if ok {
            // store the metric somewhere or log it...
        }
    })

    // Use the default cost options for measuring each incoming request.
    app.Use(httpcost.New(httpcost.DefaultOptions))

    // Handlers that do some work...
    app.Get("/", func(ctx *context.Context) {
        // Doing some slow or resource-consuming operation...
        ctx.WriteString("Hello World")
    })

    // Start the server.
    app.Listen(":8080")
}
```

> `httpcost.Get(ctx)` provides the final `Metric`, containing time, memory, CPU usage, and cost data.

Now each request’s cost is measured and stored in the request context. You can retrieve it in subsequent handlers to log or display the results.

### 2. Using Compute for Custom Jobs

To measure the cost of any custom Go function:

```go
import "github.com/kataras/iris/v14/middleware/httpcost"

// Suppose you have a custom job to benchmark:
func myLongRunningJob() {
    // ...
}

func measureMyJob() {
    totalCost := httpcost.Compute(myLongRunningJob)
    println("Cost of my job:", totalCost)
}
```

Under the hood, `Compute` uses an internal `Computer` which starts tracking resource usage, runs your function, then stops and returns the total cost.

## Configuration

The `Options` struct defines how costs are calculated:

```go
type Options struct {
    CostPerSecond float64 // e.g. 0.01
    CostPerMB     float64 // e.g. 0.001
    CostPerCPU    float64 // e.g. 0.02
}
```

Any of these set to `0` disable that resource dimension for the calculation. You can create your own `Options` or use `DefaultOptions`:

```go
opts := httpcost.Options{
    CostPerSecond: 0.02, // charge more per second
    CostPerMB:     0.0,  // ignore memory usage
    CostPerCPU:    0.02,
}
app.Use(httpcost.New(opts))
```

## Where to Use httpcost.Metric

The `httpcost.Metric` struct is especially useful in:

1. **Analytics Dashboards**: Store the metric in a database to visualize how resources are spent over time or per endpoint.
2. **Billing**: If you host user-generated code or third-party logic, you can use `Metric.Cost` as the basis for usage-based billing.
3. **Performance Monitoring**: Track CPU or memory outliers to identify poorly optimized endpoints.
4. **Resource Limits**: Automatically detect large usage, e.g. suspicious or repeated resource-heavy requests, and take actions (block, throttle, or analyze further).

## Tips & Ideas

- **Combine with Access Logs**: Use “httpcost.Metric” alongside the [AccessLog middleware](https://github.com/kataras/iris/tree/main/middleware/accesslog) to produce more detailed performance logs.  
- **Alerting**: If the cost passes a certain threshold, send an alert to your monitoring system.  
- **Per-request Pricing**: Combine the cost with user-specific factors to compute partial usage-based pricing.  
- **Limit High Resource Use**: If you detect unusual spikes in CPU or memory usage, consider adding concurrency limits or queueing strategies to protect critical workloads.

Happy coding with Iris and the **httpcost** middleware!
