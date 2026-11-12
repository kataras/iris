# Monitor Middleware

Mount this middleware on two endpoints and it reports what the process and the machine are doing right now: CPU and memory of the process, CPU, RAM and load average of the host, open connections and uptime. One endpoint answers JSON, the other serves a small self-refreshing HTML dashboard that reads it.

The metrics are also published through the standard `expvar` package (`pid_cpu`, `pid_ram`, `pid_conns`, `os_cpu`, `os_ram`, `os_total_ram`, `os_load_avg`, `os_conns`), so any collector that already reads `/debug/vars` picks them up.

## Installation

To use the `monitor` middleware, you need to import it in your Iris application:

```go
import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/monitor"
)
```

## Usage

`monitor.New(monitor.Options)` returns a `*monitor.Monitor` rather than a single handler, because it exposes **two** of them and owns the background sampler that both read:

- `m.Stats` writes the statistics as JSON.
- `m.View` renders the HTML dashboard, which polls `Stats`.

Call `m.Stop()` when the monitor is no longer needed, to end the sampler.

```go
package main

import (
    "os"

    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/basicauth"
    "github.com/kataras/iris/v14/middleware/monitor"
)

func main() {
    app := iris.New()

    m := monitor.New(monitor.Options{
        ViewTitle: "My Application",
    })
    defer m.Stop()

    // SECURITY: the stats endpoint must NOT be public, see below.
    auth := basicauth.Default(map[string]string{
        "admin": os.Getenv("MONITOR_PASSWORD"),
    })

    app.Get("/monitor", auth, m.View)
    app.Get("/monitor/stats", auth, m.Stats)

    app.Listen(":8080")
}
```

## Options

The zero value is valid; every field below is optional.

- `PID`: the process to watch. Defaults to the current one.
- `RefreshInterval`: how often the statistics are sampled. Defaults to 2 seconds.
- `ViewRefreshInterval`: how often the dashboard re-fetches them. Defaults to `RefreshInterval`.
- `ViewAnimationInterval`: when greater than zero, animates the dashboard's lines. Defaults to zero, no animation.
- `ViewTitle`: the title of the dashboard document. Defaults to empty.

## Security

**The stats endpoints must not be public.** They report the process id, resident memory, host CPU and RAM, load average and connection counts: a precise picture of the machine, useful to anyone deciding whether and when to attack it. Guard them exactly as you would `pprof`, with authentication, a private listener or an internal-only route, as the example above does.

Note also that importing this package registers its metrics into the default `expvar` map at init, so they are visible through any mounted `/debug/vars` handler too.
