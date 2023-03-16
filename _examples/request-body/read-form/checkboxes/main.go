package main

import "github.com/kataras/iris/v12"

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./templates", ".html"))

	app.Get("/", showForm)
	app.Post("/", handleForm)

	app.Listen(":8080")
}

func showForm(ctx iris.Context) {
	if err := ctx.View("form.html"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}

type formExample struct {
	Colors []string `form:"colors[]"` // or just colors, it'll work as expected.
}

func handleForm(ctx iris.Context) {
	var form formExample
	err := ctx.ReadForm(&form)
	if err != nil {
		ctx.StopWithError(iris.StatusBadRequest, err)
		return
	}

	ctx.JSON(iris.Map{"Colors": form.Colors})
}
