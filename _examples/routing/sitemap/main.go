package main

import (
	"time"

	"github.com/kataras/iris/v12"
)

const startURL = "http://localhost:8080"

func main() {
	app := newApp()

	// http://localhost:8080/sitemap.xml
	// Lists only online GET static routes.
	//
	// Reference: https://www.sitemaps.org/protocol.html
	app.Listen(":8080", iris.WithSitemap(startURL))
}

func newApp() *iris.Application {
	app := iris.New()
	app.Logger().SetLevel("debug")

	lastModified, _ := time.Parse("2006-01-02T15:04:05-07:00", "2019-12-13T21:50:33+02:00")
	app.Get("/home", handler).SetLastMod(lastModified).SetChangeFreq("hourly").SetPriority(1)
	app.Get("/articles", handler).SetChangeFreq("daily")
	app.Get("/path1", handler)
	app.Get("/path2", handler)

	app.Post("/this-should-not-be-listed", handler)
	app.Get("/this/{myparam}/should/not/be/listed", handler)
	app.Get("/this-should-not-be-listed-offline", handler).SetStatusOffline()

	// These should be excluded as well
	app.Get("/about", handler).ExcludeSitemap()
	app.Get("/offline", handler).SetStatusOffline()

	return app
}

func handler(ctx iris.Context) { ctx.WriteString(ctx.Path()) }
