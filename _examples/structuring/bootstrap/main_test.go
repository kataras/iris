package main

import (
	"testing"

	"github.com/kataras/iris/httptest"
)

// go test -v
func TestApp(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app.Application)

	// test our routes
	e.GET("/").Expect().Status(httptest.StatusOK)
	e.GET("/follower/42").Expect().Status(httptest.StatusOK).
		Body().Equal("from /follower/{id:long} with ID: 42")
	e.GET("/following/52").Expect().Status(httptest.StatusOK).
		Body().Equal("from /following/{id:long} with ID: 52")
	e.GET("/like/64").Expect().Status(httptest.StatusOK).
		Body().Equal("from /like/{id:long} with ID: 64")

	// test not found
	e.GET("/notfound").Expect().Status(httptest.StatusNotFound)
	expectedErr := map[string]interface{}{
		"app":     app.AppName,
		"status":  httptest.StatusNotFound,
		"message": "",
	}
	e.GET("/anotfoundwithjson").WithQuery("json", nil).
		Expect().Status(httptest.StatusNotFound).JSON().Equal(expectedErr)
}
