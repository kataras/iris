# Hosts

## Listen and Serve

You can start the server(s) listening to any type of `net.Listener` or even `http.Server` instance.
The method for initialization of the server should be passed at the end, via `Run` function.

The most common method that Go developers use to serve their servers are
by passing a network address with form of "hostname:ip". With Iris
we use the `iris.Addr` which is an `iris.Runner` type

```go
// Listening on tcp with network address 0.0.0.0:8080
app.Run(iris.Addr(":8080"))
```

Sometimes you have created a standard net/http server somewhere else in your app and want to use that to serve the Iris web app

```go
// Same as before but using a custom http.Server which may being used somewhere else too
app.Run(iris.Server(&http.Server{Addr:":8080"}))
```

The most advanced usage is to create a custom or a standard `net.Listener` and pass that to `app.Run`

```go
// Using a custom net.Listener
l, err := net.Listen("tcp4", ":8080")
if err != nil {
    panic(err)
}
app.Run(iris.Listener(l))
```

A more complete example, using the unix-only socket files feature 

```go
package main

import (
    "os"
    "net"

    "github.com/kataras/iris"
)

func main() {
    app := iris.New()

    // UNIX socket
    if errOs := os.Remove(socketFile); errOs != nil && !os.IsNotExist(errOs) {
        app.Logger().Fatal(errOs)
    }

    l, err := net.Listen("unix", socketFile)

    if err != nil {
        app.Logger().Fatal(err)
    }

    if err = os.Chmod(socketFile, mode); err != nil {
        app.Logger().Fatal(err)
    }

    app.Run(iris.Listener(l))
}
```

UNIX and BSD hosts can take advantage of the reuse port feature

```go
package main

import (
    // Package tcplisten provides customizable TCP net.Listener with various
    // performance-related options:
    //
    //   - SO_REUSEPORT. This option allows linear scaling server performance
    //     on multi-CPU servers.
    //     See https://www.nginx.com/blog/socket-sharding-nginx-release-1-9-1/ for details.
    //
    //   - TCP_DEFER_ACCEPT. This option expects the server reads from the accepted
    //     connection before writing to them.
    //
    //   - TCP_FASTOPEN. See https://lwn.net/Articles/508865/ for details.
    "github.com/valyala/tcplisten"

    "github.com/kataras/iris"
)

// go get github.com/valyala/tcplisten
// go run main.go

func main() {
    app := iris.New()

    app.Get("/", func(ctx iris.Context) {
        ctx.HTML("<h1>Hello World!</h1>")
    })

    listenerCfg := tcplisten.Config{
        ReusePort:   true,
        DeferAccept: true,
        FastOpen:    true,
    }

    l, err := listenerCfg.NewListener("tcp", ":8080")
    if err != nil {
        app.Logger().Fatal(err)
    }

    app.Run(iris.Listener(l))
}
```

### HTTP/2 and Secure

If you have signed file keys you can use the `iris.TLS` to serve `https` based on those certification keys

```go
// TLS using files
app.Run(iris.TLS("127.0.0.1:443", "mycert.cert", "mykey.key"))
```

The method you should use when your app is ready for **production** is the `iris.AutoTLS` which starts a secure server with automated certifications provided by https://letsencrypt.org for **free**

```go
// Automatic TLS
app.Run(iris.AutoTLS(":443", "example.com", "admin@example.com"))
```

### Any `iris.Runner`

There may be times that you want something very special to listen on, which is not a type of `net.Listener`. You are able to do that by `iris.Raw`, but you're responsible of that method

```go
// Using any func() error,
// the responsibility of starting up a listener is up to you with this way,
// for the sake of simplicity we will use the
// ListenAndServe function of the `net/http` package.
app.Run(iris.Raw(&http.Server{Addr:":8080"}).ListenAndServe)
```

## Host configurators

All the above forms of listening are accepting a last, variadic argument of `func(*iris.Supervisor)`. This is used to add configurators for that specific host you passed via those functions.

For example let's say that we want to add a callback which is fired when
the server is shutdown

```go
app.Run(iris.Addr(":8080", func(h *iris.Supervisor) {
    h.RegisterOnShutdown(func() {
        println("server terminated")
    })
}))
```

You can even do that before `app.Run` method, but the difference is that
these host configurators will be executed to all hosts that you may use to serve your web app (via `app.NewHost` we'll see that in a minute)

```go
app := iris.New()
app.ConfigureHost(func(h *iris.Supervisor) {
    h.RegisterOnShutdown(func() {
        println("server terminated")
    })
})
app.Run(iris.Addr(":8080"))
```

Access to all hosts that serve your application can be provided by
the `Application#Hosts` field, after the `Run` method.

But the most common scenario is that you may need access to the host before the `app.Run` method,
there are two ways of gain access to the host supervisor, read below.

We have already saw how to configure all application's hosts by second argument of `app.Run` or `app.ConfigureHost`. There is one more way which suits better for simple scenarios and that is to use the `app.NewHost` to create a new host
and use one of its `Serve` or `Listen` functions
to start the application via the `iris#Raw` Runner.

Note that this way needs an extra import of the `net/http` package.

Example Code:

```go
h := app.NewHost(&http.Server{Addr:":8080"})
h.RegisterOnShutdown(func(){
    println("server terminated")
})

app.Run(iris.Raw(h.ListenAndServe))
```

## Multi hosts

You can serve your Iris web app using more than one server, the `iris.Router` is compatible with the `net/http/Handler` function therefore, as you can understand, it can be used to be adapted at any `net/http` server, however there is an easier way, by using the `app.NewHost` which is also copying all the host configurators and it closes all the hosts attached to the particular web app on `app.Shutdown`.

```go
app := iris.New()
app.Get("/", indexHandler)

// run in different goroutine in order to not block the main "goroutine".
go app.Run(iris.Addr(":8080"))
// start a second server which is listening on tcp 0.0.0.0:9090,
// without "go" keyword because we want to block at the last server-run.
app.NewHost(&http.Server{Addr:":9090"}).ListenAndServe()
```

## Shutdown (Gracefully)

Let's continue by learning how to catch CONTROL+C/COMMAND+C or unix kill command and shutdown the server gracefully.

> Gracefully Shutdown on CONTROL+C/COMMAND+C or when kill command sent is ENABLED BY-DEFAULT.

In order to manually manage what to do when app is interrupted,
we have to disable the default behavior with the option `WithoutInterruptHandler`
and register a new interrupt handler (globally, across all possible hosts).


Example code:

```go
package main

import (
    "context"
    "time"

    "github.com/kataras/iris"
)


func main() {
    app := iris.New()

    iris.RegisterOnInterrupt(func() {
        timeout := 5 * time.Second
        ctx, cancel := context.WithTimeout(context.Background(), timeout)
        defer cancel()
        // close all hosts
        app.Shutdown(ctx)
    })

    app.Get("/", func(ctx iris.Context) {
        ctx.HTML(" <h1>hi, I just exist in order to see if the server is closed</h1>")
    })

    app.Run(iris.Addr(":8080"), iris.WithoutInterruptHandler)
}
```
