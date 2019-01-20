package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/mvc"
)

func main() {
	app := iris.New()

	templates := iris.HTML("./views", ".html").Layout("shared/layout.html")
	app.RegisterView(templates)

	mvc.New(app).Handle(new(Controller))

	// http://localhost:9091
	app.Run(iris.Addr(":9091"))
}

// Layout contains all the binding properties for the shared/layout.html
type Layout struct {
	Title string
}

// Controller is our example controller, request-scoped, each request has its own instance.
type Controller struct {
	Layout Layout
}

// BeginRequest is the first method fired when client requests from this Controller's root path.
func (c *Controller) BeginRequest(ctx iris.Context) {
	c.Layout.Title = "Home Page"
}

// EndRequest is the last method fired.
// It's here just to complete the BaseController
// in order to be tell iris to call the `BeginRequest` before the main method.
func (c *Controller) EndRequest(ctx iris.Context) {}

// Get handles GET http://localhost:9091
func (c *Controller) Get() mvc.View {
	return mvc.View{
		Name: "index.html",
		Data: iris.Map{
			"Layout":  c.Layout,
			"Message": "Welcome to my website!",
		},
	}
}
