package main

import (
	"net/http"
	"strings"

	"github.com/kataras/iris"
)

// In this example you'll just see one use case of .WrapRouter.
// You can use the .WrapRouter to add custom logic when or when not the router should
// be executed in order to execute the registered routes' handlers.
//
// To see how you can serve files on root "/" without a custom wrapper
// just navigate to the "file-server/single-page-application" example.
//
// This is just for the proof of concept, you can skip this tutorial if it's too much for you.
func newApp() *iris.Application {

	app := iris.New()

	app.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
		ctx.HTML("<b>Resource Not found</b>")
	})

	app.Get("/", func(ctx iris.Context) {
		ctx.ServeFile("./public/index.html", false)
	})

	app.Get("/profile/{username}", func(ctx iris.Context) {
		ctx.Writef("Hello %s", ctx.Params().Get("username"))
	})

	// serve files from the root "/", if we used .StaticWeb it could override
	// all the routes because of the underline need of wildcard.
	// Here we will see how you can by-pass this behavior
	// by creating a new file server handler and
	// setting up a wrapper for the router(like a "low-level" middleware)
	// in order to manually check if we want to process with the router as normally
	// or execute the file server handler instead.

	// use of the .StaticHandler
	// which is the same as StaticWeb but it doesn't
	// registers the route, it just returns the handler.
	fileServer := app.StaticHandler("./public", false, false)

	// wrap the router with a native net/http handler.
	// if url does not contain any "." (i.e: .css, .js...)
	// (depends on the app , you may need to add more file-server exceptions),
	// then the handler will execute the router that is responsible for the
	// registered routes (look "/" and "/profile/{username}")
	// if not then it will serve the files based on the root "/" path.
	app.WrapRouter(func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
		path := r.URL.Path
		// Note that if path has suffix of "index.html" it will auto-permant redirect to the "/",
		// so our first handler will be executed instead.

		if !strings.Contains(path, ".") { // if it's not a resource then continue to the router as normally.
			router(w, r)
			return
		}
		// acquire and release a context in order to use it to execute
		// our file server
		// remember: we use net/http.Handler because here we are in the "low-level", before the router itself.
		ctx := app.ContextPool.Acquire(w, r)
		fileServer(ctx)
		app.ContextPool.Release(ctx)
	})

	return app
}

func main() {
	app := newApp()

	// http://localhost:8080
	// http://localhost:8080/index.html
	// http://localhost:8080/app.js
	// http://localhost:8080/css/main.css
	// http://localhost:8080/profile/anyusername
	app.Run(iris.Addr(":8080"))

	// Note: In this example we just saw one use case,
	// you may want to .WrapRouter or .Downgrade in order to bypass the iris' default router, i.e:
	// you can use that method to setup custom proxies too.
	//
	// If you just want to serve static files on other path than root
	// you can just use the StaticWeb, i.e:
	// 					     .StaticWeb("/static", "./public")
	// ________________________________requestPath, systemPath
}
