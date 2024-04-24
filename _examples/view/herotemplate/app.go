package main

import (
	"bytes"

	"github.com/kataras/iris/v12/_examples/view/herotemplate/template"

	"github.com/kataras/iris/v12"
)

// $ go get -u github.com/shiyanhui/hero/hero
// $ go run app.go
//
// Read more at https://github.com/shiyanhui/hero/hero

func main() {
	app := iris.New()

	app.Get("/users", func(ctx iris.Context) {
		ctx.CompressWriter(true)
		ctx.ContentType("text/html")

		userList := []string{
			"Alice",
			"Bob",
			"Tom",
		}

		// Had better use buffer sync.Pool.
		// Hero(github.com/shiyanhui/hero/hero) exports GetBuffer and PutBuffer for this.
		//
		// buffer := hero.GetBuffer()
		// defer hero.PutBuffer(buffer)
		// buffer := new(bytes.Buffer)
		// template.UserList(userList, buffer)
		// ctx.Write(buffer.Bytes())

		// iris context implements the io.Writer:
		// _, err := template.UserListToWriter(userList, ctx)
		// OR:
		buffer := new(bytes.Buffer)
		template.UserList(userList, buffer)

		_, err := ctx.Write(buffer.Bytes())
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}
	})

	app.Listen(":8080")
}
