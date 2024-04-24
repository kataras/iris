package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./view", ".html"))

	// Use the FallbackView helper Register a fallback view
	// filename per-party when the provided was not found.
	app.FallbackView(iris.FallbackView("fallback.html"))

	// Use the FallbackViewLayout helper to register a fallback view layout.
	app.FallbackView(iris.FallbackViewLayout("layout.html"))

	// Register a custom fallback function per-party to handle everything.
	// You can register more than one. If fails (returns a not nil error of ErrViewNotExists)
	// then it proceeds to the next registered fallback.
	app.FallbackView(iris.FallbackViewFunc(func(ctx iris.Context, err iris.ErrViewNotExist) error {
		// err.Name is the previous template name.
		// err.IsLayout reports whether the failure came from the layout template.
		// err.Data is the template data provided to the previous View call.
		// [...custom logic e.g. ctx.View("fallback.html", err.Data)]
		return err
	}))

	app.Get("/", index)

	app.Listen(":8080")
}

// Register fallback view(s) in a middleware.
// func fallbackInsideAMiddleware(ctx iris.Context) {
// 	ctx.FallbackView(...)
//  To remove all previous registered fallbacks, pass nil.
//  ctx.FallbackView(nil)
// 	ctx.Next()
// }

func index(ctx iris.Context) {
	if err := ctx.View("blabla.html"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
