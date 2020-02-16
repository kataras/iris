package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestGRPCCompatible(t *testing.T) {
	app := newApp()

	e := httptest.New(t, app)
	e.POST("/login").WithJSON(map[string]string{"username": "makis"}).Expect().
		Status(httptest.StatusOK).
		JSON().Equal(map[string]string{"message": "makis logged"})
}
