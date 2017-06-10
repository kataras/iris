package main

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
)

// $ cd _example
// $ go test -v
func TestNewApp(t *testing.T) {
	app := buildApp()
	e := httptest.New(app, t)

	// redirects to /admin without basic auth
	e.GET("/").Expect().Status(iris.StatusUnauthorized)
	// without basic auth
	e.GET("/admin").Expect().Status(iris.StatusUnauthorized)

	// with valid basic auth
	e.GET("/admin").WithBasicAuth("myusername", "mypassword").Expect().
		Status(iris.StatusOK).Body().Equal("Hello authenticated user: myusername from: /admin")
	e.GET("/admin/profile").WithBasicAuth("myusername", "mypassword").Expect().
		Status(iris.StatusOK).Body().Equal("Hello authenticated user: myusername from: /admin/profile")
	e.GET("/admin/settings").WithBasicAuth("myusername", "mypassword").Expect().
		Status(iris.StatusOK).Body().Equal("Hello authenticated user: myusername from: /admin/settings")

	// with invalid basic auth
	e.GET("/admin/settings").WithBasicAuth("invalidusername", "invalidpassword").
		Expect().Status(iris.StatusUnauthorized)

}
