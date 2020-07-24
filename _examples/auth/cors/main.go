// Package main integrates the "rs/cors" net/http middleware into Iris.
// That cors third-party middleware cannot be registered through `iris.FromStd`
// as a common middleware because it should be injected before the Iris Router itself,
// it allows/dissallows HTTP Methods too.
//
// This is just an example you can use to run something, based on custom logic,
// before the Iris Router itself.
//
// In the "routing/custom-wrapper" example
// we learn how we can acquire and release an Iris context to fire an Iris Handler
// based on custom logic, before the Iris Router itself. In that example
// we will fire a net/http handler (the "rs/cors" handler one) instead.
//
// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS
package main

import (
	"github.com/kataras/iris/v12"
	"github.com/rs/cors"
)

func main() {
	app := iris.New()
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		// Enable Debugging for testing, consider disabling in production
		Debug: true,
	})
	// app.WrapRouter(func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
	// 	[custom logic...]
	//  if shouldFireNetHTTPHandler {
	//    ...ServeHTTP(w,r)
	//    return
	//  }
	//  router(w,r)
	// })
	// In our case, the cors package has a ServeHTTP
	// of the same form of app.WrapRouter's accept input argument,
	// so we can just do:
	app.WrapRouter(c.ServeHTTP)

	// Serve ./public/index.html, main.js.
	app.HandleDir("/", iris.Dir("./public"))

	// Register routes here...
	app.Get("/data", listData)

	// http://localhost:8080 and click the "fetch data" button.
	app.Listen(":8080")
}

type item struct {
	Title string `json:"title"`
}

func listData(ctx iris.Context) {
	ctx.JSON([]item{
		{"Item 1"},
		{"Item 2"},
		{"Item 3"},
	})
}
