package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
)

func main() {
	app := iris.New()
	// With AES-GCM (128) encryption:
	// j := jwt.HMAC(15*time.Minute, "secret", "itsa16bytesecret")
	// Without extra encryption, just the sign key:
	j := jwt.HMAC(15*time.Minute, "secret")

	app.Get("/", generateToken(j))
	app.Get("/protected", j.VerifyMap(), protected)

	app.Listen(":8080")
}

func generateToken(j *jwt.JWT) iris.Handler {
	return func(ctx iris.Context) {
		token, err := j.Token(iris.Map{
			"foo": "bar",
		})
		if err != nil {
			ctx.StopWithStatus(iris.StatusInternalServerError)
			return
		}

		ctx.HTML(`Token: ` + token + `<br/><br/>
		<a href="/protected?token=` + token + `">/secured?token=` + token + `</a>`)
	}
}

func protected(ctx iris.Context) {
	ctx.Writef("This is an authenticated request.\n\n")

	claims := jwt.Get(ctx).(iris.Map)

	ctx.Writef("foo=%s\n", claims["foo"])
}
