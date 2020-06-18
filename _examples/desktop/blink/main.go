// +build windows

package main

import (
	"github.com/kataras/iris/v12"
	"github.com/raintean/blink"
)

const addr = "127.0.0.1:8080"

/*
	$ go build -ldflags -H=windowsgui -o myapp.exe
	$ ./myapp.exe # run the app
*/
func main() {
	go runServer()
	showAndWaitWindow()
}

func runServer() {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<h1> Hello Desktop</h1>")
	})
	app.Listen(addr)
}

func showAndWaitWindow() {
	blink.SetDebugMode(true)
	if err := blink.InitBlink(); err != nil {
		panic(err)
	}

	view := blink.NewWebView(false, 800, 600)
	view.LoadURL(addr)
	view.SetWindowTitle("My App")
	view.MoveToCenter()
	view.ShowWindow()

	<-view.Destroy
}
