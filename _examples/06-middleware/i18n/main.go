// Internationalization: one message catalog, one language per request.
//
// i18n.New() creates the catalog and Load fills it from a glob pattern. The
// language of each file is deduced from its name (locales/en-US.yml) or from a
// directory named after it; the first language listed is the default and the
// fallback. INI, JSON, TOML and YAML files are supported.
//
// i18n.NewHandler installs the per-request detection. It checks its sources in
// a fixed order - a server-set language, ExtractFunc, URLParameter, Cookie,
// Subdomain, then the Accept-Language header, which is always consulted - and
// falls back to the default locale. Matching is lenient: "el", "el-gr" and
// "el-GR" all resolve to the registered el-GR.
//
// Handlers translate with i18n.GetMessage(ctx, key, args...) or through the
// locale returned by i18n.GetLocale(ctx).
//
// Learn: load locale files, detect the language of each request, and translate
// plain, plural and variable messages.
// Before: ../cache
//
// Run from this directory, so ./locales resolves.
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl http://localhost:8080/                           # [en-US] hello, iris
//	curl -H "Accept-Language: el" http://localhost:8080/   # [el-GR] γεια, iris
//	curl "http://localhost:8080/days/1?lang=el-GR"        # Έχεις μία ελεύθερη μέρα.
//	curl "http://localhost:8080/days/5?lang=el-GR"        # Έχεις 5 ελεύθερες μέρες.
//	curl http://localhost:8080/late/1                     # You are 1 minute late.
//	curl http://localhost:8080/late/10                    # You are 10 minutes late.
//
// Next: ../../07-dependency-injection/constructors
package main

import (
	"log"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/middleware/i18n"
)

func main() {
	newApp().Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	locales := i18n.New()
	// One glob, both files. LoadFS(embed.FS, ...) compiles them into the
	// binary instead, LoadKV loads them from a map.
	if err := locales.Load("./locales/*", "en-US", "el-GR"); err != nil {
		log.Fatal(err)
	}

	// Every detection source is opt-in except Accept-Language. Cookie: "lang"
	// and Subdomain: true are the other two; i18n.NewRouter, registered with
	// UseRouter, additionally strips language path prefixes such as
	// /el-GR/days/5 before routing.
	app.Use(i18n.NewHandler(locales, i18n.HandlerOptions{
		URLParameter: "lang",
	}))

	app.Get("/", func(ctx iris.Context) {
		// The matched locale: Language(), Tag(), Index() and GetMessage.
		locale := i18n.GetLocale(ctx)
		// Plain messages are rendered with printf verbs, indexed ones (%[1]s)
		// included, so a translation can move the value elsewhere.
		ctx.Writef("[%s] %s", locale.Language(), locale.GetMessage("hi", "iris"))
	})

	// A plural message. The count is the first argument: it selects the form
	// and it also feeds that form's printf verbs.
	app.Get("/days/{count:int}", func(ctx iris.Context) {
		count := ctx.Params().GetInt("count")
		// i18n.GetMessage is the shortcut for GetLocale(ctx).GetMessage.
		ctx.WriteString(i18n.GetMessage(ctx, "FreeDay", count))
	})

	// ${Minutes} is a Vars entry with plural forms of its own, selected by the
	// same count: "1 minute late", "10 minutes late".
	app.Get("/late/{count:int}", func(ctx iris.Context) {
		count := ctx.Params().GetInt("count")
		ctx.WriteString(i18n.GetMessage(ctx, "YouLate", count))
	})

	// A message containing {{ and }} is parsed as a Go text/template instead,
	// with the first argument as its data: i18n.GetMessage(ctx, "welcome",
	// iris.Map{"name": "Makis"}) renders "Welcome back, {{.name}}!".

	return app
}
