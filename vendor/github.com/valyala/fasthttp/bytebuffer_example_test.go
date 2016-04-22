package fasthttp_test

import (
	"fmt"

	"github.com/valyala/fasthttp"
)

func ExampleByteBuffer() {
	// This request handler sets 'Your-IP' response header
	// to 'Your IP is <ip>'. It uses ByteBuffer for constructing response
	// header value with zero memory allocations.
	yourIPRequestHandler := func(ctx *fasthttp.RequestCtx) {
		b := fasthttp.AcquireByteBuffer()
		b.B = append(b.B, "Your IP is <"...)
		b.B = fasthttp.AppendIPv4(b.B, ctx.RemoteIP())
		b.B = append(b.B, ">"...)
		ctx.Response.Header.SetBytesV("Your-IP", b.B)

		fmt.Fprintf(ctx, "Check response headers - they must contain 'Your-IP: %s'", b.B)

		// It is safe to release byte buffer now, since it is
		// no longer used.
		fasthttp.ReleaseByteBuffer(b)
	}

	// Start fasthttp server returning your ip in response headers.
	fasthttp.ListenAndServe(":8080", yourIPRequestHandler)
}
