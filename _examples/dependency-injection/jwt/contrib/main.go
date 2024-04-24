package main

import (
	"github.com/kataras/iris/v12"

	"github.com/iris-contrib/middleware/jwt"
)

var secret = []byte("My Secret Key")

func main() {
	app := iris.New()
	app.ConfigureContainer(register)

	app.Listen(":8080")
}

func register(api *iris.APIContainer) {
	j := jwt.New(jwt.Config{
		// Extract by "token" url parameter.
		Extractor: jwt.FromFirst(jwt.FromParameter("token"), jwt.FromAuthHeader),
		ValidationKeyGetter: func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		},
		SigningMethod: jwt.SigningMethodHS256,
	})

	api.Get("/authenticate", writeToken)
	// This works as usually:
	api.Get("/restricted", j.Serve, restrictedPage)

	// You can also bind the *jwt.Token (see `verifiedWithBindedTokenPage`)
	// by registering a *jwt.Token dependency.
	//
	// api.RegisterDependency(func(ctx iris.Context) *jwt.Token {
	// 	if err := j.CheckJWT(ctx); err != nil {
	// 		ctx.StopWithStatus(iris.StatusUnauthorized)
	// 		return nil
	// 	}
	//
	// 	token := j.Get(ctx)
	// 	return token
	// })
	// ^ You can do the same with MVC too, as the container is shared and works
	// the same way in both functions-as-handlers and structs-as-controllers.
	//
	// api.Get("/", restrictedPageWithBindedTokenPage)
}

func writeToken() string {
	token := jwt.NewTokenWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"foo": "bar",
	})

	tokenString, _ := token.SignedString(secret)
	return tokenString
}

func restrictedPage() string {
	return "This page can only be seen by verified clients"
}

func restrictedPageWithBindedTokenPage(token *jwt.Token) string {
	// Token[foo] value: bar
	return "Token[foo] value: " + token.Claims.(jwt.MapClaims)["foo"].(string)
}
