package main

import (
	"github.com/kataras/iris/v12"

	"github.com/iris-contrib/middleware/newrelic"
)

func main() {
	app := iris.New()
	config := newrelic.Config("APP_SERVER_NAME", "NEWRELIC_LICENSE_KEY")
	config.Enabled = true
	m, err := newrelic.New(config)
	if err != nil {
		app.Logger().Fatal(err)
	}
	app.Use(m.ServeHTTP)

	app.Get("/", func(ctx iris.Context) {
		ctx.Writef("success!\n")
	})

	app.Listen(":8080")
}
