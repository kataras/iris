// Query strings and form bodies: the two places where client input arrives
// as loose key/value pairs.
//
// ctx.Query() is a read-only, sorted view of the URL query with typed
// accessors and fixed failure defaults. ctx.ReadForm decodes an urlencoded or
// multipart body into a struct through the `form` tag; unlike ctx.BindJSON it
// returns the error instead of answering the client, so a rejected form
// renders through errors.InvalidArgument.
//
// Learn: read typed query values (a UUID too) with safe defaults, tell "absent" from
// "empty", and decode a form into a struct, rejecting a bad one with a 400.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl 'http://localhost:8080/search?q=iris&page=2&tags=go&tags=web'   # 200, the parsed values
//	curl 'http://localhost:8080/search'                                   # 200, page 1 and sort "relevance"
//	curl 'http://localhost:8080/search?q=iris&shelf=11111111-1111-1111-1111-111111111111'   # 200, shelf is parsed
//	curl -X POST http://localhost:8080/contact \
//	  -d 'name=Gerasimos' -d 'email=me@example.com' -d 'message=hello' -d 'subscribe=true'   # 200, echoes the form
//	curl -i -X POST http://localhost:8080/contact -d 'name=Gerasimos'     # 400 envelope, message is required
//
// Next: ../read-json-validate
package main

import (
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	app.Get("/search", search)
	app.Post("/contact", contact)

	app.Listen(":8080")
}

// ContactForm binds through the `form` tag; the same tag covers
// application/x-www-form-urlencoded and multipart/form-data bodies.
// The `json` tags only shape the response this example echoes back.
type ContactForm struct {
	Name      string `form:"name" json:"name"`
	Email     string `form:"email" json:"email"`
	Message   string `form:"message" json:"message"`
	Subscribe bool   `form:"subscribe" json:"subscribe"`
}

func search(ctx iris.Context) {
	// GetString returns "" when the key is absent.
	q := ctx.Query().GetString("q")

	// GetInt returns -1 when the key is absent or not a number, so a default
	// is a plain comparison instead of an error branch.
	page := ctx.Query().GetInt("page")
	if page < 1 {
		page = 1
	}

	// GetSlice returns the raw multi-value list as the client sent it:
	// ?tags=go&tags=web yields ["go", "web"]. GetStringSlice instead splits a
	// single joined value on ContextOptions.URLParamSeparator (default ","),
	// so ?tags=go,web yields the same two elements.
	tags := ctx.Query().GetSlice("tags")

	// GetUUID parses a UUID query value. The zero UUID means the key is
	// absent or the value is not a UUID.
	shelf := ctx.Query().GetUUID("shelf")

	// Lookup distinguishes "absent" from "present but empty", the one thing
	// the safe defaults cannot express.
	sort, sortRequested := ctx.Query().Lookup("sort")
	if !sortRequested {
		sort = "relevance"
	}

	ctx.JSON(iris.Map{
		"query":          q,
		"page":           page,
		"tags":           tags,
		"shelf":          shelf,
		"sort":           sort,
		"sort_requested": sortRequested,
	})
}

func contact(ctx iris.Context) {
	var form ContactForm

	// ReadForm reports a path error when the client sends a field the struct
	// does not have; that class is usually ignored on purpose. An empty body
	// leaves the struct zeroed and returns nil, unless
	// ContextOptions.FireEmptyFormError is set.
	if err := ctx.ReadForm(&form); err != nil && !iris.IsErrPath(err) {
		// The error code carries the 400, so the handler writes no status.
		errors.InvalidArgument.Err(ctx, err)
		return
	}

	if form.Name == "" || form.Message == "" {
		errors.InvalidArgument.Detail(ctx, "invalid form", "name and message are required")
		return
	}

	// Single form values without a struct: ctx.PostForm() reads body values
	// only, ctx.Form() reads body plus query, both with the same typed
	// accessors as ctx.Query().
	source := ctx.Form().GetString("source")

	ctx.JSON(iris.Map{"received": form, "source": source})
}
