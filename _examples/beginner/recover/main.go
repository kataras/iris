package main

import (
	"github.com/cdren/iris"
	"github.com/cdren/iris/context"

	"github.com/cdren/iris/middleware/recover"
)

func main() {
	app := iris.New()
	// use this recover(y) middleware
	app.Use(recover.New())

	i := 0
	// let's simmilate a panic every next request
	app.Get("/", func(ctx context.Context) {
		i++
		if i%2 == 0 {
			panic("a panic here")
		}
		ctx.Writef("Hello, refresh one time more to get panic!")
	})

	// http://localhost:8080, refresh it 5-6 times.
	app.Run(iris.Addr(":8080"))
}

// Note:
// app := iris.Default() instead of iris.New() makes use of the recovery middleware automatically.
