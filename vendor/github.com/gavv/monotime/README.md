# monotime [![GoDoc](https://godoc.org/github.com/gavv/monotime?status.svg)](https://godoc.org/github.com/gavv/monotime) [![Travis](https://img.shields.io/travis/gavv/monotime.svg)](https://travis-ci.org/gavv/monotime)

This tiny Go package is a standalone and slightly enhanced version of [`goarista/monotime`](https://github.com/aristanetworks/goarista#monotime).

It provides `monotime.Now()` function, which returns current time from monotonic clock source. It's implemented using unexported `runtime.nanotime()` function from Go runtime. It works on all platforms.

## Why?

`time.Now()` function from standard library returns *real time* (`CLOCK_REALTIME` in POSIX) which can jump forwards and backwards as the system time is changed.

For time measurements, *monotonic time* (`CLOCK_MONOTONIC` or `CLOCK_MONOTONIC_RAW` on Linux) is often preferred, which is strictly increasing, without (notable) jumps.

## Documentation

See [GoDoc](https://godoc.org/github.com/gavv/monotime).

## Usage example

```go
package main

import (
    "fmt"
    "time"

    "github.com/gavv/monotime"
)

func main() {
    var start, elapsed time.Duration

    start = monotime.Now()
    time.Sleep(time.Millisecond)
    elapsed = monotime.Since(start)

    fmt.Println(elapsed)
    // Prints: 1.062759ms
}
```

## Similar packages

* [`aristanetworks/goarista/monotime`](https://github.com/aristanetworks/goarista#monotime) (this package is based on it)
* [`spacemonkeygo/monotime`](https://github.com/spacemonkeygo/monotime) (current `runtime.nanotime()` is more complete)
* [`davecheney/junk/clock`](https://github.com/davecheney/junk/tree/master/clock) (Linux-only)
* [`jaracil/clk`](https://github.com/jaracil/clk) (Linux-only)

## License

[Apache 2.0](https://github.com/gavv/monotime/blob/master/LICENSE)
