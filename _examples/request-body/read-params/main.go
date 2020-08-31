// package main contains an example on how to use the ReadParams,
// same way you can do the ReadQuery, ReadJSON, ReadProtobuf and e.t.c.
package main

import (
	"github.com/kataras/iris/v12"
)

type myParams struct {
	Name string   `param:"name"`
	Age  int      `param:"age"`
	Tail []string `param:"tail"`
}

func main() {
	app := newApp()

	// http://localhost:8080/kataras/27/iris/web/framework
	// myParams: main.myParams{Name:"kataras", Age:27, Tail:[]string{"iris", "web", "framework"}}
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	app.Get("/{name}/{age:int}/{tail:path}", func(ctx iris.Context) {
		var p myParams
		if err := ctx.ReadParams(&p); err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		ctx.Writef("myParams: %#v", p)
	})

	return app
}
