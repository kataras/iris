package main

import (
	"testing"

	"github.com/kataras/iris/httptest"
)

// $ go test -v
func TestNewApp(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	// redirects to /admin without basic auth
	e.GET("/").Expect().Status(httptest.StatusUnauthorized)
	// without basic auth
	e.GET("/admin").Expect().Status(httptest.StatusUnauthorized)

	// with valid basic auth
	e.GET("/admin").WithBasicAuth("myusername", "mypassword").Expect().
		Status(httptest.StatusOK).Body().Equal("/admin myusername:mypassword")
	e.GET("/admin/profile").WithBasicAuth("myusername", "mypassword").Expect().
		Status(httptest.StatusOK).Body().Equal("/admin/profile myusername:mypassword")
	e.GET("/admin/settings").WithBasicAuth("myusername", "mypassword").Expect().
		Status(httptest.StatusOK).Body().Equal("/admin/settings myusername:mypassword")

	// with invalid basic auth
	e.GET("/admin/settings").WithBasicAuth("invalidusername", "invalidpassword").
		Expect().Status(httptest.StatusUnauthorized)

}
