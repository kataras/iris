package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestI18nLoaderFuncMap(t *testing.T) {
	app := newApp()

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().IsEqual("Become a MEMBER")
	e.GET("/title").Expect().Status(httptest.StatusOK).
		Body().IsEqual("Account Connections")
	e.GET("/").WithHeader("Accept-Language", "el").Expect().Status(httptest.StatusOK).
		Body().IsEqual("Γίνε ΜΈΛΟΣ")
	e.GET("/title").WithHeader("Accept-Language", "el").Expect().Status(httptest.StatusOK).
		Body().IsEqual("Λογαριασμός Συνδέσεις")
}
