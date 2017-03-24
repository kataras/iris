package main

import (
	"fmt" // just an optional helper
	"io"
	"time" // showcase the delay

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
)

func main() {
	app := iris.New()
	// output startup banner and error logs on os.Stdout
	app.Adapt(iris.DevLogger())
	// set the router, you can choose gorillamux too
	app.Adapt(httprouter.New())

	timeWaitForCloseStream := 4 * time.Second

	app.Get("/", func(ctx *iris.Context) {
		i := 0
		// goroutine in order to no block and just wait,
		// goroutine is OPTIONAL and not a very good option but it depends on the needs
		// Look the streaming_simple_2 for an alternative code style
		// Send the response in chunks and wait for a second between each chunk.
		go ctx.StreamWriter(func(w io.Writer) bool {
			i++
			fmt.Fprintf(w, "this is a message number %d\n", i) // write
			time.Sleep(time.Second)                            // imaginary delay
			if i == 4 {
				return false // close and flush
			}
			return true // continue write
		})

		// when this handler finished the client should be see the stream writer's contents
		// simulate a job here...
		time.Sleep(timeWaitForCloseStream)
	})

	app.Get("/alternative", func(ctx *iris.Context) {
		// Send the response in chunks and wait for a second between each chunk.
		ctx.StreamWriter(func(w io.Writer) bool {
			for i := 1; i <= 4; i++ {
				fmt.Fprintf(w, "this is a message number %d\n", i) // write
				time.Sleep(time.Second)
			}

			// when this handler finished the client should be see the stream writer's contents
			return false // stop and flush the contents
		})
	})

	app.Listen(":8080")
}
