package main

import (
	"fmt"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"

	"github.com/kataras/iris/middleware/recaptcha"
)

// publicDataSiteKey and secretKey and should be obtained by https://www.google.com/recaptcha.
const (
	publicDataSiteKey = ""
	secretKey         = ""
)

func main() {
	app := iris.New()

	r := recaptcha.New(secretKey)

	app.Get("/comment", showRecaptchaForm)

	// pass the middleware before the main handler or use the `recaptcha.SiteVerify`.
	app.Post("/comment", r, postComment)

	app.Run(iris.Addr(":8080"))
}

var htmlForm = `<form action="/comment" method="POST">
	    <script src="https://www.google.com/recaptcha/api.js"></script>
		<div class="g-recaptcha" data-sitekey="%s"></div>
    	<input type="submit" name="button" value="Verify">
</form>`

func showRecaptchaForm(ctx context.Context) {
	contents := fmt.Sprintf(htmlForm, publicDataSiteKey)
	ctx.HTML(contents)
}

func postComment(ctx context.Context) {
	// [...]
	ctx.JSON(context.Map{"success": true})
}
