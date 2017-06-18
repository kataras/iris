package main

import (
	"testing"

	"github.com/cdren/iris"
	"github.com/cdren/iris/httptest"
)

// $ cd $GOPATH/src/github.com/cdren/iris/_examples/intermediate/httptest
// $ go test -v
func TestNewApp(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	// redirects to /admin without basic auth
	e.GET("/").Expect().Status(iris.StatusUnauthorized)
	// without basic auth
	e.GET("/admin").Expect().Status(iris.StatusUnauthorized)

	// with valid basic auth
	e.GET("/admin").WithBasicAuth("myusername", "mypassword").Expect().
		Status(iris.StatusOK).Body().Equal("/admin myusername:mypassword")
	e.GET("/admin/profile").WithBasicAuth("myusername", "mypassword").Expect().
		Status(iris.StatusOK).Body().Equal("/admin/profile myusername:mypassword")
	e.GET("/admin/settings").WithBasicAuth("myusername", "mypassword").Expect().
		Status(iris.StatusOK).Body().Equal("/admin/settings myusername:mypassword")

	// with invalid basic auth
	e.GET("/admin/settings").WithBasicAuth("invalidusername", "invalidpassword").
		Expect().Status(iris.StatusUnauthorized)

}
