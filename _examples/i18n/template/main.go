package main

import (
	"strings"
	"text/template"

	"github.com/kataras/iris/v12"

	// go get -u golang.org/x/text/message

	"golang.org/x/text/feature/plural"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

/*
 Iris I18n supports text/template inside the translation values.
 Follow this example to learn how to use that feature.

 This is just an example on how to use template functions.
 See the "plurals" example for a more comprehensive pluralization support instead.
*/

func main() {
	app := newApp()
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	// set the printers after load, so they can be done by loop of available languages.
	printers := make(map[string]*message.Printer)

	message.Set(language.Greek, "Hello %d dog",
		plural.Selectf(1, "%d",
			"one", "Γεια σου σκυλί",
			"other", "Γεια σας %[1]d σκυλιά",
		))

	/* by variable, single word:
	message.Set(language.Greek, "Hi %d dog(s)",
		catalog.Var("dogs", plural.Selectf(1, "%d", "one", "σκυλί", "other", "σκυλιά")),
		catalog.String("Γεια %[1]d ${dogs}"))
	*/

	// Set custom functions per locale!
	app.I18n.Loader.Funcs = func(current iris.Locale) template.FuncMap {
		return template.FuncMap{
			"plural": func(word string, count int) string {
				// Your own implementation or use a 3rd-party package
				// like we do here.
				return printers[current.Language()].Sprintf(word, count)
			},
			"uppercase": func(word string) string {
				return strings.ToUpper(word)
			},
			"concat": func(words ...string) string {
				return strings.Join(words, " ")
			},
		}
	}

	err := app.I18n.Load("./locales/*/*", "en-US", "el-GR")
	if err != nil {
		panic(err)
	}

	for _, tag := range app.I18n.Tags() {
		printers[tag.String()] = message.NewPrinter(tag)
	}

	message.NewPrinter(language.Greek).Printf("Hello %d dog", 2)

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

	// showcases the other.ini translation file.
	app.Get("/other", func(ctx iris.Context) {
		ctx.Writef(`AccessLogClear: %s
Title: %s`, ctx.Tr("debug.AccessLogClear"), ctx.Tr("user.connections.Title"))
	})

	return app
}
