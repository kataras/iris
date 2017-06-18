// package main contains an example on how to use the ReadForm, but with the same way you can do the ReadJSON & ReadJSON
package main

import (
	"github.com/cdren/iris"
	"github.com/cdren/iris/context"
	"github.com/cdren/iris/view"
)

type Visitor struct {
	Username string
	Mail     string
	Data     []string `form:"mydata"`
}

func main() {
	app := iris.New()

	// set the view html template engine
	app.AttachView(view.HTML("./templates", ".html").Reload(true))

	app.Get("/", func(ctx context.Context) {
		if err := ctx.View("form.html"); err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.WriteString(err.Error())
		}
	})

	app.Post("/form_action", func(ctx context.Context) {
		visitor := Visitor{}
		err := ctx.ReadForm(&visitor)
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.WriteString(err.Error())
		}

		ctx.Writef("Visitor: %#v", visitor)
	})

	app.Run(iris.Addr(":8080"))
}
