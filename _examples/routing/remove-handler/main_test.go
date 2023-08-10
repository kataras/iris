package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestSimpleRouteRemoveHandler(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	e.GET("/api/users").Expect().Status(httptest.StatusOK).Body().IsEqual("OK")
}
