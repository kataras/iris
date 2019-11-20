package main

import (
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/i18n"
)

var i18nConfig = i18n.Config{
	Default: "en-US",
	Languages: map[string]string{
		"en-US": "./locales/locale_en-US.ini", // maps to en-US, en-us and en.
		"el-GR": "./locales/locale_el-GR.ini", // maps to el-GR, el-gr and el.
		"zh-CN": "./locales/locale_zh-CN.ini", // maps to zh-CN, zh-cn and zh.
	},
	// Optionals.
	Alternatives: map[string]string{ // optional.
		"english": "en-US", // now english maps to en-US
		"greek":   "el-GR", // and greek to el-GR
		"chinese": "zh-CN", // and chinese to zh-CN too.
	},
	URLParameter: "lang",
	Subdomain:    true,
	// Cookie: "lang",
	// SetCookie: false,
	// Indentifier: func(ctx iris.Context) string { return "zh-CN" },
}

func newApp() *iris.Application {
	app := iris.New()

	i18nMiddleware := i18n.NewI18n(i18nConfig)
	app.Use(i18nMiddleware.Handler())

	// See https://github.com/kataras/iris/issues/1369
	// if you want to enable this (SEO) feature (OPTIONAL).
	app.WrapRouter(i18nMiddleware.Wrapper())

	app.Get("/", func(ctx iris.Context) {
		// Ir tries to find the language by:
		// ctx.Values().GetString("language")
		// if that was empty then
		// it tries to find from the URLParameter set on the configuration
		// if not found then
		// it tries to find the language by the "language" cookie
		// if didn't found then it it set to the Default set on the configuration

		// hi is the key/word, 'iris' is the %s on the .ini file
		// the second parameter is optional

		hi := ctx.Translate("hi", "iris")

		// GetTranslateLanguageContextKey() == "language"
		language := ctx.Values().GetString(ctx.Application().ConfigurationReadOnly().GetTranslateLanguageContextKey())
		// return is form of 'en-US'

		// The first succeed language found saved at the cookie with name ("language"),
		// you can change that by changing the value of the:  iris.TranslateLanguageContextKey
		ctx.Writef("From the language %s translated output: %s", language, hi)
	})

	app.Get("/some-path", func(ctx iris.Context) {
		ctx.Writef("%s", ctx.Translate("hi", "iris"))
	})

	app.Get("/sitemap.xml", func(ctx iris.Context) {
		ctx.WriteString("sitemap")
	})

	// Note: It is highly recommended to use one and no more i18n middleware instances at a time,
	// the first one was already passed by `app.Use` above.
	// This middleware which registers on "/multi" route is here just for the shake of the example.
	multiLocale := i18n.New(i18n.Config{
		Default:      "en-US",
		URLParameter: "lang",
		Languages: map[string]string{
			"en-US": "./locales/locale_multi_first_en-US.ini, ./locales/locale_multi_second_en-US.ini",
			"el-GR": "./locales/locale_multi_first_el-GR.ini, ./locales/locale_multi_second_el-GR.ini",
		},
	})

	app.Get("/multi", multiLocale, func(ctx iris.Context) {
		language := ctx.Values().GetString(ctx.Application().ConfigurationReadOnly().GetTranslateLanguageContextKey())

		fromFirstFileValue := ctx.Translate("key1")
		fromSecondFileValue := ctx.Translate("key2")
		ctx.Writef("From the language: %s, translated output:\n%s=%s\n%s=%s",
			language, "key1", fromFirstFileValue,
			"key2", fromSecondFileValue)
	})

	// using in inside your templates:
	view := iris.HTML("./templates", ".html")
	app.RegisterView(view)

	app.Get("/templates", func(ctx iris.Context) {
		ctx.View("index.html", iris.Map{
			"tr":     ctx.Translate,     // word, arguments...
			"trLang": ctx.TranslateLang, // locale, word, arguments...
		})
		// it will return "hello, iris"
		// when {{call .tr "hi" "iris"}}
	})
	//

	return app
}

func main() {
	app := newApp()

	// go to http://localhost:8080/el-gr/some-path (by path prefix)
	// or http://el.mydomain.com8080/some-path (by subdomain - test locally with the hosts file)
	// or http://localhost:8080/zh-CN/templates (by path prefix with uppercase)
	// or http://localhost:8080/some-path?lang=el-GR (by url parameter)
	// or http://localhost:8080 (default is en-US)
	// or http://localhost:8080/?lang=zh-CN
	//
	// go to http://localhost:8080/multi?lang=el-GR
	// or http://localhost:8080/multi (default is en-US)
	// or http://localhost:8080/multi?lang=en-US
	//
	// or use cookies to set the language.
	//
	app.Run(iris.Addr(":8080"))
}
