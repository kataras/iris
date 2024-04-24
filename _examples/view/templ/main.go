package main

import (
	"github.com/kataras/iris/v12"
)

// $ go install github.com/a-h/templ/cmd/templ@latest
// $ templ generate
// $ go run .
func main() {
	component := hello("Makis")

	app := iris.New()
	app.Get("/", iris.Component(component))

	app.Listen(":8080")
}
