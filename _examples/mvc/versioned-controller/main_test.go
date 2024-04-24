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
	e.GET("/data").WithHeader(versioning.AcceptVersionHeaderKey, "1.0.0").Expect().
		Status(iris.StatusOK).Body().IsEqual("data (v1.x)")
	e.GET("/data").WithHeader(versioning.AcceptVersionHeaderKey, "2.3.0").Expect().
		Status(iris.StatusOK).Body().IsEqual("data (v2.x)")
	e.GET("/data").WithHeader(versioning.AcceptVersionHeaderKey, "3.1.0").Expect().
		Status(iris.StatusOK).Body().IsEqual("data (v3.x)")

	// Test invalid version or no version at all.
	e.GET("/data").WithHeader(versioning.AcceptVersionHeaderKey, "4.0.0").Expect().
		Status(iris.StatusOK).Body().IsEqual("data")
	e.GET("/data").Expect().
		Status(iris.StatusOK).Body().IsEqual("data")

	// Test Deprecated (v1)
	ex := e.GET("/data").WithHeader(versioning.AcceptVersionHeaderKey, "1.0.0").Expect()
	ex.Status(iris.StatusOK).Body().IsEqual("data (v1.x)")
	ex.Header("X-API-Warn").Equal(opts.WarnMessage)
	expectedDateStr := opts.DeprecationDate.Format(app.ConfigurationReadOnly().GetTimeFormat())
	ex.Header("X-API-Deprecation-Date").Equal(expectedDateStr)
}
