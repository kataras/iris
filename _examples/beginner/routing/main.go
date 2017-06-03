package main

import (
	"io/ioutil"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

/*
Read:
"basic"
"dynamic-path"
and "reverse" examples if you want to release Iris' real power.
*/

const maxBodySize = 1 << 20

var app *iris.Application

func init() {
	app = iris.New()
}

func registerErrors() {
	// set a custom 404 handler
	app.OnErrorCode(iris.StatusNotFound, func(ctx context.Context) {
		ctx.HTML("<h1> custom http error page </h1>")
	})
}

func registerGamesRoutes() {
	gamesMiddleware := func(ctx context.Context) {
		println(ctx.Method() + ": " + ctx.Path())
		ctx.Next()
	}

	// party is just a group of routes with the same prefix
	// and middleware, i.e: "/games" and gamesMiddleware.
	games := app.Party("/games", gamesMiddleware)
	{ // braces are optional of course, it's just a style of code

		// "GET" method
		games.Get("/{gameID:int}/clans", h)
		games.Get("/{gameID:int}/clans/clan/{clanPublicID:int}", h)
		games.Get("/{gameID:int}/clans/search", h)

		// "PUT" method
		games.Put("/{gameID:int}/players/{clanPublicID:int}", h)
		games.Put("/{gameID:int}/clans/clan/{clanPublicID:int}", h)
		// remember: "clanPublicID" should not be changed to other routes with the same prefix.
		// "POST" method
		games.Post("/{gameID:int}/clans", h)
		games.Post("/{gameID:int}/players", h)
		games.Post("/{gameID:int}/clans/{clanPublicID:int}/leave", h)
		games.Post("/{gameID:int}/clans/{clanPublicID:int}/memberships/application", h)
		games.Post("/{gameID:int}/clans/{clanPublicID:int}/memberships/application/{action}", h) // {action} == {action:string}
		games.Post("/{gameID:int}/clans/{clanPublicID:int}/memberships/invitation", h)
		games.Post("/{gameID:int}/clans/{clanPublicID:int}/memberships/invitation/{action}", h)
		games.Post("/{gameID:int}/clans/{clanPublicID:int}/memberships/delete", h)
		games.Post("/{gameID:int}/clans/{clanPublicID:int}/memberships/promote", h)
		games.Post("/{gameID:int}/clans/{clanPublicID:int}/memberships/demote", h)
	}
}

func registerSubdomains() {
	mysubdomain := app.Party("mysubdomain.")
	// http://mysubdomain.myhost.com
	mysubdomain.Get("/", func(ctx context.Context) {
		ctx.Writef("Hello from subdomain: %s , from host: %s, method: %s and path: %s", ctx.Subdomain(), ctx.Host(), ctx.Method(), ctx.Path())
	})
}
func main() {
	registerErrors()
	registerGamesRoutes()
	registerSubdomains()

	// more random examples below:

	app.Handle("GET", "/healthcheck", h)

	// "POST" method
	// this handler reads raw body from the client/request
	// and sends back the same body
	// remember, we have limit to that body in order
	// to protect ourselves from "over heating".
	app.Post("/", func(ctx context.Context) {
		ctx.SetMaxRequestBodySize(maxBodySize) // set max request body that client can send.
		// get request body
		b, err := ioutil.ReadAll(ctx.Request().Body)
		// if is larger then send a bad request status
		if err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.Writef(err.Error())
			return
		}
		// send back the post body
		ctx.Write(b)
	})

	// start the server on 0.0.0.0:8080
	app.Run(iris.Addr(":8080"))
}

func h(ctx context.Context) {
	ctx.HTML("<h1>Path: " + ctx.Path() + "</h1>")
}
