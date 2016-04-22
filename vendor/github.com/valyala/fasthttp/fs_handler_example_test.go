package fasthttp_test

import (
	"bytes"
	"log"

	"github.com/valyala/fasthttp"
)

// Setup file handlers (aka 'file server config')
var (
	// Handler for serving images from /img/ path,
	// i.e. /img/foo/bar.jpg will be served from
	// /var/www/images/foo/bar.jpb .
	imgPrefix  = []byte("/img/")
	imgHandler = fasthttp.FSHandler("/var/www/images", 1)

	// Handler for serving css from /static/css/ path,
	// i.e. /static/css/foo/bar.css will be served from
	// /home/dev/css/foo/bar.css .
	cssPrefix  = []byte("/static/css/")
	cssHandler = fasthttp.FSHandler("/home/dev/css", 2)

	// Handler for serving the rest of requests,
	// i.e. /foo/bar/baz.html will be served from
	// /var/www/files/foo/bar/baz.html .
	filesHandler = fasthttp.FSHandler("/var/www/files", 0)
)

// Main request handler
func requestHandler(ctx *fasthttp.RequestCtx) {
	path := ctx.Path()
	switch {
	case bytes.HasPrefix(path, imgPrefix):
		imgHandler(ctx)
	case bytes.HasPrefix(path, cssPrefix):
		cssHandler(ctx)
	default:
		filesHandler(ctx)
	}
}

func ExampleFSHandler() {
	if err := fasthttp.ListenAndServe(":80", requestHandler); err != nil {
		log.Fatalf("Error in server: %s", err)
	}
}
