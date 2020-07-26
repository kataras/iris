// package main contains an example on how to use the ReadQuery,
// same way you can do the ReadJSON & ReadProtobuf and e.t.c.
package main

import (
	"github.com/kataras/iris/v12"
)

type MyType struct {
	Name string `url:"name"`
	Age  int    `url:"age"`
}

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		var t MyType
		err := ctx.ReadQuery(&t)
		if err != nil && !iris.IsErrPath(err) {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		ctx.Writef("MyType: %#v", t)
	})

	// http://localhost:8080?name=iris&age=3
	app.Listen(":8080")
}
