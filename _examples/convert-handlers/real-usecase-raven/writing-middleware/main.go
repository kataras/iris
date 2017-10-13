package main

import (
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/kataras/iris"

	"github.com/getsentry/raven-go"
)

// At this example you will see how to convert any net/http middleware
// that has the form of `(HandlerFunc) HandlerFunc`.
// If the `raven.RecoveryHandler` had the form of
// `(http.HandlerFunc)` or `(http.HandlerFunc, next http.HandlerFunc)`
// you could just use the `irisMiddleware := iris.FromStd(nativeHandler)`
// but it doesn't, however as you already know Iris can work with net/http directly
// because of the `ctx.ResponseWriter()` and `ctx.Request()` are the original
// http.ResponseWriter and *http.Request.
// (this one is a big advantage, as a result you can use Iris for ever :)).
//
// The source code of the native middleware does not change at all.
// https://github.com/getsentry/raven-go/blob/379f8d0a68ca237cf8893a1cdfd4f574125e2c51/http.go#L70
// The only addition is the Line 18 and Line 39 (instead of handler(w,r))
// and you have a new iris middleware ready to use!
func irisRavenMiddleware(ctx iris.Context) {
	w, r := ctx.ResponseWriter(), ctx.Request()

	defer func() {
		if rval := recover(); rval != nil {
			debug.PrintStack()
			rvalStr := fmt.Sprint(rval)
			packet := raven.NewPacket(rvalStr, raven.NewException(errors.New(rvalStr), raven.NewStacktrace(2, 3, nil)), raven.NewHttp(r))
			raven.Capture(packet, nil)
			w.WriteHeader(iris.StatusInternalServerError)
		}
	}()

	ctx.Next()
}

// https://docs.sentry.io/clients/go/integrations/http/
func init() {
	raven.SetDSN("https://<key>:<secret>@sentry.io/<project>")
}

func main() {
	app := iris.New()
	app.Use(irisRavenMiddleware)

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("Hi")
	})

	app.Run(iris.Addr(":8080"))
}
