package main

import (
	"github.com/valyala/fasthttp"
	_"time"
	"log"
	"bytes"
)

var thepath = []byte("/rest/hello")
var themethod =[]byte("GET")

func myHandler(ctx *fasthttp.RequestCtx) {
	if bytes.Equal(ctx.Method(),themethod) &&  bytes.Equal(ctx.Path(),thepath) {
		//time.Sleep(time.Duration(500) * time.Millisecond)
		ctx.SetBodyString("Hello world")
	}

}

func main() {
	log.Fatal(fasthttp.ListenAndServe(":8080", myHandler))
}
