// Package main shows how you can add middleware to an mvc Application, simply
// by using its `Router` which is a sub router(an iris.Party) of the main iris app.
package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/cache"
	"github.com/kataras/iris/v12/mvc"
)

var cacheHandler = cache.Handler(10 * time.Second)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	mvc.Configure(app, configure)

	// http://localhost:8080
	// http://localhost:8080/other
	//
	// refresh every 10 seconds and you'll see different time output.
	app.Listen(":8080")
}

func configure(m *mvc.Application) {
	m.Router.Use(cacheHandler)
	m.Handle(&exampleController{
		timeFormat: "Mon, Jan 02 2006 15:04:05",
	} /* ,mvc.IgnoreEmbedded --- Can be used to ignore any embedded struct method handlers */)
}

type exampleController struct {
	timeFormat string
}

func (c *exampleController) Get() string {
	now := time.Now().Format(c.timeFormat)
	return "last time executed without cache: " + now
}

func (c *exampleController) GetOther() string {
	now := time.Now().Format(c.timeFormat)
	return "/other: " + now
}
