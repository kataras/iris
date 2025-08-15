package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

// go test -v
func TestApp(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app.Application)

	// test our routes
	e.GET("/").Expect().Status(httptest.StatusOK)
	e.GET("/follower/42").Expect().Status(httptest.StatusOK).
		Body().IsEqual("from /follower/{id:int64} with ID: 42")
	e.GET("/following/52").Expect().Status(httptest.StatusOK).
		Body().IsEqual("from /following/{id:int64} with ID: 52")
	e.GET("/like/64").Expect().Status(httptest.StatusOK).
		Body().IsEqual("from /like/{id:int64} with ID: 64")

	// test not found
	e.GET("/notfound").Expect().Status(httptest.StatusNotFound)
	expectedErr := map[string]any{
		"app":     app.AppName,
		"status":  httptest.StatusNotFound,
		"message": "",
	}
	e.GET("/anotfoundwithjson").WithQuery("json", nil).
		Expect().Status(httptest.StatusNotFound).JSON().IsEqual(expectedErr)
}
