package fasthttp_test

import (
	"bufio"
	"fmt"
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

func ExampleRequestCtx_SetBodyStreamWriter() {
	// Start fasthttp server for streaming responses.
	if err := fasthttp.ListenAndServe(":8080", responseStreamHandler); err != nil {
		log.Fatalf("unexpected error in server: %s", err)
	}
}

func responseStreamHandler(ctx *fasthttp.RequestCtx) {
	// Send the response in chunks and wait for a second between each chunk.
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		for i := 0; i < 10; i++ {
			fmt.Fprintf(w, "this is a message number %d", i)

			// Do not forget flushing streamed data to the client.
			if err := w.Flush(); err != nil {
				return
			}
			time.Sleep(time.Second)
		}
	})
}
