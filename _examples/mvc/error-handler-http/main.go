package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

func main() {
	app := newApp()
	app.Logger().SetLevel("debug")
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))

	m := mvc.New(app)
	m.Handle(new(controller))

	return app
}

type controller struct{}

func (c *controller) Get() string {
	return "Hello!"
}

// The input parameter of mvc.Code is optional but a good practise to follow.
// You could  register a Context and get its error code through ctx.GetStatusCode().
//
// This can accept dependencies and output values like any other Controller Method,
// however be careful if your registered dependencies depend only on successful(200...) requests.
//
// Also note that, if you register more than one controller.HandleHTTPError
// in the same Party, you need to use the RouteOverlap feature as shown
// in the "authenticated-controller" example, and a dependency on
// a controller's field (or method's input argument) is required
// to select which, between those two controllers, is responsible
// to handle http errors.
func (c *controller) HandleHTTPError(statusCode mvc.Code) mvc.View {
	code := int(statusCode) // cast it to int.

	view := mvc.View{
		Code: code,
		Name: "unexpected-error.html",
	}

	switch code {
	case 404:
		view.Name = "404.html"
		// [...]
	case 500:
		view.Name = "500.html"
	}

	return view
}
