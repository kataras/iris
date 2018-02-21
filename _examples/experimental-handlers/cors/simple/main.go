package main

// $ go get github.com/rs/cors
// $ go run main.go

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/core/router"

	"github.com/iris-contrib/middleware/cors"
)

func main() {

	app := iris.New()
	crs := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // allows everything, use that to change the hosts.
		AllowedMethods:   router.AllMethods[:],
		AllowCredentials: true,
	})

	v1 := app.Party("/api/v1")
	v1.Use(crs)
	{
		v1.Get("/home", func(ctx iris.Context) {
			ctx.WriteString("Hello from /home")
		})
		v1.Get("/about", func(ctx iris.Context) {
			ctx.WriteString("Hello from /about")
		})
		v1.Post("/send", func(ctx iris.Context) {
			ctx.WriteString("sent")
		})
		v1.Put("/send", func(ctx iris.Context) {
			ctx.WriteString("updated")
		})
		v1.Delete("/send", func(ctx iris.Context) {
			ctx.WriteString("deleted")
		})
	}

	// or use that to wrap the entire router
	// even before the path and method matching
	// this should work better and with all cors' features.
	// Use that instead, if suits you.
	// app.WrapRouter(cors.WrapNext(cors.Options{
	// 	AllowedOrigins:   []string{"*"},
	// 	AllowCredentials: true,
	// }))
	app.Run(iris.Addr("localhost:8080"))
}
