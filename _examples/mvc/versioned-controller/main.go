package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

// Optional deprecated X-API-XXX headers for version 1.
var opts = mvc.DeprecationOptions{
	WarnMessage:     "deprecated, see <this link>",
	DeprecationDate: time.Now().UTC(),
	DeprecationInfo: "a bigger version is available, see <this link> for more information",
}

func main() {
	app := newApp()

	// See main_test.go for request examples.
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	dataRouter := app.Party("/data")
	{
		m := mvc.New(dataRouter)

		m.Handle(new(v1Controller), mvc.Version("1.0.0"), mvc.Deprecated(opts))
		m.Handle(new(v2Controller), mvc.Version("2.3.0"))
		m.Handle(new(v3Controller), mvc.Version(">=3.0.0 <4.0.0"))
		m.Handle(new(noVersionController)) // or if missing it will respond with 501 version not found.
	}

	return app
}

type v1Controller struct{}

func (c *v1Controller) Get() string {
	return "data (v1.x)"
}

type v2Controller struct{}

func (c *v2Controller) Get() string {
	return "data (v2.x)"
}

type v3Controller struct{}

func (c *v3Controller) Get() string {
	return "data (v3.x)"
}

type noVersionController struct{}

func (c *noVersionController) Get() string {
	return "data"
}
