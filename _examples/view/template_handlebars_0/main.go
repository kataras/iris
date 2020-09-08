package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	// Init the handlebars engine
	e := iris.Handlebars("./templates", ".html").Reload(true)
	// Register a helper.
	e.AddFunc("fullName", func(person map[string]string) string {
		return person["firstName"] + " " + person["lastName"]
	})

	app.RegisterView(e)

	app.Get("/", func(ctx iris.Context) {
		viewData := iris.Map{
			"author": map[string]string{"firstName": "Jean", "lastName": "Valjean"},
			"body":   "Life is difficult",
			"comments": []iris.Map{{
				"author": map[string]string{"firstName": "Marcel", "lastName": "Beliveau"},
				"body":   "LOL!",
			}},
		}

		ctx.View("example.html", viewData)
	})

	exampleRouter := app.Party("/example")
	/* See context-view-data example: Set data through one or more middleware */
	exampleRouter.Use(func(ctx iris.Context) {
		ctx.ViewData("author", map[string]string{"firstName": "Jean", "lastName": "Valjean"})
		ctx.ViewData("body", "Life is difficult")
		ctx.ViewData("comments", []iris.Map{{
			"author": map[string]string{"firstName": "Marcel", "lastName": "Beliveau"},
			"body":   "LOL!",
		}})

		// OR:
		// ctx.ViewData("", iris.Map{
		// 	"author": map[string]string{"firstName": "Jean", "lastName": "Valjean"},
		// 	"body":   "Life is difficult",
		// 	"comments": []iris.Map{{
		// 		"author": map[string]string{"firstName": "Marcel", "lastName": "Beliveau"},
		// 		"body":   "LOL!",
		// 	}},
		// })

		ctx.Next()
	})

	mvc.New(exampleRouter).Handle(new(controller))

	// Read more about its syntax at:
	// https://github.com/aymerick/raymond and
	// https://handlebarsjs.com/guide

	// http://localhost:8080
	// http://localhost:8080/example
	app.Listen(":8080")
}

type controller struct{}

func (c *controller) Get() mvc.Result {
	return mvc.View{
		Name: "example",
		Code: 200,
	}
}
