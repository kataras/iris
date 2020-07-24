package main

import (
	"github.com/kataras/iris/v12"
)

const (
	addr      = "example.com:80"
	subdomain = "v1"
)

func newApp() *iris.Application {
	app := iris.New()
	app.Favicon("./assets/favicon.ico")

	v1 := app.Subdomain(subdomain)
	v1.HandleDir("/", iris.Dir("./assets"))

	// http://v1.example.com
	// http://v1.example.com/css/main.css
	// http://v1.example.com/js/jquery-2.1.1.js
	// http://v1.example.com/favicon.ico
	return app
}

func main() {
	app := newApp()
	app.Listen(addr)
}
