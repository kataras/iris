package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New() // defaults to these

	// - standard html  | iris.HTML(...)
	// - django         | iris.Django(...)
	// - pug(jade)      | iris.Pug(...)
	// - handlebars     | iris.Handlebars(...)
	// - amber          | iris.Amber(...)

	tmpl := iris.HTML("./templates", ".html")
	tmpl.Reload(true) // reload templates on each request (development mode)
	// default template funcs are:
	//
	// - {{ urlpath "mynamedroute" "pathParameter_ifneeded" }}
	// - {{ render "header.html" }}
	// - {{ render_r "header.html" }} // partial relative path to current page
	// - {{ yield }}
	// - {{ current }}
	tmpl.AddFunc("greet", func(s string) string {
		return "Greetings " + s + "!"
	})
	app.RegisterView(tmpl)

	app.Get("/", hi)

	// http://localhost:8080
	app.Run(iris.Addr(":8080"), iris.WithCharset("UTF-8")) // defaults to that but you can change it.
}

func hi(ctx iris.Context) {
	ctx.ViewData("Title", "Hi Page")
	ctx.ViewData("Name", "iris") // {{.Name}} will render: iris
	// ctx.ViewData("", myCcustomStruct{})
	ctx.View("hi.html")
}

/*
Note:

In case you're wondering, the code behind the view engines derives from the "github.com/kataras/iris/view" package,
access to the engines' variables can be granded by "github.com/kataras/iris" package too.

    iris.HTML(...) is a shortcut of view.HTML(...)
    iris.Django(...)     >> >>      view.Django(...)
    iris.Pug(...)        >> >>      view.Pug(...)
    iris.Handlebars(...) >> >>      view.Handlebars(...)
    iris.Amber(...)      >> >>      view.Amber(...)
*/
