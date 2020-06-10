package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		// GetReferrer extracts and returns the information from the "Referer" (or "Referrer") header
		// and url query parameter as specified in https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy.
		r := ctx.GetReferrer()
		switch r.Type {
		case iris.ReferrerSearch:
			ctx.Writef("Search %s: %s\n", r.Label, r.Query)
			ctx.Writef("Google: %s\n", r.GoogleType)
		case iris.ReferrerSocial:
			ctx.Writef("Social %s\n", r.Label)
		case iris.ReferrerIndirect:
			ctx.Writef("Indirect: %s\n", r.URL)
		}
	})

	// http://localhost:8080?referrer=https://twitter.com/Xinterio/status/1023566830974251008
	// http://localhost:8080?referrer=https://www.google.com/search?q=Top+6+golang+web+frameworks&oq=Top+6+golang+web+frameworks
	app.Listen(":8080")
}
