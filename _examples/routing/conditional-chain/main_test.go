package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestNewConditionalHandler(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	e.GET("/api/v1/users").Expect().Status(httptest.StatusOK).
		Body().IsEqual("requested: <b>/api/v1/users</b>")
	e.GET("/api/v1/users").WithQuery("admin", "true").Expect().Status(httptest.StatusOK).
		Body().IsEqual("<title>Admin</title>\n<h1>Hello Admin</h1><br>requested: <b>/api/v1/users</b>")
}
