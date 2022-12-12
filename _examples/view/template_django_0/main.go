package main

import (
	"time"

	"github.com/kataras/iris/v12"
	// optionally, register filters like `timesince`.
	_ "github.com/iris-contrib/pongo2-addons/v4"
)

var startTime = time.Now()

func main() {
	app := iris.New()

	tmpl := iris.Django("./templates", ".html")
	tmpl.Reload(true)                             // reload templates on each request (development mode)
	tmpl.AddFunc("greet", func(s string) string { // {{greet(name)}}
		return "Greetings " + s + "!"
	})

	// tmpl.RegisterFilter("myFilter", myFilter) // {{"simple input for filter"|myFilter}}
	app.RegisterView(tmpl)

	app.Get("/", hi)

	// http://localhost:8080
	app.Listen(":8080")
}

func hi(ctx iris.Context) {
	// ctx.ViewData("title", "Hi Page")
	// ctx.ViewData("name", "iris")
	// ctx.ViewData("serverStartTime", startTime)
	// or if you set all view data in the same handler you can use the
	// iris.Map/pongo2.Context/map[string]interface{}, look below:

	if err := ctx.View("hi.html", iris.Map{
		"title":           "Hi Page",
		"name":            "iris",
		"serverStartTime": startTime,
	}); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
