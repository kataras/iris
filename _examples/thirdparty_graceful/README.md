## Package information

This package is third-party, it is not build'n Iris package, it's working out of the box, no port needed.

More can be found here: 
[https://github.com/tylerb/graceful](https://github.com/tylerb/graceful)


## Description

Graceful is a Go 1.3+ package enabling graceful shutdown of http.Handler servers.


## Behaviour

When Graceful is sent a SIGINT or SIGTERM (possibly from ^C or a kill command), it:

1. Disables keepalive connections.
2. Closes the listening socket, allowing another process to listen on that port immediately.
3. Starts a timer of `timeout` duration to give active requests a chance to finish.
4. When timeout expires, closes all active connections.
5. Closes the `stopChan`, waking up any blocking goroutines.
6. Returns from the function, allowing the server to terminate.

## Notes

If the `timeout` argument to `Run` is 0, the server never times out, allowing all active requests to complete.

If you wish to stop the server in some way other than an OS signal, you may call the `Stop()` function.
This function stops the server, gracefully, using the new timeout value you provide. The `StopChan()` function
returns a channel on which you can block while waiting for the server to stop. This channel will be closed when
the server is stopped, allowing your execution to proceed. Multiple goroutines can block on this channel at the
same time and all will be signalled when stopping is complete.

## Usage

```go

package main

import (
	"github.com/kataras/iris"
	"gopkg.in/tylerb/graceful.v1"
	"time"
)

func main() {
	api := iris.New()
	api.Get("/home", func(c *iris.Context) {
		c.Write("Hello from the /home")
	})

	graceful.Run(":8080", 10*time.Second, api)
}


```