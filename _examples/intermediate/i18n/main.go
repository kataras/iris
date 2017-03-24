package main

import (
	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/httprouter"
	"gopkg.in/kataras/iris.v6/middleware/i18n"
)

func main() {
	app := iris.New()
	app.Adapt(iris.DevLogger()) // adapt a simple internal logger to print any errors
	app.Adapt(httprouter.New()) // adapt a router, you can use gorillamux too

	app.Use(i18n.New(i18n.Config{
		Default:      "en-US",
		URLParameter: "lang",
		Languages: map[string]string{
			"en-US": "./locales/locale_en-US.ini",
			"el-GR": "./locales/locale_el-GR.ini",
			"zh-CN": "./locales/locale_zh-CN.ini"}}))

	app.Get("/", func(ctx *iris.Context) {

		// it tries to find the language by:
		// ctx.Get("language") , that should be setted on other middleware before the i18n middleware*
		// if that was empty then
		// it tries to find from the URLParameter setted on the configuration
		// if not found then
		// it tries to find the language by the "lang" cookie
		// if didn't found then it it set to the Default setted on the configuration

		// hi is the key, 'kataras' is the %s on the .ini file
		// the second parameter is optional

		// hi := ctx.Translate("hi", "kataras")
		// or:
		hi := i18n.Translate(ctx, "hi", "kataras")

		language := ctx.Get(iris.TranslateLanguageContextKey) // language is the language key, example 'en-US'

		// The first succeed language found saved at the cookie with name ("language"),
		//  you can change that by changing the value of the:  iris.TranslateLanguageContextKey
		ctx.Writef("From the language %s translated output: %s", language, hi)
	})

	// go to http://localhost:8080/?lang=el-GR
	// or http://localhost:8080
	// or http://localhost:8080/?lang=zh-CN
	app.Listen(":8080")

}
