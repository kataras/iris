package main

import (
	"github.com/kataras/iris/v12"
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
		games.Get("/{gameID:uint64}/clans", h)
		games.Get("/{gameID:uint64}/clans/clan/{clanPublicID:uint64}", h)
		games.Get("/{gameID:uint64}/clans/search", h)

		// "PUT" method
		games.Put("/{gameID:uint64}/players/{clanPublicID:uint64}", h)
		games.Put("/{gameID:uint64}/clans/clan/{clanPublicID:uint64}", h)
		// remember: "clanPublicID" should not be changed to other routes with the same prefix.
		// "POST" method
		games.Post("/{gameID:uint64}/clans", h)
		games.Post("/{gameID:uint64}/players", h)
		games.Post("/{gameID:uint64}/clans/{clanPublicID:uint64}/leave", h)
		games.Post("/{gameID:uint64}/clans/{clanPublicID:uint64}/memberships/application", h)
		games.Post("/{gameID:uint64}/clans/{clanPublicID:uint64}/memberships/application/{action}", h) // {action} == {action:string}
		games.Post("/{gameID:uint64}/clans/{clanPublicID:uint64}/memberships/invitation", h)
		games.Post("/{gameID:uint64}/clans/{clanPublicID:uint64}/memberships/invitation/{action}", h)
		games.Post("/{gameID:uint64}/clans/{clanPublicID:uint64}/memberships/delete", h)
		games.Post("/{gameID:uint64}/clans/{clanPublicID:uint64}/memberships/promote", h)
		games.Post("/{gameID:uint64}/clans/{clanPublicID:uint64}/memberships/demote", h)

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
	mysubdomain := app.Subdomain("mysubdomain")
	// http://mysubdomain.myhost.com
	mysubdomain.Get("/", h)

	willdcardSubdomain := app.WildcardSubdomain()
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
		b, err := ctx.GetBody()
		// if is larger then send a bad request status
		if err != nil {
			ctx.StatusCode(iris.StatusBadRequest)
			ctx.WriteString(err.Error())
			return
		}
		// send back the post body
		ctx.Write(b)
	})

	app.HandleMany("POST PUT", "/postvalue", func(ctx iris.Context) {
		name := ctx.PostValueDefault("name", "iris")
		headervale := ctx.GetHeader("headername")
		ctx.Writef("Hello %s | %s", name, headervale)
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
	app.Logger().SetLevel("debug")

	/*
		// GET
		http://localhost:8080/healthcheck
		http://localhost:8080/games/42/clans
		http://localhost:8080/games/42/clans/clan/93
		http://localhost:8080/games/42/clans/search
		http://mysubdomain.localhost:8080/

		// PUT
		http://localhost:8080/postvalue
		http://localhost:8080/games/42/players/93
		http://localhost:8080/games/42/clans/clan/93

		// POST
		http://localhost:8080/
		http://localhost:8080/postvalue
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
	app.Listen(":8080")
}
