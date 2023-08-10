// Package basicauth_tests performs black-box testing of the basicauth middleware.
// Note that, a secondary test is also available at: _examples/auth/basicauth/main_test.go
package basicauth_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/kataras/iris/v12/middleware/basicauth"
)

func TestBasicAuthUseRouter(t *testing.T) {
	app := iris.New()
	users := map[string]string{
		"usr":   "pss",
		"admin": "admin",
	}

	auth := basicauth.New(basicauth.Options{
		Allow:    basicauth.AllowUsers(users),
		Realm:    basicauth.DefaultRealm,
		MaxTries: 1,
	})

	app.UseRouter(auth)

	app.Get("/user_json", func(ctx iris.Context) {
		ctx.JSON(ctx.User())
	})

	app.Get("/user_string", func(ctx iris.Context) {
		user := ctx.User()

		authorization, _ := user.GetAuthorization()
		username, _ := user.GetUsername()
		password, _ := user.GetPassword()
		ctx.Writef("%s\n%s\n%s", authorization, username, password)
	})

	app.Get("/", func(ctx iris.Context) {
		username, _, _ := ctx.Request().BasicAuth()
		ctx.Writef("Hello, %s!", username)
	})

	app.Subdomain("static").Get("/", func(ctx iris.Context) {
		username, _, _ := ctx.Request().BasicAuth()
		ctx.Writef("Static, %s", username)
	})

	resetWithUseRouter := app.Subdomain("reset_with_use_router").ResetRouterFilters()
	resetWithUseRouter.UseRouter(func(ctx iris.Context) {
		ctx.Record()
		ctx.Writef("with use router\n")
		ctx.Next()
	})
	resetWithUseRouter.Get("/", func(ctx iris.Context) {
		username, _, _ := ctx.Request().BasicAuth()
		ctx.Writef("%s", username) // username should be empty.
	})
	// ^ order of these should not matter.
	app.Subdomain("reset").ResetRouterFilters().Get("/", func(ctx iris.Context) {
		username, _, _ := ctx.Request().BasicAuth()
		ctx.Writef("%s", username) // username should be empty.
	})

	e := httptest.New(t, app.Configure(
		iris.WithFireMethodNotAllowed,
		iris.WithResetOnFireErrorCode,
	))

	for username, password := range users {
		// Test pass authentication and route found.
		e.GET("/").WithBasicAuth(username, password).Expect().
			Status(httptest.StatusOK).Body().IsEqual(fmt.Sprintf("Hello, %s!", username))
		e.GET("/user_json").WithBasicAuth(username, password).Expect().
			Status(httptest.StatusOK).JSON().Object().ContainsMap(iris.Map{
			"username": username,
		})
		e.GET("/user_string").WithBasicAuth(username, password).Expect().
			Status(httptest.StatusOK).Body().
			Equal(fmt.Sprintf("%s\n%s\n%s", "Basic Authentication", username, password))

		// Test empty auth.
		e.GET("/").Expect().Status(httptest.StatusUnauthorized).Body().IsEqual("Unauthorized")
		// Test invalid auth.
		e.GET("/").WithBasicAuth(username, "invalid_password").Expect().
			Status(httptest.StatusForbidden)
		e.GET("/").WithBasicAuth("invaid_username", password).Expect().
			Status(httptest.StatusForbidden)

		// Test different method, it should pass the authentication (no stop on 401)
		// but it doesn't fire the GET route, instead it gives 405.
		e.POST("/").WithBasicAuth(username, password).Expect().
			Status(httptest.StatusMethodNotAllowed).Body().IsEqual("Method Not Allowed")

		// Test pass the authentication but route not found.
		e.GET("/notfound").WithBasicAuth(username, password).Expect().
			Status(httptest.StatusNotFound).Body().IsEqual("Not Found")

		// Test empty auth.
		e.GET("/notfound").Expect().Status(httptest.StatusUnauthorized).Body().IsEqual("Unauthorized")
		// Test invalid auth.
		e.GET("/notfound").WithBasicAuth(username, "invalid_password").Expect().
			Status(httptest.StatusForbidden)
		e.GET("/notfound").WithBasicAuth("invaid_username", password).Expect().
			Status(httptest.StatusForbidden)

		// Test subdomain inherited.
		sub := e.Builder(func(req *httptest.Request) {
			req.WithURL("http://static.mydomain.com")
		})

		// Test pass and route found.
		sub.GET("/").WithBasicAuth(username, password).Expect().
			Status(httptest.StatusOK).Body().IsEqual(fmt.Sprintf("Static, %s", username))

		// Test empty auth.
		sub.GET("/").Expect().Status(httptest.StatusUnauthorized)
		// Test invalid auth.
		sub.GET("/").WithBasicAuth(username, "invalid_password").Expect().
			Status(httptest.StatusForbidden)
		sub.GET("/").WithBasicAuth("invaid_username", password).Expect().
			Status(httptest.StatusForbidden)

		// Test pass the authentication but route not found.
		sub.GET("/notfound").WithBasicAuth(username, password).Expect().
			Status(httptest.StatusNotFound).Body().IsEqual("Not Found")

		// Test empty auth.
		sub.GET("/notfound").Expect().Status(httptest.StatusUnauthorized).Body().IsEqual("Unauthorized")
		// Test invalid auth.
		sub.GET("/notfound").WithBasicAuth(username, "invalid_password").Expect().
			Status(httptest.StatusForbidden)
		sub.GET("/notfound").WithBasicAuth("invaid_username", password).Expect().
			Status(httptest.StatusForbidden)

		// Test a reset-ed Party with a single one UseRouter
		// which writes on matched routes and reset and send the error on errors.
		// (all should pass without auth).
		sub = e.Builder(func(req *httptest.Request) {
			req.WithURL("http://reset_with_use_router.mydomain.com")
		})
		sub.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("with use router\n")
		sub.POST("/").Expect().Status(httptest.StatusMethodNotAllowed).Body().IsEqual("Method Not Allowed")
		sub.GET("/notfound").Expect().Status(httptest.StatusNotFound).Body().IsEqual("Not Found")

		// Test a reset-ed Party (all should pass without auth).
		sub = e.Builder(func(req *httptest.Request) {
			req.WithURL("http://reset.mydomain.com")
		})
		sub.GET("/").Expect().Status(httptest.StatusOK).Body().IsEmpty()
		sub.POST("/").Expect().Status(httptest.StatusMethodNotAllowed).Body().IsEqual("Method Not Allowed")
		sub.GET("/notfound").Expect().Status(httptest.StatusNotFound).Body().IsEqual("Not Found")
	}
}
