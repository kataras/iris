package main

import (
	"bytes"
	"log"

	"github.com/kataras/iris/_examples/http_responsewriter/hero/template"

	"github.com/kataras/iris"
)

// $ go get -u github.com/shiyanhui/hero/hero
// $ go run app.go
//
// Read more at https://github.com/shiyanhui/hero/hero
func main() {

	app := iris.New()

	app.Get("/users", func(ctx iris.Context) {
		var userList = []string{
			"Alice",
			"Bob",
			"Tom",
		}

		// Had better use buffer sync.Pool.
		// Hero exports GetBuffer and PutBuffer for this.
		//
		// buffer := hero.GetBuffer()
		// defer hero.PutBuffer(buffer)
		buffer := new(bytes.Buffer)
		template.UserList(userList, buffer)

		if _, err := ctx.Write(buffer.Bytes()); err != nil {
			log.Printf("ERR: %s\n", err)
		}
	})

	app.Get("/users2", func(ctx iris.Context) {
		var userList = []string{
			"Alice",
			"Bob",
			"Tom",
		}

		// using an io.Writer for automatic buffer management (i.e. hero built-in buffer pool),
		// iris context implements the io.Writer by its ResponseWriter
		// which is an enhanced version of the standard http.ResponseWriter
		// but still 100% compatible.
		template.UserListToWriter(userList, ctx)
	})

	app.Run(iris.Addr(":8080"))
}
