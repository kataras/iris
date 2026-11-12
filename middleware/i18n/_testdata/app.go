package _testdata

import (
	"fmt"
	"html/template"

	"github.com/kataras/blocks"
	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/core/router"

	"github.com/kataras/iris/v14/middleware/i18n"
	"github.com/kataras/iris/v14/middleware/view"
)

/*
	See i18n-template for a more advanced translation key-values.
*/
// "./_testdata/locales/*/*"
func useLocalization(r router.Router, localesPath string) *i18n.I18n /* we return this value because we need it to bind the view's "tr" template function */ {
	// Load the translations.
	locales := i18n.New()
	// Set the default message func for the localization, first.
	defaultMessageFunc := func(langMatched, key string, args ...any) string {
		msg := fmt.Sprintf("lang: %s: not found key: %s: args: %v", langMatched, key, args)
		r.Application().Logger().Warn(msg)
		return msg
	}
	locales.SetDefaultMessageFunc(defaultMessageFunc)
	// First parameter: Glob filpath patern,
	// Second variadic parameter: Optional language tags, the first one is the default/fallback one.
	if err := locales.Load(localesPath, "en-US", "el-GR", "zh-CN"); err != nil {
		panic(err)
	}
	// Set the English language as the default one.
	// Although because we defined the languages explicitly above (Load) the default is set automatically.
	locales.SetDefault("en-US")

	// Register the i18n router, optionally, to redirect locale (e.g en-us.example.com) subdomains and paths (e.g. /en-US/api/user).
	router := i18n.NewRouter(locales, i18n.RouterOptions{
		Cookie:    "lang",
		Subdomain: true,
	})
	r.UseRouter(router)

	// Register the i18n handler, required, to retrieve the matched locale
	// through i18n.GetLocale, i18n.GetMessage package-level functions and e.t.c.
	handler := i18n.NewHandler(locales, i18n.HandlerOptions{
		URLParameter: "lang",
		Subdomain:    true,
		Cookie:       "lang",
	})
	r.Use(handler)

	return locales
}

// NewApp returns the Iris Application to test.
// localesGlob = "./_testdata/locales/*/*"
// viewsPath = ./_testdata/views
func NewApp(localesGlob, viewsPath string) *iris.Application {
	app := iris.New()
	locales := useLocalization(app, localesGlob)

	app.Get("/not-matched", func(ctx iris.Context) {
		text := i18n.GetMessage(ctx, "not_found_key", "some", "values", 42)
		ctx.WriteString(text)
		// lang: en-US: not found key: not_found_key: args: [some values 42]
	})

	app.Get("/", func(ctx iris.Context) {
		hi := i18n.GetMessage(ctx, "hi", "iris")
		locale := i18n.GetLocale(ctx)

		ctx.Writef("From the language %s translated output: %s", locale.Language(), hi)
	})

	app.Get("/some-path", func(ctx iris.Context) {
		ctx.Writef("%s", i18n.GetMessage(ctx, "hi", "iris"))
	})

	app.Get("/other", func(ctx iris.Context) {
		language := i18n.GetLocale(ctx).Language()

		fromFirstFileValue := i18n.GetMessage(ctx, "key1")
		fromSecondFileValue := i18n.GetMessage(ctx, "key2")
		ctx.Writef("From the language: %s, translated output:\n%s=%s\n%s=%s",
			language, "key1", fromFirstFileValue,
			"key2", fromSecondFileValue)
	})

	// Using in inside your views:
	templates := blocks.New(viewsPath)
	// It accepts a language code as its first argument: {{ tr "el-GR" "hi" "iris"}}
	templates.AddFunc("tr", func(lang, key string, args ...any) string {
		if locale, ok := locales.GetLocale(lang); ok {
			return locale.GetMessage(key, args...)
		}

		return "[INVALID LOCALE] " + lang
	})
	templates.Load()
	app.UseRouter(view.Handler(templates))

	app.Get("/templates", func(ctx iris.Context) {
		if err := view.Render(ctx, "index.html", "", iris.Map{
			"tr": func(key string, args ...any) string { // {{call .tr "hi" "iris"}}
				return i18n.GetMessage(ctx, key, args...)
			},
			"trUnsafe": func(key string, args ...any) template.HTML {
				return template.HTML(i18n.GetMessage(ctx, key, args...))
			}, // tr function as data with the Iris Context.
		}); err != nil {
			ctx.HTML(fmt.Sprintf("<h3>%s</h3>", err.Error()))
			return
		}
	})
	//

	// go to http://localhost:8080/el-gr/some-path
	// ^ (by path prefix)
	//
	// or http://el.mydomain.com:8080/some-path
	// ^ (by subdomain - test locally with the hosts file)
	//
	// or http://localhost:8080/zh-CN/templates
	// ^ (by path prefix with uppercase)
	//
	// or http://localhost:8080/some-path?lang=el-GR
	// ^ (by url parameter)
	//
	// or http://localhost:8080 (default is en-US)
	// or http://localhost:8080/?lang=zh-CN
	//
	// go to http://localhost:8080/other?lang=el-GR
	// or http://localhost:8080/other (default is en-US)
	// or http://localhost:8080/other?lang=en-US
	//
	// or use cookies to set the language.
	return app
}
