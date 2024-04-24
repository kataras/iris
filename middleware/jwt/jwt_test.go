package jwt_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/kataras/iris/v12/middleware/jwt"
)

var testAlg, testSecret = jwt.HS256, []byte("sercrethatmaycontainch@r$")

type fooClaims struct {
	Foo string `json:"foo"`
}

// The actual tests are inside the kataras/jwt repository.
// This runs simple checks of just the middleware part.
func TestJWT(t *testing.T) {
	app := iris.New()

	signer := jwt.NewSigner(testAlg, testSecret, 3*time.Second)
	app.Get("/", func(ctx iris.Context) {
		claims := fooClaims{Foo: "bar"}
		token, err := signer.Sign(claims)
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}
		ctx.Write(token)
	})

	verifier := jwt.NewVerifier(testAlg, testSecret)
	verifier.ErrorHandler = func(ctx iris.Context, err error) { // app.OnErrorCode(401, ...)
		ctx.StopWithError(iris.StatusUnauthorized, err)
	}
	middleware := verifier.Verify(func() interface{} { return new(fooClaims) })
	app.Get("/protected", middleware, func(ctx iris.Context) {
		claims := jwt.Get(ctx).(*fooClaims)
		ctx.WriteString(claims.Foo)
	})

	e := httptest.New(t, app)

	// Get generated token.
	token := e.GET("/").Expect().Status(iris.StatusOK).Body().Raw()
	// Test Header.
	headerValue := fmt.Sprintf("Bearer %s", token)
	e.GET("/protected").WithHeader("Authorization", headerValue).Expect().
		Status(iris.StatusOK).Body().IsEqual("bar")
	// Test URL query.
	e.GET("/protected").WithQuery("token", token).Expect().
		Status(iris.StatusOK).Body().IsEqual("bar")

	// Test unauthorized.
	e.GET("/protected").Expect().Status(iris.StatusUnauthorized)
	e.GET("/protected").WithHeader("Authorization", "missing bearer").Expect().Status(iris.StatusUnauthorized)
	e.GET("/protected").WithQuery("token", "invalid_token").Expect().Status(iris.StatusUnauthorized)
	// Test expired (note checks happen based on second round).
	time.Sleep(5 * time.Second)
	e.GET("/protected").WithHeader("Authorization", headerValue).Expect().
		Status(iris.StatusUnauthorized).Body().IsEqual("jwt: token expired")
}
