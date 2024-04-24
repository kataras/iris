package main

import (
	"os"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/hcaptcha"
)

// Get the following values from: https://dashboard.hcaptcha.com
// Also, check: https://docs.hcaptcha.com/#localdev to test on local environment.
var (
	siteKey   = os.Getenv("HCAPTCHA-SITE-KEY")
	secretKey = os.Getenv("HCAPTCHA-SECRET-KEY")
)

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./templates", ".html"))

	hCaptcha := hcaptcha.New(secretKey)
	app.Get("/register", registerForm)
	app.Post("/register", hCaptcha, register) // See `hcaptcha.SiteVerify` for manual validation too.

	app.Logger().Infof("SiteKey = %s\tSecretKey = %s",
		siteKey, secretKey)

	// GET: http://yourdomain.com/register
	app.Listen(":80")
}

func register(ctx iris.Context) {
	hcaptchaResp, ok := hcaptcha.Get(ctx)
	if !ok {
		ctx.StatusCode(iris.StatusUnauthorized)
		ctx.WriteString("Are you a bot?")
		return
	}

	ctx.Writef("Register action here...action was asked by a Human.\nResponse value is: %#+v", hcaptchaResp)
}

func registerForm(ctx iris.Context) {
	ctx.ViewData("SiteKey", siteKey)
	if err := ctx.View("register_form.html"); err != nil {
		ctx.HTML("<h3>%s</h3>", err.Error())
		return
	}
}
