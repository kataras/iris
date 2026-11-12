package view_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/view"

	"github.com/kataras/blocks"
)

func newApp() *iris.Application {
	// Create a new blocks instance.
	// It's a template engine that supports layouts and blocks.
	// Read about its syntax at: https://github.com/kataras/blocks
	templates := blocks.New("./_testdata/views")
	// Load the templates or panic.
	if err := templates.Load(); err != nil {
		panic(err)
	}

	// Create a new Iris application.
	app := iris.New()
	// Register the view engine as a top-level router middleware.
	// Note: You can register more view engines at children-level too.
	app.UseRouter(view.Handler(templates))

	configureRoutes(app)
	return app
}

func configureRoutes(app *iris.Application) {
	app.Get("/", index)
	app.Get("/500", internalServerError)

	// The same with "/" but with different syntax.
	app.Get("/index_2", view.WithTemplate(&view.Template{
		Code:   iris.StatusOK,
		Name:   "index",
		Layout: "main",
		Data: iris.Map{
			"Title": "Page Title",
		},
	}))

	// The same with "/500" but with different syntax.
	app.Get("/500_2", view.WithTemplate(&view.Template{ // or ctx.Render(&view.View{...})
		Code:   iris.StatusInternalServerError,
		Name:   "500",
		Layout: "error",
		Data: iris.Map{
			"Code":    iris.StatusInternalServerError,
			"Message": "Internal Server Error",
		},
		// You can register request data as well:
		// DataFunc: func(ctx iris.Context) (any, error) {
		//
		// },
	}))
}

func index(ctx iris.Context) {
	data := iris.Map{
		"Title": "Page Title",
	}

	view.Layout(ctx, "main")
	if err := view.Render(ctx, "index", "", data); err != nil {
		ctx.HTML(fmt.Sprintf("<h3>%s</h3>", err.Error()))
		return
	}
}

func internalServerError(ctx iris.Context) {
	ctx.StatusCode(iris.StatusInternalServerError)

	data := iris.Map{
		"Code":    iris.StatusInternalServerError,
		"Message": "Internal Server Error",
	}

	view.Layout(ctx, "error")
	if err := view.Render(ctx, "500", "", data); err != nil {
		ctx.HTML(fmt.Sprintf("<h3>%s</h3>", err.Error()))
		return
	}
}

func TestBlocks(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	e.GET("/").Expect().Status(httptest.StatusOK).Body().Contains("Page Title")
	e.GET("/500").Expect().Status(httptest.StatusInternalServerError).Body().Contains("Internal Server Error")

	e.GET("/index_2").Expect().Status(httptest.StatusOK).Body().Contains("Page Title")
	e.GET("/500_2").Expect().Status(httptest.StatusInternalServerError).Body().Contains("Internal Server Error")
}
