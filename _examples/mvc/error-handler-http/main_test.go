package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestControllerHandleHTTPError(t *testing.T) {
	const (
		expectedIndex    = "Hello!"
		expectedNotFound = "<h3>Not Found Custom Page Rendered through Controller's HandleHTTPError</h3>"
	)

	app := newApp()

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual(expectedIndex)
	e.GET("/a_notefound").Expect().Status(httptest.StatusNotFound).ContentType("text/html").Body().IsEqual(expectedNotFound)
}
