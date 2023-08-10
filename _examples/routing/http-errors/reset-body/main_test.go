package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestResetCompressionAndFireError(t *testing.T) { // #1569
	app := newApp()

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(httptest.StatusBadRequest).Body().IsEqual("custom error")
}
