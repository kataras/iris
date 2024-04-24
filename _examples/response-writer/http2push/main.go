// Server push lets the server preemptively "push" website assets
// to the client without the user having explicitly asked for them.
// When used with care, we can send what we know the user is going
// to need for the page they're requesting.
package main

import (
	"net/http"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()
	app.Get("/", pushHandler)
	app.Get("/main.js", simpleAssetHandler)

	app.Run(iris.TLS("127.0.0.1:443", "mycert.crt", "mykey.key"))
	// $ openssl req -new -newkey rsa:4096 -x509 -sha256 \
	// -days 365 -nodes -out mycert.crt -keyout mykey.key
}

func pushHandler(ctx iris.Context) {
	// The target must either be an absolute path (like "/path") or an absolute
	// URL that contains a valid host and the same scheme as the parent request.
	// If the target is a path, it will inherit the scheme and host of the
	// parent request.
	target := "/main.js"

	if pusher, ok := ctx.ResponseWriter().Naive().(http.Pusher); ok {
		err := pusher.Push(target, nil)
		if err != nil {
			if err == iris.ErrPushNotSupported {
				ctx.StopWithText(iris.StatusHTTPVersionNotSupported, "HTTP/2 push not supported.")
			} else {
				ctx.StopWithError(iris.StatusInternalServerError, err)
			}
			return
		}
	}

	ctx.HTML(`<html><body><script src="%s"></script></body></html>`, target)
}

func simpleAssetHandler(ctx iris.Context) {
	ctx.ServeFile("./public/main.js")
}
