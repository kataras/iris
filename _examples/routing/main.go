package main

import (
	"io/ioutil"

	"github.com/kataras/iris"
)

/*
Read:
"overview"
"basic"
"dynamic-path"
and "reverse" examples if you want to release iris' real power.
*/

const maxBodySize = 1 << 20
const notFoundHTML = "<h1> custom http error page </h1>"

func registerErrors(app *iris.Application) {
	// set a custom 404 handler
	app.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
		ctx.HTML(notFoundHTML)
	})
}

func registerGamesRoutes(app *iris.Application) {
	gamesMiddleware := func(ctx iris.Context) {
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

		gamesCh := games.Party("/challenge")
		{
			// games/challenge
			gamesCh.Get("/", h)

			gamesChBeginner := gamesCh.Party("/beginner")
			{
				// games/challenge/beginner/start
				gamesChBeginner.Get("/start", h)
				levelBeginner := gamesChBeginner.Party("/level")
				{
					// games/challenge/beginner/level/first
					levelBeginner.Get("/first", h)
				}
			}

			gamesChIntermediate := gamesCh.Party("/intermediate")
			{
				// games/challenge/intermediate
				gamesChIntermediate.Get("/", h)
				// games/challenge/intermediate/start
				gamesChIntermediate.Get("/start", h)
			}
		}

	}
}

func registerSubdomains(app *iris.Application) {
	mysubdomain := app.Party("mysubdomain.")
	// http://mysubdomain.myhost.com
	mysubdomain.Get("/", h)

	willdcardSubdomain := app.Party("*.")
	willdcardSubdomain.Get("/", h)
	willdcardSubdomain.Party("/party").Get("/", h)
}

func newApp() *iris.Application {
	app := iris.New()
	registerErrors(app)
	registerGamesRoutes(app)
	registerSubdomains(app)

	app.Handle("GET", "/healthcheck", h)

	// "POST" method
	// this handler reads raw body from the client/request
	// and sends back the same body
	// remember, we have limit to that body in order
	// to protect ourselves from "over heating".
	app.Post("/", iris.LimitRequestBodySize(maxBodySize), func(ctx iris.Context) {
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

	return app
}

func h(ctx iris.Context) {
	method := ctx.Method()       // the http method requested a server's resource.
	subdomain := ctx.Subdomain() // the subdomain, if any.

	// the request path (without scheme and host).
	path := ctx.Path()
	// how to get all parameters, if we don't know
	// the names:
	paramsLen := ctx.Params().Len()

	ctx.Params().Visit(func(name string, value string) {
		ctx.Writef("%s = %s\n", name, value)
	})
	ctx.Writef("Info\n\n")
	ctx.Writef("Method: %s\nSubdomain: %s\nPath: %s\nParameters length: %d", method, subdomain, path, paramsLen)
}

func main() {
	app := newApp()

	/*
		// GET
		http://localhost:8080/healthcheck
		http://localhost:8080/games/42/clans
		http://localhost:8080/games/42/clans/clan/93
		http://localhost:8080/games/42/clans/search
		http://mysubdomain.localhost:8080/

		// PUT
		http://localhost:8080/games/42/players/93
		http://localhost:8080/games/42/clans/clan/93

		// POST
		http://localhost:8080/
		http://localhost:8080/games/42/clans
		http://localhost:8080/games/42/players
		http://localhost:8080/games/42/clans/93/leave
		http://localhost:8080/games/42/clans/93/memberships/application
		http://localhost:8080/games/42/clans/93/memberships/application/anystring
		http://localhost:8080/games/42/clans/93/memberships/invitation
		http://localhost:8080/games/42/clans/93/memberships/invitation/anystring
		http://localhost:8080/games/42/clans/93/memberships/delete
		http://localhost:8080/games/42/clans/93/memberships/promote
		http://localhost:8080/games/42/clans/93/memberships/demote

		// FIRE NOT FOUND
		http://localhost:8080/coudlntfound
	*/
	app.Run(iris.Addr(":8080"))
}
