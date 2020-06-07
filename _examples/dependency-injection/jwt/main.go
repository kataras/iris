package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
)

func main() {
	app := iris.New()
	app.ConfigureContainer(register)

	app.Listen(":8080")
}

func register(api *iris.APIContainer) {
	j := jwt.HMAC(15*time.Minute, "secret", "secretforencrypt")

	api.RegisterDependency(func(ctx iris.Context) (claims userClaims) {
		if err := j.VerifyToken(ctx, &claims); err != nil {
			ctx.StopWithError(iris.StatusUnauthorized, err)
			return
		}

		return
	})

	api.Get("/authenticate", writeToken(j))
	api.Get("/restricted", restrictedPage)
}

type userClaims struct {
	jwt.Claims
	Username string
}

func writeToken(j *jwt.JWT) iris.Handler {
	return func(ctx iris.Context) {
		j.WriteToken(ctx, userClaims{
			Claims:   j.Expiry(jwt.Claims{Issuer: "an-issuer"}),
			Username: "kataras",
		})
	}
}

func restrictedPage(claims userClaims) string {
	// userClaims.Username: kataras
	return "userClaims.Username: " + claims.Username
}
