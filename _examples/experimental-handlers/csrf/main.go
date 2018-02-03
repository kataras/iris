// This middleware provides Cross-Site Request Forgery
// protection.
//
// It securely generates a masked (unique-per-request) token that
// can be embedded in the HTTP response (e.g. form field or HTTP header).
// The original (unmasked) token is stored in the session, which is inaccessible
// by an attacker (provided you are using HTTPS). Subsequent requests are
// expected to include this token, which is compared against the session token.
// Requests that do not provide a matching token are served with a HTTP 403
// 'Forbidden' error response.
package main

// $ go get -u github.com/iris-contrib/middleware/...

import (
	"github.com/kataras/iris"

	"github.com/iris-contrib/middleware/csrf"
)

func main() {
	app := iris.New()
	app.RegisterView(iris.HTML("./views", ".html"))
	// Note that the authentication key provided should be 32 bytes
	// long and persist across application restarts.
	protect := csrf.Protect([]byte("9AB0F421E53A477C084477AEA06096F5"),
		csrf.Secure(false)) // Defaults to true, but pass `false` while no https (devmode).

	users := app.Party("/user", protect)
	{
		users.Get("/signup", getSignupForm)
		// // POST requests without a valid token will return a HTTP 403 Forbidden.
		users.Post("/signup", postSignupForm)
	}

	// GET: http://localhost:8080/user/signup
	// POST: http://localhost:8080/user/signup
	app.Run(iris.Addr(":8080"))
}

func getSignupForm(ctx iris.Context) {
	// views/user/signup.html just needs a {{ .csrfField }} template tag for
	// csrf.TemplateField to inject the CSRF token into. Easy!
	ctx.ViewData(csrf.TemplateTag, csrf.TemplateField(ctx))
	ctx.View("user/signup.html")

	// We could also retrieve the token directly from csrf.Token(ctx) and
	// set it in the request header - ctx.GetHeader("X-CSRF-Token", token)
	// This is useful if you're sending JSON to clients or a front-end JavaScript
	// framework.
}

func postSignupForm(ctx iris.Context) {
	ctx.Writef("You're welcome mate!")
}
