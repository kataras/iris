package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestMVCOverlapping(t *testing.T) {
	app := newApp()

	e := httptest.New(t, app, httptest.URL("http://example.com"))
	// unauthenticated.
	e.GET("/user").Expect().Status(httptest.StatusOK).Body().Equal("custom action to redirect on authentication page")
	// login.
	e.POST("/user/login").Expect().Status(httptest.StatusOK)
	// authenticated.
	e.GET("/user").Expect().Status(httptest.StatusOK).Body().Equal(`UserController.Get: The Authenticated type
can be used to secure a controller's method too.`)
	// logout.
	e.POST("/user/logout").Expect().Status(httptest.StatusOK)
	// unauthenticated.
	e.GET("/user").Expect().Status(httptest.StatusOK).Body().Equal("custom action to redirect on authentication page")
}
