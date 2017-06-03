package main

// We don't have to reinvert the wheel, so we will use a good cors middleware
// as a router wrapper for our entire app.
// Follow the steps below:
//  +------------------------------------------------------------+
//  | 		Cors installation                                    |
//  +------------------------------------------------------------+
// 			go get -u github.com/rs/cors
//
//  +------------------------------------------------------------+
//  | 		Cors wrapper usage                                   |
//  +------------------------------------------------------------+
//			import "github.com/rs/cors"
//
// 			app := iris.New()
// 			corsOptions := cors.Options{/* your options here */}
// 			corsWrapper := cors.New(corsOptions).ServeHTTP
// 			app.Wrap(corsWrapper)
//
// 			[your code goes here...]
//
// 			app.Run(iris.Addr(":8080"))

import (
	"github.com/rs/cors"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

func main() {

	app := iris.New()
	corsOptions := cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	}

	corsWrapper := cors.New(corsOptions).ServeHTTP

	app.WrapRouter(corsWrapper)

	v1 := app.Party("/api/v1")
	{
		v1.Get("/", h)
		v1.Put("/put", h)
		v1.Post("/post", h)
	}

	app.Run(iris.Addr(":8080"))
}

func h(ctx context.Context) {
	ctx.Application().Log(ctx.Path())
	ctx.Writef("Hello from %s", ctx.Path())
}
