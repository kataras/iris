// Content negotiation: one URL, several representations.
//
// ctx.Negotiation() declares what the server can produce, in priority order.
// ctx.Negotiate then matches the request's Accept header against that list and
// renders the corresponding field of iris.N with the matching writer. When
// nothing matches it sets 406 Not Acceptable and returns ErrContentNotSupported.
//
// Learn: offer several representations of one resource and let the Accept
// header pick, with a 406 when none fits.
// Before: ../writers
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -H 'Accept: application/json' http://localhost:8080/user           # 200, JSON
//	curl -H 'Accept: text/xml'         http://localhost:8080/user           # 200, XML
//	curl -H 'Accept: text/plain'       http://localhost:8080/user           # 200, "user 42: Jane"
//	curl -i -H 'Accept: image/png'     http://localhost:8080/user           # 406
//	curl -H 'Accept: text/xml'         http://localhost:8080/user/builder   # 200, XML from the builder's own values
//
// Next: ../files
package main

import (
	"fmt"

	"github.com/kataras/iris/v14"
)

func main() {
	app := iris.New()

	// The values live in iris.N; the builder only declares the offers.
	app.Get("/user", func(ctx iris.Context) {
		ctx.Negotiation().JSON().XML().Text() // server-side offers, best first.

		err := ctx.Negotiate(iris.N{
			JSON: user,
			XML:  user,
			Text: fmt.Sprintf("user %d: %s", user.ID, user.Name),
		})
		if err != nil {
			// 406 is already set; nothing left to write.
			ctx.Application().Logger().Debugf("not acceptable: %v", err)
		}
	})

	// The builder can carry the content itself, then Negotiate takes nil.
	app.Get("/user/builder", func(ctx iris.Context) {
		ctx.Negotiation().
			JSON(user).
			XML(user).
			Text(fmt.Sprintf("user %d: %s", user.ID, user.Name)).
			Charset("utf-8")

		ctx.Negotiate(nil)
	})

	app.Listen(":8080")
}

// User carries both json and xml tags, one per offered representation.
type User struct {
	ID   int    `json:"id" xml:"id"`
	Name string `json:"name" xml:"name"`
}

var user = User{ID: 42, Name: "Jane"}
