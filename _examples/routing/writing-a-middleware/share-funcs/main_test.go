package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestShareFuncs(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual("Hello, Gophers!")
	e.GET("/2").Expect().Status(httptest.StatusOK).Body().IsEqual("Hello, Gophers [2]!")
	e.GET("/3").Expect().Status(httptest.StatusOK).Body().IsEqual("OK, job was executed.\nSee the command prompt.")
}
