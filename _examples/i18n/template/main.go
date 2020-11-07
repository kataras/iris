package main

import (
	"strings"
	"text/template"

	"github.com/kataras/iris/v12"
)

/*
 Iris I18n supports text/template inside the translation values.
 Follow this example to learn how to use that feature.
*/

func main() {
	app := newApp()
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	// Set custom functions per locale!
	app.I18n.Loader.Funcs = func(current iris.Locale) template.FuncMap {
		return template.FuncMap{
			"uppercase": func(word string) string {
				return strings.ToUpper(word)
			},
		}
	}

	err := app.I18n.Load("./locales/*/*.ini", "en-US", "el-GR")
	if err != nil {
		panic(err)
	}

	app.Get("/", func(ctx iris.Context) {
		text := ctx.Tr("forms.register") // en-US: prints "Become a MEMBER".
		ctx.WriteString(text)
	})

	app.Get("/title", func(ctx iris.Context) {
		text := ctx.Tr("user.connections.Title") // en-US: prints "Accounts Connections".
		ctx.WriteString(text)
	})

	return app
}
