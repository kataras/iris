package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestReadParams(t *testing.T) {
	app := newApp()

	e := httptest.New(t, app)

	expectedBody := `myParams: main.myParams{Name:"kataras", Age:27, Tail:[]string{"iris", "web", "framework"}}`
	e.GET("/kataras/27/iris/web/framework").Expect().Status(httptest.StatusOK).Body().IsEqual(expectedBody)
}
