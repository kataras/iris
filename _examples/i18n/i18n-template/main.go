package main

import (
	"strings"
	"text/template"

	"github.com/kataras/iris/v12"
	// go get -u github.com/gertd/go-pluralize
	"github.com/gertd/go-pluralize"
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

	pluralize := pluralize.NewClient()

	// Set custom functions per locale!
	app.I18n.Loader.Funcs = func(current iris.Locale) template.FuncMap {
		return template.FuncMap{
			"plural": func(word string, count int) string {
				// Your own implementation or use a 3rd-party package
				// like we do here.
				//
				// Note that this is only for english,
				// but you can use the "current" locale
				// and make a map with dictionaries to
				// pluralize words based on the given language.
				return pluralize.Pluralize(word, count, true)
			},
			"uppercase": func(word string) string {
				return strings.ToUpper(word)
			},
			"concat": func(words ...string) string {
				return strings.Join(words, " ")
			},
		}
	}

	app.I18n.Load("./locales/*/*", "en-US", "el-GR")

	app.Get("/", func(ctx iris.Context) {
		text := ctx.Tr("HiDogs", iris.Map{
			"count": 2,
		}) // en-US: prints "Hi 2 dogs".
		ctx.WriteString(text)
	})

	app.Get("/singular", func(ctx iris.Context) {
		text := ctx.Tr("HiDogs", iris.Map{
			"count": 1,
		}) // en-US: prints "Hi 1 dog".
		ctx.WriteString(text)
	})

	app.Get("/members", func(ctx iris.Context) {
		text := ctx.Tr("forms.registered_members", iris.Map{
			"count": 42,
		}) // en-US: prints "There are 42 members registered".
		ctx.WriteString(text)
	})

	return app
}
