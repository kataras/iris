// Package basicauth_test performs black-box testing of the basicauth middleware.
package basicauth_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/basicauth"
)

func TestBasicAuthUse(t *testing.T) {
	app := iris.New(iris.Options{
		Router: iris.RouterOptions{
			FireMethodNotAllowed: true,
		},
	})
	app.EnableStatusCodeResponse()

	users := map[string]string{
		"usr":   "pss",
		"admin": "admin",
	}

	auth := basicauth.New(basicauth.Options[*iris.SimpleUser]{
		Allow:    basicauth.AllowUsersMap(users),
		Realm:    basicauth.DefaultRealm,
		MaxTries: 1,
	})
	// To test the `Use` method we need to register the middleware only on matched routes,
	// so un-registered endpoints will fire 404 instead of 401.
	app.UseRelevant(auth)

	app.Get("/user_json", func(ctx iris.Context) {
		ctx.JSON(ctx.User())
	})

	app.Get("/x", func(ctx iris.Context) {
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

	resetWithUseRouter := app.Subdomain("reset_with_use_router")
	resetWithUseRouter.Get("/", func(ctx iris.Context) {
		username, _, _ := ctx.Request().BasicAuth()
		ctx.Writef("%s", username) // username should be empty.
	})
	// ^ order of these should not matter.
	resetSubdomain := app.Subdomain("reset")
	resetSubdomain.Get("/", func(ctx iris.Context) {
		username, _, _ := ctx.Request().BasicAuth()
		ctx.Writef("%s", username) // username should be empty.
	})

	e := httptest.New(t, app, httptest.URL("http://example.com"))

	for username, password := range users {
		t.Logf("Testing with user: %s:%s\n", username, password)

		// Test pass authentication and route found.
		e.GET("/").WithBasicAuth(username, password).Expect().
			Status(httptest.StatusOK).Body().IsEqual(fmt.Sprintf("Hello, %s!", username))
		e.GET("/user_json").WithBasicAuth(username, password).Expect().
			Status(httptest.StatusOK).JSON().Object().ContainsSubset(iris.Map{
			"username": username,
		})
		e.GET("/user_string").WithBasicAuth(username, password).Expect().
			Status(httptest.StatusNotFound).Body().IsEqual("Not Found")

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
		e.GET("/notfound").Expect().Status(httptest.StatusNotFound).Body().IsEqual("Not Found")
		// Test invalid auth.
		e.GET("/notfound").WithBasicAuth(username, "invalid_password").Expect().
			Status(httptest.StatusNotFound).Body().IsEqual("Not Found")
		e.GET("/notfound").WithBasicAuth("invaid_username", password).Expect().
			Status(httptest.StatusNotFound)

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
		sub.GET("/notfound").Expect().Status(httptest.StatusNotFound).Body().IsEqual("Not Found")
		// Test invalid auth.
		sub.GET("/notfound").WithBasicAuth(username, "invalid_password").Expect().
			Status(httptest.StatusNotFound).Body().IsEqual("Not Found")
		sub.GET("/notfound").WithBasicAuth("invaid_username", password).Expect().
			Status(httptest.StatusNotFound).Body().IsEqual("Not Found")

		// Test a reset-ed Router with a single one UseRouter
		// which writes on matched routes and reset and send the error on errors.
		// (all should pass without auth).
		sub = e.Builder(func(req *httptest.Request) {
			req.WithURL("http://reset_with_use_router.mydomain.com")
		})
		sub.GET("/").Expect().Status(httptest.StatusUnauthorized).Body().IsEqual("Unauthorized")
		sub.POST("/").Expect().Status(httptest.StatusMethodNotAllowed).Body().IsEqual("Method Not Allowed")
		sub.GET("/notfound").Expect().Status(httptest.StatusNotFound).Body().IsEqual("Not Found")

		// Test a reset-ed Router (all should pass without auth).
		sub = e.Builder(func(req *httptest.Request) {
			req.WithURL("http://reset.mydomain.com")
		})
		sub.GET("/").Expect().Status(httptest.StatusUnauthorized).Body().IsEqual("Unauthorized")
		sub.POST("/").Expect().Status(httptest.StatusMethodNotAllowed).Body().IsEqual("Method Not Allowed")
		sub.GET("/notfound").Expect().Status(httptest.StatusNotFound).Body().IsEqual("Not Found")
	}
}

type operator struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// The user type flows from Options[U] to GetUser[U]; ctx.User() keeps working
// with the context.User view of the same value.
func TestBasicAuthGetUser(t *testing.T) {
	operators := []operator{{"makis", "correct", "admin"}}

	app := iris.New()
	app.Get("/protected", basicauth.New(basicauth.Options[operator]{
		Allow: basicauth.AllowUsers(operators),
	}), func(ctx iris.Context) {
		op, ok := basicauth.GetUser[operator](ctx)
		if !ok {
			ctx.StopWithText(500, "GetUser[operator] reported false")
			return
		}

		if _, ok := basicauth.GetUser[*iris.SimpleUser](ctx); ok {
			ctx.StopWithText(500, "GetUser with the wrong type must report false")
			return
		}

		raw, _ := ctx.User().GetRaw()
		if raw.(operator) != op {
			ctx.StopWithText(500, "ctx.User().GetRaw() must be the same value")
			return
		}

		ctx.WriteString(op.Role)
	})

	e := httptest.New(t, app)
	e.GET("/protected").WithBasicAuth("makis", "correct").
		Expect().Status(httptest.StatusOK).Body().IsEqual("admin")
	e.GET("/protected").WithBasicAuth("makis", "wrong").
		Expect().Status(httptest.StatusUnauthorized)
}

// Default stores a *context.SimpleUser with the identity fields filled in.
func TestBasicAuthDefaultUser(t *testing.T) {
	app := iris.New()
	app.Get("/protected", basicauth.Default(map[string]string{"makis": "correct"}), func(ctx iris.Context) {
		u, ok := basicauth.GetUser[*iris.SimpleUser](ctx)
		if !ok {
			ctx.StopWithText(500, "GetUser[*iris.SimpleUser] reported false")
			return
		}

		if ctx.User() != u {
			ctx.StopWithText(500, "ctx.User() must be the same *SimpleUser")
			return
		}

		if _, ok := basicauth.AuthorizedAt(ctx); ok {
			ctx.StopWithText(500, "AuthorizedAt must report false without MaxAge")
			return
		}

		ctx.JSON(iris.Map{"id": u.ID, "username": u.Username, "authorization": u.Authorization})
	})

	e := httptest.New(t, app)
	e.GET("/protected").WithBasicAuth("makis", "correct").
		Expect().Status(httptest.StatusOK).JSON().Object().IsEqual(iris.Map{
		"id":            "makis",
		"username":      "makis",
		"authorization": "Basic Authentication",
	})
}

// ctx.Logout() forgets the stored login: the user is gone for the rest of the
// request, the response is a 401 and the next request is a fresh first login.
func TestBasicAuthLogout(t *testing.T) {
	app := iris.New()
	auth := basicauth.Default(map[string]string{"makis": "correct"})
	app.Get("/logout", auth, func(ctx iris.Context) {
		if err := ctx.Logout(); err != nil {
			ctx.StopWithError(500, err)
			return
		}

		if ctx.User() != nil {
			ctx.StopWithText(500, "ctx.User() must be nil after logout")
			return
		}
		if _, ok := basicauth.GetUser[*iris.SimpleUser](ctx); ok {
			ctx.StopWithText(500, "GetUser must report false after logout")
			return
		}
		if _, _, ok := ctx.Request().BasicAuth(); ok {
			ctx.StopWithText(500, "the Authorization header must be removed after logout")
			return
		}
	})

	e := httptest.New(t, app)
	e.GET("/logout").WithBasicAuth("makis", "correct").Expect().Status(httptest.StatusUnauthorized)
	e.GET("/logout").WithBasicAuth("makis", "correct").Expect().Status(httptest.StatusUnauthorized)
}

func TestBasicAuthRealm(t *testing.T) {
	app := iris.New()
	app.Get("/protected", basicauth.New(basicauth.Options[*iris.SimpleUser]{
		Realm: "Operators Only",
		Allow: basicauth.AllowUsersMap(map[string]string{"makis": "correct"}),
	}), func(ctx iris.Context) {})

	e := httptest.New(t, app)
	e.GET("/protected").Expect().Status(httptest.StatusUnauthorized).
		Header("WWW-Authenticate").IsEqual(`Basic realm="Operators Only"`)
}

func TestBasicAuthProxy(t *testing.T) {
	app := iris.New()
	app.Get("/protected", basicauth.New(basicauth.Options[*iris.SimpleUser]{
		Realm: basicauth.DefaultRealm,
		Proxy: true,
		Allow: basicauth.AllowUsersMap(map[string]string{"makis": "correct"}),
	}), func(ctx iris.Context) {
		username, _ := ctx.User().GetUsername()
		ctx.WriteString(username)
	})

	e := httptest.New(t, app)

	// Without credentials the proxy challenge is a 407 with Proxy-Authenticate.
	resp := e.GET("/protected").Expect().Status(httptest.StatusProxyAuthRequired)
	resp.Header("Proxy-Authenticate").IsEqual(`Basic realm="Authorization Required"`)
	resp.Header("WWW-Authenticate").IsEmpty()

	// Credentials must be sent through Proxy-Authorization, not Authorization.
	e.GET("/protected").WithBasicAuth("makis", "correct").Expect().Status(httptest.StatusProxyAuthRequired)
	e.GET("/protected").WithHeader("Proxy-Authorization", "Basic bWFraXM6Y29ycmVjdA==").
		Expect().Status(httptest.StatusOK).Body().IsEqual("makis")
}
