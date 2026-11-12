package jwt_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/jwt"
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
	middleware := verifier.Verify(func() any { return new(fooClaims) })
	app.Get("/protected", middleware, func(ctx iris.Context) {
		claims, ok := jwt.Get[*fooClaims](ctx)
		if !ok {
			ctx.StopWithStatus(iris.StatusInternalServerError)
			return
		}
		ctx.WriteString(claims.Foo)
	})

	e := httptest.New(t, app)

	// Get generated token.
	token := e.GET("/").Expect().Status(iris.StatusOK).Body().Raw()
	// Test Header.
	headerValue := fmt.Sprintf("Bearer %s", token)
	e.GET("/protected").WithHeader("Authorization", headerValue).Expect().
		Status(iris.StatusOK).Body().IsEqual("bar")
	// URL query tokens are NOT accepted by default as of v14 (opt in with WithQueryToken).
	e.GET("/protected").WithQuery("token", token).Expect().Status(iris.StatusUnauthorized)

	// Test unauthorized.
	e.GET("/protected").Expect().Status(iris.StatusUnauthorized)
	e.GET("/protected").WithHeader("Authorization", "missing bearer").Expect().Status(iris.StatusUnauthorized)
	e.GET("/protected").WithQuery("token", "invalid_token").Expect().Status(iris.StatusUnauthorized)
	// Test expired (note checks happen based on second round).
	time.Sleep(5 * time.Second)
	e.GET("/protected").WithHeader("Authorization", headerValue).Expect().
		Status(iris.StatusUnauthorized).Body().IsEqual("jwt: token expired")
}

// TestJWTWithQueryToken verifies that the query-parameter extractor, disabled by
// default as of v14, can be opted back in with Verifier.WithQueryToken.
func TestJWTWithQueryToken(t *testing.T) {
	app := iris.New()

	signer := jwt.NewSigner(testAlg, testSecret, 30*time.Second)
	app.Get("/", func(ctx iris.Context) {
		token, err := signer.Sign(fooClaims{Foo: "bar"})
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}
		ctx.Write(token)
	})

	verifier := jwt.NewVerifier(testAlg, testSecret).WithQueryToken()
	verifier.ErrorHandler = func(ctx iris.Context, err error) {
		ctx.StopWithError(iris.StatusUnauthorized, err)
	}
	app.Get("/protected", verifier.Verify(func() any { return new(fooClaims) }), func(ctx iris.Context) {
		claims, _ := jwt.Get[*fooClaims](ctx)
		ctx.WriteString(claims.Foo)
	})

	e := httptest.New(t, app)
	token := e.GET("/").Expect().Status(iris.StatusOK).Body().Raw()

	// Opted in: the ?token= query parameter is accepted again.
	e.GET("/protected").WithQuery("token", token).Expect().Status(iris.StatusOK).Body().IsEqual("bar")
}

// Validators passed to NewVerifier (jwt.Expected{Issuer/Audience}) used to be
// stored and then never run, unless a Blocklist happened to be configured too.
// A token minted by a different service sharing the signing key - or with any
// foreign "iss"/"aud" - was therefore accepted.
func TestJWTVerifierExpectedIssuerIsEnforced(t *testing.T) {
	type claims struct {
		Foo string `json:"foo"`
	}

	newToken := func(issuer string) string {
		signer := jwt.NewSigner(testAlg, testSecret, 30*time.Second)
		token, err := signer.Sign(claims{Foo: "bar"}, jwt.Claims{Issuer: issuer})
		if err != nil {
			t.Fatal(err)
		}
		return string(token)
	}

	app := iris.New()

	verifier := jwt.NewVerifier(testAlg, testSecret, jwt.Expected{Issuer: "my-app"})
	verifier.ErrorHandler = func(ctx iris.Context, err error) {
		ctx.StopWithError(iris.StatusUnauthorized, err)
	}

	app.Get("/protected", verifier.Verify(func() any { return new(claims) }), func(ctx iris.Context) {
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)

	e.GET("/protected").WithHeader("Authorization", "Bearer "+newToken("my-app")).
		Expect().Status(iris.StatusOK).Body().IsEqual("ok")

	// A token from a peer service that shares the signing key must be rejected.
	e.GET("/protected").WithHeader("Authorization", "Bearer "+newToken("other-app")).
		Expect().Status(iris.StatusUnauthorized)
}

// TestGetWrongTypeReportsFalse checks that jwt.Get reports false, without a
// panic, when the stored claims are not of the requested type.
func TestGetWrongTypeReportsFalse(t *testing.T) {
	app := iris.New()

	signer := jwt.NewSigner(testAlg, testSecret, 30*time.Second)
	app.Get("/", func(ctx iris.Context) {
		token, err := signer.Sign(fooClaims{Foo: "bar"})
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}
		ctx.Write(token)
	})

	verifier := jwt.NewVerifier(testAlg, testSecret)
	app.Get("/protected", verifier.Verify(func() any { return new(fooClaims) }), func(ctx iris.Context) {
		// Stored value is a *fooClaims: a value type does not match.
		if _, ok := jwt.Get[fooClaims](ctx); ok {
			ctx.WriteString("unexpected match")
			return
		}
		claims, ok := jwt.Get[*fooClaims](ctx)
		if !ok {
			ctx.WriteString("no claims")
			return
		}
		ctx.WriteString(claims.Foo)
	})
	app.Get("/anonymous", func(ctx iris.Context) {
		_, ok := jwt.Get[*fooClaims](ctx)
		ctx.Writef("%v", ok)
	})

	e := httptest.New(t, app)
	token := e.GET("/").Expect().Status(iris.StatusOK).Body().Raw()
	e.GET("/protected").WithHeader("Authorization", "Bearer "+token).Expect().
		Status(iris.StatusOK).Body().IsEqual("bar")
	e.GET("/anonymous").Expect().Status(iris.StatusOK).Body().IsEqual("false")
}
