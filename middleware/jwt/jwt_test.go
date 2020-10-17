// Package jwt_test contains simple Iris jwt tests. Most of the jwt functionality is already tested inside the jose package itself.
package jwt_test

import (
	"os"
	"testing"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/kataras/iris/v12/middleware/jwt"
)

type userClaims struct {
	// Optionally:
	Issuer   string       `json:"iss"`
	Subject  string       `json:"sub"`
	Audience jwt.Audience `json:"aud"`
	//
	Username string `json:"username"`
}

const testMaxAge = 7 * time.Second

// Random RSA verification and encryption.
func TestRSA(t *testing.T) {
	j := jwt.RSA(testMaxAge)
	t.Cleanup(func() {
		os.Remove(jwt.DefaultSignFilename)
		os.Remove(jwt.DefaultEncFilename)
	})
	testWriteVerifyBlockToken(t, j)
}

// HMAC verification and encryption.
func TestHMAC(t *testing.T) {
	j := jwt.HMAC(testMaxAge, "secret", "itsa16bytesecret")
	testWriteVerifyBlockToken(t, j)
}

func TestNew_HMAC(t *testing.T) {
	j, err := jwt.New(testMaxAge, jwt.HS256, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	err = j.WithEncryption(jwt.A128GCM, jwt.DIRECT, []byte("itsa16bytesecret"))
	if err != nil {
		t.Fatal(err)
	}

	testWriteVerifyBlockToken(t, j)
}

// HMAC verification only (unecrypted).
func TestVerify(t *testing.T) {
	j, err := jwt.New(testMaxAge, jwt.HS256, []byte("another secret"))
	if err != nil {
		t.Fatal(err)
	}
	testWriteVerifyBlockToken(t, j)
}

func testWriteVerifyBlockToken(t *testing.T, j *jwt.JWT) {
	t.Helper()

	j.UseBlocklist()
	j.Extractors = append(j.Extractors, jwt.FromJSON("access_token"))

	customClaims := &userClaims{
		Issuer:   "an-issuer",
		Audience: jwt.Audience{"an-audience"},
		Subject:  "user",
		Username: "kataras",
	}

	app := iris.New()
	app.OnErrorCode(iris.StatusUnauthorized, func(ctx iris.Context) {
		if err := ctx.GetErr(); err != nil {
			// Test accessing the private error and set this as the response body.
			ctx.WriteString(err.Error())
		} else { // Else the default behavior
			ctx.WriteString(iris.StatusText(iris.StatusUnauthorized))
		}
	})

	app.Get("/auth", func(ctx iris.Context) {
		j.WriteToken(ctx, customClaims)
	})

	app.Post("/protected", func(ctx iris.Context) {
		var claims userClaims
		_, err := j.VerifyToken(ctx, &claims)
		if err != nil {
			// t.Logf("%s: %v", ctx.Path(), err)
			ctx.StopWithError(iris.StatusUnauthorized, iris.PrivateError(err))
			return
		}

		ctx.JSON(claims)
	})

	m := app.Party("/middleware")
	m.Use(j.Verify(func() interface{} {
		return new(userClaims)
	}))
	m.Post("/protected", func(ctx iris.Context) {
		claims := jwt.Get(ctx)
		ctx.JSON(claims)
	})
	m.Post("/invalidate", func(ctx iris.Context) {
		ctx.Logout() // OR j.Invalidate(ctx)
	})

	e := httptest.New(t, app)

	// Get token.
	rawToken := e.GET("/auth").Expect().Status(httptest.StatusOK).Body().Raw()
	if rawToken == "" {
		t.Fatalf("empty token")
	}

	restrictedPaths := [...]string{"/protected", "/middleware/protected"}

	now := time.Now()
	for _, path := range restrictedPaths {
		// Authorization Header.
		e.POST(path).WithHeader("Authorization", "Bearer "+rawToken).Expect().
			Status(httptest.StatusOK).JSON().Equal(customClaims)

		// URL Query.
		e.POST(path).WithQuery("token", rawToken).Expect().
			Status(httptest.StatusOK).JSON().Equal(customClaims)

		// JSON Body.
		e.POST(path).WithJSON(iris.Map{"access_token": rawToken}).Expect().
			Status(httptest.StatusOK).JSON().Equal(customClaims)

		// Missing "Bearer".
		e.POST(path).WithHeader("Authorization", rawToken).Expect().
			Status(httptest.StatusUnauthorized).Body().Equal("token is missing")
	}

	// Invalidate the token.
	e.POST("/middleware/invalidate").WithQuery("token", rawToken).Expect().
		Status(httptest.StatusOK)
	// Token is blocked by server.
	e.POST("/middleware/protected").WithQuery("token", rawToken).Expect().
		Status(httptest.StatusUnauthorized).Body().Equal("token is blocked")

	expireRemDur := testMaxAge - time.Since(now)

	// Expiration.
	time.Sleep(expireRemDur /* -end */)
	for _, path := range restrictedPaths {
		e.POST(path).WithQuery("token", rawToken).Expect().
			Status(httptest.StatusUnauthorized).Body().Equal("token is expired (exp)")
	}
}

func TestVerifyMap(t *testing.T) {
	j := jwt.HMAC(testMaxAge, "secret", "itsa16bytesecret")
	expectedClaims := iris.Map{
		"iss":      "tester",
		"username": "makis",
		"roles":    []string{"admin"},
	}

	app := iris.New()
	app.Get("/user/auth", func(ctx iris.Context) {
		err := j.WriteToken(ctx, expectedClaims)
		if err != nil {
			ctx.StopWithError(iris.StatusUnauthorized, err)
			return
		}

		if expectedClaims["exp"] == nil || expectedClaims["iat"] == nil {
			ctx.StopWithText(iris.StatusBadRequest,
				"exp or/and iat is nil - this means that the expiry was not set")
			return
		}
	})

	userAPI := app.Party("/user")
	userAPI.Post("/", func(ctx iris.Context) {
		var claims iris.Map
		if _, err := j.VerifyToken(ctx, &claims); err != nil {
			ctx.StopWithError(iris.StatusUnauthorized, iris.PrivateError(err))
			return
		}

		ctx.JSON(claims)
	})

	// Test map + Verify middleware.
	userAPI.Post("/middleware", j.Verify(nil), func(ctx iris.Context) {
		claims := jwt.Get(ctx)
		ctx.JSON(claims)
	})

	e := httptest.New(t, app, httptest.LogLevel("error"))
	token := e.GET("/user/auth").Expect().Status(httptest.StatusOK).Body().Raw()
	if token == "" {
		t.Fatalf("empty token")
	}

	e.POST("/user").WithHeader("Authorization", "Bearer "+token).Expect().
		Status(httptest.StatusOK).JSON().Equal(expectedClaims)

	e.POST("/user/middleware").WithHeader("Authorization", "Bearer "+token).Expect().
		Status(httptest.StatusOK).JSON().Equal(expectedClaims)

	e.POST("/user").Expect().Status(httptest.StatusUnauthorized)
}

type customClaims struct {
	Username string `json:"username"`
	Token    string `json:"token"`
}

func (c *customClaims) SetToken(tok string) {
	c.Token = tok
}

func TestVerifyStruct(t *testing.T) {
	maxAge := testMaxAge / 2
	j := jwt.HMAC(maxAge, "secret", "itsa16bytesecret")

	app := iris.New()
	app.Get("/user/auth", func(ctx iris.Context) {
		err := j.WriteToken(ctx, customClaims{
			Username: "makis",
		})
		if err != nil {
			ctx.StopWithError(iris.StatusUnauthorized, err)
			return
		}
	})

	userAPI := app.Party("/user")
	userAPI.Post("/", func(ctx iris.Context) {
		var claims customClaims
		if _, err := j.VerifyToken(ctx, &claims); err != nil {
			ctx.StopWithError(iris.StatusUnauthorized, iris.PrivateError(err))
			return
		}

		ctx.JSON(claims)
	})

	e := httptest.New(t, app)
	token := e.GET("/user/auth").Expect().Status(httptest.StatusOK).Body().Raw()
	if token == "" {
		t.Fatalf("empty token")
	}
	e.POST("/user").WithHeader("Authorization", "Bearer "+token).Expect().
		Status(httptest.StatusOK).JSON().Object().ContainsMap(iris.Map{
		"username": "makis",
		"token":    token, // Test SetToken.
	})

	e.POST("/user").Expect().Status(httptest.StatusUnauthorized)
	time.Sleep(maxAge)
	e.POST("/user").WithHeader("Authorization", "Bearer "+token).Expect().Status(httptest.StatusUnauthorized)
}

func TestVerifyJSON(t *testing.T) {
	j := jwt.HMAC(testMaxAge, "secret", "itsa16bytesecret")

	app := iris.New()
	app.Get("/user/auth", func(ctx iris.Context) {
		err := j.WriteToken(ctx, iris.Map{"roles": []string{"admin"}})
		if err != nil {
			ctx.StopWithError(iris.StatusUnauthorized, err)
			return
		}
	})

	app.Post("/", j.VerifyJSON(), func(ctx iris.Context) {
		claims := struct {
			Roles []string `json:"roles"`
		}{}
		jwt.ReadJSON(ctx, &claims)
		ctx.JSON(claims)
	})

	e := httptest.New(t, app, httptest.LogLevel("error"))
	token := e.GET("/user/auth").Expect().Status(httptest.StatusOK).Body().Raw()
	if token == "" {
		t.Fatalf("empty token")
	}

	e.POST("/").WithHeader("Authorization", "Bearer "+token).Expect().
		Status(httptest.StatusOK).JSON().Equal(iris.Map{"roles": []string{"admin"}})

	e.POST("/").Expect().Status(httptest.StatusUnauthorized)
}

func TestVerifyUserAndExpected(t *testing.T) { // Tests the jwt.User struct + context validator + expected.
	maxAge := testMaxAge / 2
	j := jwt.HMAC(maxAge, "secret", "itsa16bytesecret")
	expectedUser := j.NewUser(jwt.Username("makis"), jwt.Roles("admin"), jwt.Fields(iris.Map{
		"custom": true,
	})) // only for the sake of the test, we iniitalize it here.
	expectedUser.Issuer = "tester"

	app := iris.New()
	app.Get("/user/auth", func(ctx iris.Context) {
		tok, err := expectedUser.GetToken()
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}
		ctx.WriteString(tok)
	})

	userAPI := app.Party("/user")
	userAPI.Use(jwt.WithExpected(jwt.Expected{Issuer: "tester"}, j.VerifyUser()))
	userAPI.Post("/", func(ctx iris.Context) {
		user := ctx.User()
		ctx.JSON(user)
	})

	e := httptest.New(t, app)
	token := e.GET("/user/auth").Expect().Status(httptest.StatusOK).Body().Raw()
	if token == "" {
		t.Fatalf("empty token")
	}

	e.POST("/user").WithHeader("Authorization", "Bearer "+token).Expect().
		Status(httptest.StatusOK).JSON().Equal(expectedUser)

	// Test generic client message if we don't manage the private error by ourselves.
	e.POST("/user").Expect().Status(httptest.StatusUnauthorized).Body().Equal("Unauthorized")
}
