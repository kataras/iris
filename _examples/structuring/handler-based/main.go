package main

import (
	"github.com/kataras/iris/_examples/structuring/handler-based/bootstrap"
	"github.com/kataras/iris/_examples/structuring/handler-based/middleware/identity"
	"github.com/kataras/iris/_examples/structuring/handler-based/routes"
)

var app = bootstrap.New("Awesome App", "kataras2006@hotmail.com",
	identity.Configure,
	routes.Configure,
)

func init() {
	app.Bootstrap()
}

func main() {
	app.Listen(":8080")
}
