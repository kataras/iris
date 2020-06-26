package main

import (
	"time"

	"github.com/kataras/iris/v12"
)

func main() {
	startup := time.Now()

	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		s := startup.Format(ctx.Application().ConfigurationReadOnly().GetTimeFormat())
		ctx.Writef("This server started at: %s\n", s)
	})

	// This option allows linear scaling server performance on multi-CPU servers.
	// See https://www.nginx.com/blog/socket-sharding-nginx-release-1-9-1/ for details.
	app.Listen(":8080", iris.WithSocketSharding)
}
