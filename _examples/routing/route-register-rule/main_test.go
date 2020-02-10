package main

import (
	"testing"

	"github.com/kataras/iris/v12/core/router"
	"github.com/kataras/iris/v12/httptest"
)

func TestRouteRegisterRuleExample(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

	for _, method := range router.AllMethods {
		tt := e.Request(method, "/").Expect().Status(httptest.StatusOK).Body()
		if method == "GET" {
			tt.Equal("From [./main.go:28] GET: / -> github.com/kataras/iris/v12/_examples/routing/route-register-rule.getHandler()")
		} else {
			tt.Equal("From [./main.go:30] " + method + ": / -> github.com/kataras/iris/v12/_examples/routing/route-register-rule.anyHandler()")
		}
	}
}
