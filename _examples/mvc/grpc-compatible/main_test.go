package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestGRPCCompatible(t *testing.T) {
	app := newApp()

	e := httptest.New(t, app)
	e.POST("/helloworld.Greeter/SayHello").WithJSON(map[string]string{"name": "makis"}).Expect().
		Status(httptest.StatusOK).
		JSON().IsEqual(map[string]string{"message": "Hello makis"})
}
