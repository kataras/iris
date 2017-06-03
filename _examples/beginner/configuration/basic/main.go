package main

import (
	"github.com/kataras/iris"
)

func main() {
	app := iris.New()

	// [...]

	// Good when you want to modify the whole configuration.
	app.Run(iris.Addr(":8080"), iris.WithConfiguration(iris.Configuration{ // default configuration:
		DisableBanner:                     false,
		DisableTray:                       false,
		DisableInterruptHandler:           false,
		DisablePathCorrection:             false,
		EnablePathEscape:                  false,
		FireMethodNotAllowed:              false,
		DisableBodyConsumptionOnUnmarshal: false,
		DisableAutoFireStatusCode:         false,
		TimeFormat:                        "Mon, 02 Jan 2006 15:04:05 GMT",
		Charset:                           "UTF-8",
	}))

	// or before run:
	// app.Configure(iris.WithConfiguration(...))
	// app.Run(iris.Addr(":8080"))
}
