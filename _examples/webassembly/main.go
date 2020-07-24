package main

import (
	"github.com/kataras/iris/v12"
)

/*
You need to build the hello.wasm first, download the go1.14 and execute the below command:
$ cd client && GOARCH=wasm GOOS=js /home/$yourname/go1.14/bin/go build -o hello.wasm hello_go114.go
*/

func main() {
	app := iris.New()

	// we could serve your assets like this the sake of the example,
	// never include the .go files there in production.
	app.HandleDir("/", iris.Dir("./client"))

	app.Get("/", func(ctx iris.Context) {
		// ctx.CompressWriter(true)
		ctx.ServeFile("./client/hello.html")
	})

	// visit http://localhost:8080
	// you should get an html output like this:
	// Hello, the current time is: 2018-07-09 05:54:12.564 +0000 UTC m=+0.003900161
	app.Listen(":8080")
}
