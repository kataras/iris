package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/versioning"
)

func main() {
	app := iris.New()

	examplePerRoute(app)
	examplePerParty(app)

	// Read the README.md before any action.
	app.Run(iris.Addr(":8080"))
}

// How to test:
// Open Postman
// GET: localhost:8080/api/cats
// Headers[1] = Accept-Version: "1" and repeat with
// Headers[1] = Accept-Version: "2.5"
// or even "Accept": "application/json; version=2.5"
func examplePerRoute(app *iris.Application) {
	app.Get("/api/cats", versioning.NewMatcher(versioning.Map{
		"1":                 catsVersionExactly1Handler,
		">= 2, < 3":         catsV2Handler,
		versioning.NotFound: versioning.NotFoundHandler,
	}))
}

// How to test:
// Open Postman
// GET: localhost:8080/api/users
// Headers[1] = Accept-Version: "1.9.9" and repeat with
// Headers[1] = Accept-Version: "2.5"
//
// POST: localhost:8080/api/users/new
// Headers[1] = Accept-Version: "1.8.3"
//
// POST: localhost:8080/api/users
// Headers[1] = Accept-Version: "2"
func examplePerParty(app *iris.Application) {
	usersAPI := app.Party("/api/users")

	// version 1.
	usersAPIV1 := versioning.NewGroup(">= 1, < 2")
	usersAPIV1.Get("/", func(ctx iris.Context) {
		ctx.Writef("v1 resource: /api/users handler")
	})
	usersAPIV1.Post("/new", func(ctx iris.Context) {
		ctx.Writef("v1 resource: /api/users/new post handler")
	})

	// version 2.
	usersAPIV2 := versioning.NewGroup(">= 2, < 3")
	usersAPIV2.Get("/", func(ctx iris.Context) {
		ctx.Writef("v2 resource: /api/users handler")
	})
	usersAPIV2.Post("/", func(ctx iris.Context) {
		ctx.Writef("v2 resource: /api/users post handler")
	})

	versioning.RegisterGroups(usersAPI, versioning.NotFoundHandler, usersAPIV1, usersAPIV2)
}

func catsVersionExactly1Handler(ctx iris.Context) {
	ctx.Writef("v1 exactly resource: /api/cats handler")
}

func catsV2Handler(ctx iris.Context) {
	ctx.Writef("v2 resource: /api/cats handler")
}
