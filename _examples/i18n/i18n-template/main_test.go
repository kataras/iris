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
	e.GET("/members").Expect().Status(httptest.StatusOK).
		Body().Equal("There are 42 members registered")
	e.GET("/").WithHeader("Accept-Language", "el").Expect().Status(httptest.StatusOK).
		Body().Equal("Γειά 2 σκυλί")

	e.GET("/other").Expect().Status(httptest.StatusOK).
		Body().Equal("AccessLogClear: Clear Access Log\nTitle: Account Connections")
	e.GET("/other").WithHeader("Accept-Language", "el").Expect().Status(httptest.StatusOK).
		Body().Equal("AccessLogClear: Καθαρισμός Πρόσβαση στο αρχείο καταγραφής\nTitle: Λογαριασμός Συνδέσεις")
}
