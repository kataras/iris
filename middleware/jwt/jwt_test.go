// Package jwt_test contains simple Iris jwt tests. Most of the jwt functionality is already tested inside the jose package itself.
package jwt_test

import (
	"testing"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/kataras/iris/v12/middleware/jwt"
)

type userClaims struct {
	jwt.Claims
	Username string
}

const testMaxAge = 3 * time.Second

// Random RSA verification and encryption.
func TestRSA(t *testing.T) {
	j := jwt.Random(testMaxAge)
	testWriteVerifyToken(t, j)
}

// HMAC verification and encryption.
func TestHMAC(t *testing.T) {
	j, err := jwt.New(testMaxAge, jwt.HS256, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	err = j.WithEncryption(jwt.A128GCM, jwt.DIRECT, []byte("itsa16bytesecret"))
	if err != nil {
		t.Fatal(err)
	}

	testWriteVerifyToken(t, j)
}

// HMAC verification only (unecrypted).
func TestVerify(t *testing.T) {
	j, err := jwt.New(testMaxAge, jwt.HS256, []byte("another secret"))
	if err != nil {
		t.Fatal(err)
	}
	testWriteVerifyToken(t, j)
}

func testWriteVerifyToken(t *testing.T, j *jwt.JWT) {
	t.Helper()

	j.Extractors = append(j.Extractors, jwt.FromJSON("access_token"))
	standardClaims := jwt.Claims{Issuer: "an-issuer", Audience: jwt.Audience{"an-audience"}}
	expectedClaims := userClaims{
		Claims:   j.Expiry(standardClaims),
		Username: "kataras",
	}

	app := iris.New()
	app.Get("/auth", func(ctx iris.Context) {
		j.WriteToken(ctx, expectedClaims)
	})

	app.Post("/restricted", func(ctx iris.Context) {
		var claims userClaims
		if err := j.VerifyToken(ctx, &claims); err != nil {
			ctx.StopWithStatus(iris.StatusUnauthorized)
			return
		}

		ctx.JSON(claims)
	})

	app.Post("/restricted_middleware", j.Verify, func(ctx iris.Context) {
		var claims userClaims
		if err := jwt.ReadClaims(ctx, &claims); err != nil {
			ctx.StopWithStatus(iris.StatusUnauthorized)
			return
		}

		ctx.JSON(claims)
	})

	e := httptest.New(t, app)

	// Get token.
	rawToken := e.GET("/auth").Expect().Status(httptest.StatusOK).Body().Raw()
	if rawToken == "" {
		t.Fatalf("empty token")
	}

	restrictedPaths := [...]string{"/restricted", "/restricted_middleware"}

	now := time.Now()
	for _, path := range restrictedPaths {
		// Authorization Header.
		e.POST(path).WithHeader("Authorization", "Bearer "+rawToken).Expect().
			Status(httptest.StatusOK).JSON().Equal(expectedClaims)

		// URL Query.
		e.POST(path).WithQuery("token", rawToken).Expect().
			Status(httptest.StatusOK).JSON().Equal(expectedClaims)

		// JSON Body.
		e.POST(path).WithJSON(iris.Map{"access_token": rawToken}).Expect().
			Status(httptest.StatusOK).JSON().Equal(expectedClaims)

		// Missing "Bearer".
		e.POST(path).WithHeader("Authorization", rawToken).Expect().
			Status(httptest.StatusUnauthorized)
	}
	expireRemDur := testMaxAge - time.Since(now)

	// Expiration.
	time.Sleep(expireRemDur /* -end */)
	for _, path := range restrictedPaths {
		e.POST(path).WithQuery("token", rawToken).Expect().Status(httptest.StatusUnauthorized)
	}
}
