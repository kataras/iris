package middleware

import (
	"testing"

	"github.com/iris-contrib/middleware/basicauth"
	"github.com/kataras/iris"
	. "github.com/kataras/iris/test"
)

/* Yes, middleware front-end tests also here, so if you want to test you have to go get at least one middleware */

func TestMiddlewareBasicAuth(t *testing.T) {
	var (
		api       = iris.New()
		user1     = "myusername"
		user1pass = "mypassword"
		user2     = "mySecondusername"
		user2pass = "mySecondpassword"
		users     = map[string]string{user1: user1pass, user2: user2pass}
		config    = basicauth.Config{ // default configuration, same as .Default(users)
			Users:      users,
			Realm:      "Authorization Required",
			ContextKey: "user",
		}

		authentication = basicauth.New(config)
	)

	// for global api.Use(authentication)
	h := func(ctx *iris.Context) {
		// username := ctx.GetString(config.ContextKey)
		// or
		username := config.User(ctx)
		ctx.Write("%s", username)
	}

	api.Get("/secret", authentication, h)
	api.Get("/secret/profile", authentication, h)
	api.Get("/othersecret", authentication, h)
	api.Get("/no_authenticate", h) // the body should be empty here

	e := Tester(api, t)

	testBasicAuth := func(path, username, password string) {
		e.GET(path).WithBasicAuth(username, password).Expect().Status(iris.StatusOK).Body().Equal(username)
	}
	testBasicAuthInvalid := func(path, username, password string) {
		e.GET(path).WithBasicAuth(username, password).Expect().Status(iris.StatusUnauthorized)
	}

	// valid auth
	testBasicAuth("/secret", user1, user1pass)
	testBasicAuth("/secret", user2, user2pass)
	testBasicAuth("/secret/profile", user1, user1pass)
	testBasicAuth("/secret/profile", user2, user2pass)
	testBasicAuth("/othersecret", user1, user1pass)
	testBasicAuth("/othersecret", user2, user2pass)
	// invalid auth
	testBasicAuthInvalid("/secret", user1+"invalid", user1pass)
	testBasicAuthInvalid("/secret", user2, user2pass+"invalid")
	testBasicAuthInvalid("/secret/profile", user1+"invalid", user1pass+"c")
	testBasicAuthInvalid("/secret/profile", user2, user2pass+"invalid")
	testBasicAuthInvalid("/othersecret", user1+"invalid", user1pass)
	testBasicAuthInvalid("/othersecret", user2, user2pass+"invalid")
	// no auth
	e.GET("/no_authenticate").Expect().Status(iris.StatusOK).Body().Empty()

}
