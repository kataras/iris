package main

import (
	"github.com/kataras/iris/v12"
	"github.com/zserge/lorca"
)

const addr = "127.0.0.1:8080"

/*
	$ go build -ldflags="-H windowsgui" -o myapp.exe # build for windows
	$ ./myapp.exe # run
*/
func main() {
	go runServer()
	showAndWaitWindow()
}

func runServer() {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<head><title>My App</title></head><body><h1>Hello Desktop</h1></body>")
	})
	app.Listen(addr)
}

func showAndWaitWindow() {
	webview, err := lorca.New("http://"+addr, "", 800, 600)
	if err != nil {
		panic(err)
	}
	defer webview.Close()

	// webview.SetBounds(lorca.Bounds{
	// 	WindowState: lorca.WindowStateFullscreen,
	// })

	// Wait for the browser window to be closed
	<-webview.Done()
}
