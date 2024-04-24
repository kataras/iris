package main

import (
	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New() // defaults to these

	tmpl := iris.HTML("./templates", ".html")
	tmpl.Reload(true) // reload templates on each request (development mode)
	// default template funcs are:
	//
	// - {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
	// - {{ render "header.html" . }}
	// - {{ render_r "header.html" . }} // partial relative path to current page
	// - {{ yield . }}
	// - {{ current . }}
	tmpl.AddFunc("greet", func(s string) string {
		return "Greetings " + s + "!"
	})
	app.RegisterView(tmpl)

	app.Get("/", hi)

	// http://localhost:8080
	app.Listen(":8080", iris.WithCharset("utf-8")) // defaults to that but you can change it.
}

func hi(ctx iris.Context) {
	ctx.ViewData("Title", "Hi Page")
	ctx.ViewData("Name", "iris") // {{.Name}} will render: iris
	// ctx.ViewData("", myCcustomStruct{})
	if err := ctx.View("hi.html"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
