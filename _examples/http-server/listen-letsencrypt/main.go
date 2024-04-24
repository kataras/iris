// Package main provide one-line integration with letsencrypt.org
package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("Hello from SECURE SERVER!")
	})

	app.Get("/test2", func(ctx iris.Context) {
		ctx.Writef("Welcome to secure server from /test2!")
	})

	app.Get("/redirect", func(ctx iris.Context) {
		ctx.Redirect("/test2")
	})

	// NOTE: This will not work on domains like this,
	// use real whitelisted domain(or domains split by whitespaces)
	// and a non-public e-mail instead or edit your hosts file.
	app.Run(iris.AutoTLS(":443", "example.com", "mail@example.com"))

	// Note: to disable automatic "http://" to "https://" redirections pass
	// the `iris.AutoTLSNoRedirect` host configurator to AutoTLS function, example:
	/*
		var fallbackServer = func(acme func(http.Handler) http.Handler) *http.Server {
			// Use any http.Server and Handler, as long as it's wrapped by `acme` one.
			// In that case we share the application through non-tls users too:
			srv := &http.Server{Handler: acme(app)}
			go srv.ListenAndServe()
			return srv
		}

		app.Run(iris.AutoTLS(":443", "example.com myip", "mail@example.com",
			iris.AutoTLSNoRedirect(fallbackServer)))
	*/
}
