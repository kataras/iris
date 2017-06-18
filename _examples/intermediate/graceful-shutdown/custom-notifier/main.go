package main

import (
	stdContext "context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cdren/iris"
	"github.com/cdren/iris/context"
)

func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout

	app.Get("/", func(ctx context.Context) {
		ctx.HTML(" <h1>hi, I just exist in order to see if the server is closed</h1>")
	})

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch,
			// kill -SIGINT XXXX or Ctrl+c
			os.Interrupt,
			syscall.SIGINT, // register that too, it should be ok
			// os.Kill  is equivalent with the syscall.Kill
			os.Kill,
			syscall.SIGKILL, // register that too, it should be ok
			// kill -SIGTERM XXXX
			syscall.SIGTERM,
		)
		select {
		case <-ch:
			println("Shutdown the server gracefully...")

			timeout := 5 * time.Second // give the server 5 seconds to wait for idle connections.
			ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
			defer cancel()
			app.Shutdown(ctx)
		}
	}()

	// Start the server and disable the default interrupt handler in order to handle it clear and simple by our own, without
	// any issues.
	app.Run(iris.Addr(":8080"), iris.WithoutInterruptHandler)
}
