package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/gorillamux" // import the gorillamux adaptor
)

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger()) // writes both prod and dev logs to the os.Stdout
	app.Adapt(gorillamux.New()) // uses the gorillamux for routing and reverse routing

	// set a custom 404 handler
	app.OnError(iris.StatusNotFound, func(ctx *iris.Context) {
		ctx.HTML(iris.StatusNotFound, "<h1> custom http error page </h1>")
	})

	app.Get("/healthcheck", h)

	gamesMiddleware := func(ctx *iris.Context) {
		println(ctx.Method() + ": " + ctx.Path())
		ctx.Next()
	}

	games := app.Party("/games", gamesMiddleware)
	{ // braces are optional of course, it's just a style of code
		games.Get("/{gameID:[0-9]+}/clans", h)
		games.Get("/{gameID:[0-9]+}/clans/clan/{publicID:[0-9]+}", h)
		games.Get("/{gameID:[0-9]+}/clans/search", h)

		games.Put("/{gameID:[0-9]+}/players/{publicID:[0-9]+}", h)
		games.Put("/{gameID:[0-9]+}/clans/clan/{publicID:[0-9]+}", h)

		games.Post("/{gameID:[0-9]+}/clans", h)
		games.Post("/{gameID:[0-9]+}/players", h)
		games.Post("/{gameID:[0-9]+}/clans/{publicID:[0-9]+}/leave", h)
		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/application", h)
		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/application/:action", h)
		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/invitation", h)
		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/invitation/:action", h)
		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/delete", h)
		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/promote", h)
		games.Post("/{gameID:[0-9]+}/clans/{clanPublicID:[0-9]+}/memberships/demote", h)
	}

	myroute := app.Get("/anything/{anythingparameter:.*}", func(ctx *iris.Context) {
		s := ctx.Param("anythingparameter")
		ctx.Writef("The path after /anything is: %s", s)
	}) // .ChangeName("myroute")

	app.Get("/reverse_myroute", func(ctx *iris.Context) {
		// reverse routing snippet using templates:
		// https://github.com/kataras/iris/tree/v6/adaptors/view/_examples/template_html_3 (gorillamux)
		// https://github.com/kataras/iris/tree/v6/adaptors/view/_examples/template_html_4 (httprouter)

		myrouteRequestPath := app.Path(myroute.Name(), "anythingparameter", "something/here")
		ctx.Writef("Should be '/anything/something/here': %s", myrouteRequestPath)
	})

	p := app.Party("mysubdomain.")
	// http://mysubdomain.myhost.com/
	p.Get("/", h)

	app.Listen(":8080")
}

func h(ctx *iris.Context) {
	ctx.HTML(iris.StatusOK, "<h1>Path<h1/>"+ctx.Path())
}
