package main

import (
	"net/http"
	"strings"

	"github.com/kataras/iris/v12"
)

// In this example you'll just see one use case of .WrapRouter.
// You can use the .WrapRouter to add custom logic when or when not the router should
// be executed in order to execute the registered routes' handlers.
func newApp() *iris.Application {
	app := iris.New()

	app.OnErrorCode(iris.StatusNotFound, func(ctx iris.Context) {
		ctx.HTML("<b>Resource Not found</b>")
	})

	app.Get("/profile/{username}", func(ctx iris.Context) {
		ctx.Writef("Hello %s", ctx.Params().Get("username"))
	})

	app.HandleDir("/", iris.Dir("./public"))

	myOtherHandler := func(ctx iris.Context) {
		ctx.Writef("inside a handler which is fired manually by our custom router wrapper")
	}

	// wrap the router with a native net/http handler.
	// if url does not contain any "." (i.e: .css, .js...)
	// (depends on the app , you may need to add more file-server exceptions),
	// then the handler will execute the router that is responsible for the
	// registered routes (look "/" and "/profile/{username}")
	// if not then it will serve the files based on the root "/" path.
	app.WrapRouter(func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
		path := r.URL.Path

		if strings.HasPrefix(path, "/other") {
			// acquire and release a context in order to use it to execute
			// our custom handler
			// remember: we use net/http.Handler because here we are in the "low-level", before the router itself.
			ctx := app.ContextPool.Acquire(w, r)
			myOtherHandler(ctx)
			app.ContextPool.Release(ctx)
			return
		}

		router.ServeHTTP(w, r) // else continue serving routes as usual.
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
	// http://localhost:8080/other/random
	app.Listen(":8080")

	// Note: In this example we just saw one use case,
	// you may want to .WrapRouter or .Downgrade in order to bypass the iris' default router, i.e:
	// you can use that method to setup custom proxies too.
}
