package main

import (
	"testing"

	"github.com/kataras/iris/httptest"
)

func TestMVCSession(t *testing.T) {
	e := httptest.New(t, newApp(), httptest.URL("http://example.com"))

	e1 := e.GET("/").Expect().Status(httptest.StatusOK)
	e1.Cookies().NotEmpty()
	e1.Body().Contains("1 visit")

	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().Contains("2 visit")

	e.GET("/").Expect().Status(httptest.StatusOK).
		Body().Contains("3 visit")
}
