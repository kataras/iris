package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	templates := iris.HTML("./views", ".html").Layout("shared/layout.html")
	app.RegisterView(templates)

	app.Controller("/", new(Controller))

	// http://localhost:9091
	app.Run(iris.Addr(":9091"))
}

// Layout contains all the binding properties for the shared/layout.html
type Layout struct {
	Title string
}

// Controller is our example controller.
type Controller struct {
	iris.Controller

	Layout Layout `iris:"model"`
}

// BeginRequest is the first method fires when client requests from this Controller's path.
func (c *Controller) BeginRequest(ctx iris.Context) {
	c.Controller.BeginRequest(ctx)

	c.Layout.Title = "Home Page"
}

// Get handles GET http://localhost:9091
func (c *Controller) Get() {
	c.Tmpl = "index.html"
	c.Data["Message"] = "Welcome to my website!"
}
