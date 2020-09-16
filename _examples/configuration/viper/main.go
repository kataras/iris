package main

import (
	"fmt"

	"app/config"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.TextYAML(config.C)
	})

	addr := fmt.Sprintf("%s:%d", config.C.Addr.Internal.IP, config.C.Addr.Internal.Plain)
	app.Listen(addr, iris.WithConfiguration(config.C.Iris))
}
