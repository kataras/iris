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
	// http://localhost:8080/restricted
	app.Listen(":8080")
}

var (
	secret = []byte("secret")
	signer = jwt.NewSigner(jwt.HS256, secret, 15*time.Minute)
	verify = jwt.NewVerifier(jwt.HS256, secret, jwt.Expected{Issuer: "myapp"}).Verify(func() interface{} {
		return new(userClaims)
	})
)

func register(api *iris.APIContainer) {
	// To register the middleware in the whole api container:
	// api.Use(verify)
	// Otherwise, protect routes when userClaims is expected on the functions input
	// by calling the middleware manually, see below.
	api.RegisterDependency(func(ctx iris.Context) (claims *userClaims) {
		if ctx.Proceed(verify) { // the "verify" middleware will stop the execution if it's failed to verify the request.
			// Map the input parameter of "restricted" function with the claims.
			return jwt.Get(ctx).(*userClaims)
		}

		return nil
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
	standardClaims := jwt.Claims{
		Issuer: "myapp",
	}

	token, err := signer.Sign(claims, standardClaims)
	if err != nil {
		ctx.StopWithError(iris.StatusInternalServerError, err)
		return
	}

	ctx.Write(token)
}

func restrictedPage(claims *userClaims) string {
	// userClaims.Username: kataras
	return "userClaims.Username: " + claims.Username
}
