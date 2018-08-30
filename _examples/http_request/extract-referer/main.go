package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx context.Context) /* or iris.Context, it's the same for Go 1.9+. */ {
		// GetReferrer extracts and returns the information from the "Referer" header as specified
		// in https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy or by the URL query parameter "referer".
		r := ctx.GetReferrer()
		switch r.Type {
		case context.ReferrerSearch:
			ctx.Writef("Search %s: %s\n", r.Label, r.Query)
			ctx.Writef("Google: %s\n", r.GoogleType)
		case context.ReferrerSocial:
			ctx.Writef("Social %s\n", r.Label)
		case context.ReferrerIndirect:
			ctx.Writef("Indirect: %s\n", r.URL)
		}
	})

	// http://localhost:8080?referer=https://twitter.com/Xinterio/status/1023566830974251008
	// http://localhost:8080?referer=https://www.google.com/search?q=Top+6+golang+web+frameworks&oq=Top+6+golang+web+frameworks
	app.Run(iris.Addr(":8080"), iris.WithoutServerError(iris.ErrServerClosed))
}
