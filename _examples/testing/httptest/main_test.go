package main

import (
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

// $ go test -v
func TestNewApp(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app, httptest.Strict(true))

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

func TestHandlerUsingNetHTTP(t *testing.T) {
	handler := func(ctx iris.Context) {
		ctx.WriteString("Hello, World!")
	}

	// A shortcut for net/http/httptest.NewRecorder/NewRequest.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	httptest.Do(w, r, handler)
	if expected, got := "Hello, World!", w.Body.String(); expected != got {
		t.Fatalf("expected body: %s but got: %s", expected, got)
	}
}
