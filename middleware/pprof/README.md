# Iris Pprof Middleware

## Overview

This middleware mounts the standard library's `net/http/pprof` handlers on an Iris route, so the CPU, heap, goroutine and trace profiles of a running application are reachable over HTTP.

## Features

- **CPU Profiling**: Capture CPU usage over time.
- **Heap Profiling**: Analyze memory allocations.
- **Goroutine Profiling**: Inspect stack traces of all current goroutines.
- **Thread Creation Profiling**: Track the creation of new OS threads.
- **Block Profiling**: Identify blocking on synchronization primitives.
- **Mutex Profiling**: Trace holders of contended mutexes.
- **Execution Tracing**: Record execution trace of the program.
- **Command Line Invocation**: View the command line invocation of the current program.

## Installation

To use the Iris Pprof middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/pprof"
```

## Usage

### Basic Setup

To use the pprof middleware Router, you need to import the necessary packages and configure the middleware in your Iris application. Here is a basic example:

```go
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/pprof"
)

func main() {
	app := iris.New()

	// Register the pprof router.
	app.Router("/debug/pprof", pprof.New())

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<h1> Please click <a href='/debug/pprof'>here</a> to access pprof</h1>")
	})

	app.Listen(":8080")
}
```

> There is `pprof.Handler` which is an `iris.Handler` that you can use to register the pprof routes manually or execute inside an existing route handler.

### Accessing Pprof

Once the middleware is registered, you can access the pprof tools by navigating to `/debug/pprof` in your web browser. The available profiles include:

- `/debug/pprof/allocs`: A sampling of all past memory allocations.
- `/debug/pprof/block`: Stack traces that led to blocking on synchronization primitives.
- `/debug/pprof/cmdline`: The command line invocation of the current program.
- `/debug/pprof/goroutine`: Stack traces of all current goroutines.
- `/debug/pprof/heap`: A sampling of memory allocations of live objects.
- `/debug/pprof/mutex`: Stack traces of holders of contended mutexes.
- `/debug/pprof/profile`: CPU profile. You can specify the duration in the seconds GET parameter.
- `/debug/pprof/threadcreate`: Stack traces that led to the creation of new OS threads.
- `/debug/pprof/trace`: A trace of execution of the current program.

### Protecting Pprof

You can protect the pprof routes with an authentication middleware to prevent unauthorized access.

```go
import (
	"github.com/kataras/iris/v14/middleware/basicauth"
	"github.com/kataras/iris/v14/middleware/pprof"
)
```

```go
auth := basicauth.Default(map[string]string{
	"admin": "admin",
	"gerasimos": "p@ss",
})
app.Router("/debug/pprof", auth, pprof.New())
```

## References

- [Go Pprof Package](https://pkg.go.dev/net/http/pprof)
- [Profiling Go Programs](https://blog.golang.org/profiling-go-programs)
- [Go Execution Tracer](https://golang.org/pkg/runtime/trace/)
