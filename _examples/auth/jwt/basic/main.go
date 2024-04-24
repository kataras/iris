package main

import (
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/jwt"
)

/*
Documentation:
    https://github.com/kataras/jwt#table-of-contents
*/

// Replace with your own key and keep them secret.
// The "signatureSharedKey" is used for the HMAC(HS256) signature algorithm.
var signatureSharedKey = []byte("sercrethatmaycontainch@r32length")

func main() {
	app := iris.New()

	app.Get("/", generateToken)
	app.Get("/protected", protected)

	app.Listen(":8080")
}

type fooClaims struct {
	Foo string `json:"foo"`
}

func generateToken(ctx iris.Context) {
	claims := fooClaims{
		Foo: "bar",
	}

	// Sign and generate compact form token.
	token, err := jwt.Sign(jwt.HS256, signatureSharedKey, claims, jwt.MaxAge(10*time.Minute))
	if err != nil {
		ctx.StopWithStatus(iris.StatusInternalServerError)
		return
	}

	tokenString := string(token) // or jwt.BytesToString
	ctx.HTML(`Token: ` + tokenString + `<br/><br/>
		<a href="/protected?token=` + tokenString + `">/protected?token=` + tokenString + `</a>`)
}

func protected(ctx iris.Context) {
	// Extract the token, e.g. cookie, Authorization: Bearer $token
	// or URL query.
	token := ctx.URLParam("token")
	// Verify the token.
	verifiedToken, err := jwt.Verify(jwt.HS256, signatureSharedKey, []byte(token))
	if err != nil {
		ctx.StopWithStatus(iris.StatusUnauthorized)
		return
	}

	ctx.Writef("This is an authenticated request.\n\n")

	// Decode the custom claims.
	var claims fooClaims
	verifiedToken.Claims(&claims)

	// Just an example on how you can retrieve all the standard claims (set by jwt.MaxAge, "exp").
	standardClaims := jwt.GetVerifiedToken(ctx).StandardClaims

	expiresAtString := standardClaims.ExpiresAt().Format(ctx.Application().ConfigurationReadOnly().GetTimeFormat())
	timeLeft := standardClaims.Timeleft()

	ctx.Writef("foo=%s\nexpires at: %s\ntime left: %s\n", claims.Foo, expiresAtString, timeLeft)
}
