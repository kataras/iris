package main

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
)

func TestSessionsEncodeDecode(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app, httptest.URL("http://example.com"))

	es := e.GET("/set").Expect()
	es.Status(iris.StatusOK)
	es.Cookies().NotEmpty()
	es.Body().Equal("All ok session setted to: iris")

	e.GET("/get").Expect().Status(iris.StatusOK).Body().Equal("The name on the /set was: iris")
	// delete and re-get
	e.GET("/delete").Expect().Status(iris.StatusOK)
	e.GET("/get").Expect().Status(iris.StatusOK).Body().Equal("The name on the /set was: ")
	// set, clear and re-get
	e.GET("/set").Expect().Body().Equal("All ok session setted to: iris")
	e.GET("/clear").Expect().Status(iris.StatusOK)
	e.GET("/get").Expect().Status(iris.StatusOK).Body().Equal("The name on the /set was: ")
}
