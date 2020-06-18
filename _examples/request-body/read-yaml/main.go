package main

import (
	"github.com/kataras/iris/v12"
)

func newApp() *iris.Application {
	app := iris.New()
	app.Post("/", handler)

	return app
}

// simple yaml stuff, read more at https://yaml.org/start.html
type product struct {
	Invoice  int     `yaml:"invoice"`
	Tax      float32 `yaml:"tax"`
	Total    float32 `yaml:"total"`
	Comments string  `yaml:"comments"`
}

func handler(ctx iris.Context) {
	var p product
	if err := ctx.ReadYAML(&p); err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	ctx.Writef("Received: %#+v", p)
}

func main() {
	app := newApp()
	app.Listen(":8080")
}
