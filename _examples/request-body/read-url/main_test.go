package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestReadURL(t *testing.T) {
	app := newApp()

	e := httptest.New(t, app)

	expectedBody := `myURL: main.myURL{Name:"kataras", Age:27, Tail:[]string{"iris", "web", "framework"}}`
	e.GET("/iris/web/framework").WithQuery("name", "kataras").WithQuery("age", 27).Expect().Status(httptest.StatusOK).Body().IsEqual(expectedBody)
}
