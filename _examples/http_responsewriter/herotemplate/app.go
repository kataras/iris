package main

import (
	"bytes"

	"github.com/kataras/iris/_examples/http_responsewriter/herotemplate/template"

	"github.com/kataras/iris"
)

// $ go get -u github.com/shiyanhui/hero/hero
// $ go run app.go
//
// Read more at https://github.com/shiyanhui/hero/hero

func main() {

	app := iris.New()

	app.Get("/users", func(ctx iris.Context) {
		ctx.Gzip(true)
		ctx.ContentType("text/html")

		var userList = []string{
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

		// using an io.Writer for automatic buffer management (i.e. hero built-in buffer pool),
		// iris context implements the io.Writer by its ResponseWriter
		// which is an enhanced version of the standard http.ResponseWriter
		// but still 100% compatible, GzipResponseWriter too:
		// _, err := template.UserListToWriter(userList, ctx.GzipResponseWriter())
		buffer := new(bytes.Buffer)
		template.UserList(userList, buffer)

		_, err := ctx.Write(buffer.Bytes())
		if err != nil {
			ctx.StatusCode(iris.StatusInternalServerError)
			ctx.WriteString(err.Error())
		}
	})

	app.Run(iris.Addr(":8080"))
}
