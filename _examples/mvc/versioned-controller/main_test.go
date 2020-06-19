package main

import (
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/kataras/iris/v12/versioning"
)

func TestVersionedController(t *testing.T) {
	app := newApp()

	e := httptest.New(t, app)
	e.GET("/data").WithHeader(versioning.AcceptVersionHeaderKey, "1").Expect().
		Status(iris.StatusOK).Body().Equal("data (v1.x)")
	e.GET("/data").WithHeader(versioning.AcceptVersionHeaderKey, "2.3.0").Expect().
		Status(iris.StatusOK).Body().Equal("data (v2.x)")
	e.GET("/data").WithHeader(versioning.AcceptVersionHeaderKey, "3.1").Expect().
		Status(iris.StatusOK).Body().Equal("data (v3.x)")
	// Test invalid version or no version at all.
	e.GET("/data").WithHeader(versioning.AcceptVersionHeaderKey, "4").Expect().
		Status(iris.StatusOK).Body().Equal("data")
	e.GET("/data").Expect().
		Status(iris.StatusOK).Body().Equal("data")
}
