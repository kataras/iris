package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestI18nLoaderFuncMap(t *testing.T) {
	app := newApp()

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().Equal("Become a MEMBER")
	e.GET("/title").Expect().Status(httptest.StatusOK).
		Body().Equal("Account Connections")
	e.GET("/").WithHeader("Accept-Language", "el").Expect().Status(httptest.StatusOK).
		Body().Equal("Γίνε ΜΈΛΟΣ")
	e.GET("/title").WithHeader("Accept-Language", "el").Expect().Status(httptest.StatusOK).
		Body().Equal("Λογαριασμός Συνδέσεις")
}
