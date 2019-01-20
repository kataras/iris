package main

import (
	"testing"

	"github.com/kataras/iris/httptest"
)

func TestCustomContextNewImpl(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app, httptest.URL("http://localhost:8080"))

	e.GET("/").Expect().
		Status(httptest.StatusOK).
		ContentType("text/html").
		Body().Equal("<b>Hello from our *Context</b>")

	expectedName := "iris"
	e.POST("/set").WithFormField("name", expectedName).Expect().
		Status(httptest.StatusOK).
		Body().Equal("set session = " + expectedName)

	e.GET("/get").Expect().
		Status(httptest.StatusOK).
		Body().Equal(expectedName)
}
