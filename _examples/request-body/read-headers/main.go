// package main contains an example on how to use the ReadHeaders,
// same way you can do the ReadQuery, ReadJSON, ReadProtobuf and e.t.c.
package main

import (
	"github.com/kataras/iris/v12"
)

type myHeaders struct {
	RequestID      string `header:"X-Request-Id,required"`
	Authentication string `header:"Authentication,required"`
}

func main() {
	app := newApp()

	// http://localhost:8080
	/*
		myHeaders: main.myHeaders{
			RequestID: "373713f0-6b4b-42ea-ab9f-e2e04bc38e73",
			Authentication: "Bearer my-token",
		}
	*/
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		var hs myHeaders
		if err := ctx.ReadHeaders(&hs); err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		ctx.Writef("myHeaders: %#v", hs)
	})

	return app
}
