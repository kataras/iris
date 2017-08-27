// +build linux darwin dragonfly freebsd netbsd openbsd rumprun

package main

import (
	// Package tcplisten provides customizable TCP net.Listener with various
	// performance-related options:
	//
	//   - SO_REUSEPORT. This option allows linear scaling server performance
	//     on multi-CPU servers.
	//     See https://www.nginx.com/blog/socket-sharding-nginx-release-1-9-1/ for details.
	//
	//   - TCP_DEFER_ACCEPT. This option expects the server reads from the accepted
	//     connection before writing to them.
	//
	//   - TCP_FASTOPEN. See https://lwn.net/Articles/508865/ for details.
	"github.com/valyala/tcplisten"

	"github.com/kataras/iris"
)

// $ go get github.com/valyala/tcplisten
// $ go run main.go

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<b>Hello World!</b>")
	})

	listenerCfg := tcplisten.Config{
		ReusePort:   true,
		DeferAccept: true,
		FastOpen:    true,
	}

	l, err := listenerCfg.NewListener("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	app.Run(iris.Listener(l))
}
