// Package main an example on how to naming your routes & use the custom 'url' HTML Template Engine, same for other template engines.
package main

import (
	"github.com/cdren/iris"
	"github.com/cdren/iris/context"
	"github.com/cdren/iris/core/router"
	"github.com/cdren/iris/view"
)

const (
	host = "127.0.0.1:8080"
)

func main() {
	app := iris.New()

	// create a custom path reverser, iris let you define your own host and scheme
	// which is useful when you have nginx or caddy in front of iris.
	rv := router.NewRoutePathReverser(app, router.WithHost(host), router.WithScheme("http"))
	// locate and define our templates as usual.
	templates := view.HTML("./templates", ".html")
	// add a custom func of "url" and pass the rv.URL as its template function body,
	// so {{url "routename" "paramsOrSubdomainAsFirstArgument"}} will work inside our templates.
	templates.AddFunc("url", rv.URL)

	app.AttachView(templates)

	// wildcard subdomain, will catch username1.... username2.... username3... username4.... username5...
	// that our below links are providing via page.html's first argument which is the subdomain.

	subdomain := app.Party("*.")

	mypathRoute, _ := subdomain.Get("/mypath", emptyHandler)
	mypathRoute.Name = "my-page1"

	mypath2Route, _ := subdomain.Get("/mypath2/{paramfirst}/{paramsecond}", emptyHandler)
	mypath2Route.Name = "my-page2"

	mypath3Route, _ := subdomain.Get("/mypath3/{paramfirst}/statichere/{paramsecond}", emptyHandler)
	mypath3Route.Name = "my-page3"

	mypath4Route, _ := subdomain.Get("/mypath4/{paramfirst}/statichere/{paramsecond}/{otherparam}/{something:path}", emptyHandler)
	mypath4Route.Name = "my-page4"

	mypath5Route, _ := subdomain.Handle("GET", "/mypath5/{paramfirst}/statichere/{paramsecond}/{otherparam}/anything/{something:path}", emptyHandler)
	mypath5Route.Name = "my-page5"

	mypath6Route, err := subdomain.Get("/mypath6/{paramfirst}/{paramsecond}/staticParam/{paramThirdAfterStatic}", emptyHandler)
	if err != nil { // catch any route problems when declare a route or on err := app.Run(...); err != nil { panic(err) }
		panic(err)
	}
	mypath6Route.Name = "my-page6"

	app.Get("/", func(ctx context.Context) {
		// for username5./mypath6...
		paramsAsArray := []string{"username5", "theParam1", "theParam2", "paramThirdAfterStatic"}
		ctx.ViewData("ParamsAsArray", paramsAsArray)
		if err := ctx.View("page.html"); err != nil {
			panic(err)
		}
	})

	// http://127.0.0.1:8080
	app.Run(iris.Addr(host))
}

func emptyHandler(ctx context.Context) {
	ctx.Writef("Hello from subdomain: %s , you're in path:  %s", ctx.Subdomain(), ctx.Path())
}

// Note:
// If you got an empty string on {{ url }} or {{ urlpath }} it means that
// args length are not aligned with the route's parameters length
// or the route didn't found by the passed name.
