// package main contains an example on how to use the ReadForm, but with the same way you can do the ReadJSON & ReadJSON
package main

import (
	"github.com/kataras/iris"
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
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.WriteString(err.Error())
		}

		ctx.Writef("MyType: %#v", t)
	})

	// http://localhost:8080?name=iris&age=3
	app.Run(iris.Addr(":8080"))
}
