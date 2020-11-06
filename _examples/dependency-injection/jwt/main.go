package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
)

func main() {
	app := iris.New()
	app.ConfigureContainer(register)

	// http://localhost:8080/authenticate
	// http://localhost:8080/restricted (Header: Authorization = Bearer $token)
	app.Listen(":8080")
}

var secret = []byte("secret")

func register(api *iris.APIContainer) {
	api.RegisterDependency(func(ctx iris.Context) (claims userClaims) {
		/* Using the middleware:
		if ctx.Proceed(verify) {
			// ^ the "verify" middleware will stop the execution if it's failed to verify the request.
			// Map the input parameter of "restricted" function with the claims.
			return jwt.Get(ctx).(*userClaims)
		}*/
		token := jwt.FromHeader(ctx)
		if token == "" {
			ctx.StopWithError(iris.StatusUnauthorized, jwt.ErrMissing)
			return
		}

		verifiedToken, err := jwt.Verify(jwt.HS256, secret, []byte(token))
		if err != nil {
			ctx.StopWithError(iris.StatusUnauthorized, err)
			return
		}

		verifiedToken.Claims(&claims)
		return
	})

	api.Get("/authenticate", writeToken)
	api.Get("/restricted", restrictedPage)
}

type userClaims struct {
	Username string `json:"username"`
}

func writeToken(ctx iris.Context) {
	claims := userClaims{
		Username: "kataras",
	}

	token, err := jwt.Sign(jwt.HS256, secret, claims, jwt.MaxAge(1*time.Minute))
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	ctx.Write(token)
}

func restrictedPage(claims userClaims) string {
	// userClaims.Username: kataras
	return "userClaims.Username: " + claims.Username
}
