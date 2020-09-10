package main

import (
	"github.com/kataras/iris/v12"

	// go get -u github.com/gertd/go-pluralize
	"github.com/gertd/go-pluralize"
)

func main() {
	app := newApp()
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	pluralize := pluralize.NewClient()
	app.I18n.Loader.FuncMap = map[string]interface{}{
		"plural": func(word string, count int) string {
			// Your own implementation or use a 3rd-party package
			// like we do here.
			//
			// Note that this is only for english,
			// but you can accept the language code
			// and use a map with dictionaries to
			// pluralize words based on the given language.
			return pluralize.Pluralize(word, count, true)
		},
	}
	app.I18n.Load("./locales/*/*.yml", "en-US", "el-GR")

	app.Get("/", func(ctx iris.Context) {
		text := ctx.Tr("HiDogs", iris.Map{
			"locale": ctx.GetLocale(),
			"count":  2,
		}) // prints "Hi 2 dogs".
		ctx.WriteString(text)
	})

	return app
}
