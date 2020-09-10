package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestI18nLoaderFuncMap(t *testing.T) {
	app := newApp()

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().Equal("Hi 2 dogs")
	e.GET("/singular").Expect().Status(httptest.StatusOK).
		Body().Equal("Hi 1 dog")
	e.GET("/").WithHeader("Accept-Language", "el").Expect().Status(httptest.StatusOK).
		Body().Equal("Γειά 2 σκυλί")
}
