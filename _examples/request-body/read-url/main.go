// package main contains an example on how to use the ReadURL,
// same way you can do the ReadQuery, ReadParams, ReadJSON, ReadProtobuf and e.t.c.
package main

import (
	"github.com/kataras/iris/v12"
)

type myURL struct {
	Name string   `url:"name"` // or `param:"name"`
	Age  int      `url:"age"`  // >> >>
	Tail []string `url:"tail"` // >> >>
}

func main() {
	app := newApp()

	// http://localhost:8080/iris/web/framework?name=kataras&age=27
	// myURL: main.myURL{Name:"kataras", Age:27, Tail:[]string{"iris", "web", "framework"}}
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	app.Get("/{tail:path}", func(ctx iris.Context) {
		var u myURL
		// ReadURL is a shortcut of ReadParams + ReadQuery.
		if err := ctx.ReadURL(&u); err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		ctx.Writef("myURL: %#v", u)
	})

	return app
}
