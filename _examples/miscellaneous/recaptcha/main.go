package main

import (
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/recaptcha"
)

// keys should be obtained by https://www.google.com/recaptcha
const (
	recaptchaPublic = ""
	recaptchaSecret = ""
)

func showRecaptchaForm(ctx iris.Context, path string) {
	ctx.HTML(recaptcha.GetFormHTML(recaptchaPublic, path))
}

func main() {
	app := iris.New()

	// On both Get and Post on this example, so you can easly
	// use a single route to show a form and the main subject if recaptcha's validation result succeed.
	app.HandleMany("GET POST", "/", func(ctx iris.Context) {
		if ctx.Method() == iris.MethodGet {
			showRecaptchaForm(ctx, "/")
			return
		}

		result := recaptcha.SiteFerify(ctx, recaptchaSecret)
		if !result.Success {
			/* redirect here if u want or do nothing */
			ctx.HTML("<b> failed please try again </b>")
			return
		}

		ctx.Writef("succeed.")
	})

	app.Run(iris.Addr(":8080"))
}
