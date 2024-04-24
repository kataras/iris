package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestBasicAuth(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	// redirects to /admin without basic auth
	e.GET("/").Expect().Status(httptest.StatusUnauthorized)
	// without basic auth
	e.GET("/admin").Expect().Status(httptest.StatusUnauthorized)

	// with valid basic auth
	e.GET("/admin").WithBasicAuth("myusername", "mypassword").Expect().
		Status(httptest.StatusOK).Body().IsEqual("/admin myusername:mypassword")
	e.GET("/admin/profile").WithBasicAuth("myusername", "mypassword").Expect().
		Status(httptest.StatusOK).Body().IsEqual("/admin/profile myusername:mypassword")
	e.GET("/admin/settings").WithBasicAuth("myusername", "mypassword").Expect().
		Status(httptest.StatusOK).Body().IsEqual("/admin/settings myusername:mypassword")

	// with invalid basic auth
	e.GET("/admin/settings").WithBasicAuth("invalidusername", "invalidpassword").
		Expect().Status(httptest.StatusUnauthorized)
}
