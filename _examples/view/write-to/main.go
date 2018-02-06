package main

import (
	"os"

	"github.com/kataras/iris"
)

type mailData struct {
	Title    string
	Body     string
	RefTitle string
	RefLink  string
}

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")
	app.RegisterView(iris.HTML("./views", ".html"))

	// you need to call `app.Build` manually before using the `app.View` func,
	// so templates are built in that state.
	app.Build()

	// Or a string-buffered writer to use its body to send an e-mail
	// for sending e-mails you can use the https://github.com/kataras/go-mailer
	// or any other third-party package you like.
	//
	// The template's parsed result will be written to that writer.
	writer := os.Stdout
	err := app.View(writer, "email/simple.html", "shared/email.html", mailData{
		Title:    "This is my e-mail title",
		Body:     "This is my e-mail body",
		RefTitle: "Iris web framework",
		RefLink:  "https://iris-go.com",
	})

	if err != nil {
		app.Logger().Errorf("error from app.View: %v", err)
	}

	app.Run(iris.Addr(":8080"))
}
