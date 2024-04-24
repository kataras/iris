package main

import (
	stdContext "context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<h1>hi, I just exist in order to see if the server is closed</h1>")
	})

	idleConnsClosed := make(chan struct{})
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
			println("shutdown...")

			timeout := 10 * time.Second
			ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
			defer cancel()
			app.Shutdown(ctx)
			close(idleConnsClosed)
		}
	}()

	// Start the server and disable the default interrupt handler in order to
	// handle it clear and simple by our own, without any issues.
	app.Listen(":8080", iris.WithoutInterruptHandler)
	<-idleConnsClosed
}
