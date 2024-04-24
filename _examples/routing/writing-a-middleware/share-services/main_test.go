package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestShareServices(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("Hello, Gophers!")
}
