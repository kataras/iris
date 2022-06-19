package main

import (
	"context"
	"net/http"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	httpThirdPartyWrapper := StandardWrapper(Options{
		Message: "test_value",
	})

	// This case
	app.WrapRouter(func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
		httpThirdPartyWrapper(router).ServeHTTP(w, r)
		// If was func(http.HandlerFunc) http.HandlerFunc:
		// httpThirdPartyWrapper(router.ServeHTTP).ServeHTTP(w, r)
	})

	app.Get("/", index)
	app.Listen(":8080")
}

func index(ctx iris.Context) {
	ctx.Writef("Message: %s\n", ctx.Value(msgContextKey))
}

type Options struct {
	Message string
}

type contextKey uint8

var (
	msgContextKey contextKey = 1
)

func StandardWrapper(opts Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// ...
			req := r.WithContext(context.WithValue(r.Context(), msgContextKey, opts.Message))
			next.ServeHTTP(w, req)
		})
	}
}
