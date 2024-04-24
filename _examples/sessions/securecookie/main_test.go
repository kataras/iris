package main

import (
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

func TestSessionsEncodeDecode(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app, httptest.URL("http://example.com"))

	es := e.GET("/set").Expect()
	es.Status(iris.StatusOK)
	es.Cookies().NotEmpty()
	es.Body().IsEqual("All ok session set to: iris [isNew=true]")

	e.GET("/get").Expect().Status(iris.StatusOK).Body().IsEqual("The username on the /set was: iris")
	// delete and re-get
	e.GET("/delete").Expect().Status(iris.StatusOK)
	e.GET("/get").Expect().Status(iris.StatusOK).Body().IsEqual("The username on the /set was: ")
	// set, clear and re-get
	e.GET("/set").Expect().Body().IsEqual("All ok session set to: iris [isNew=false]")
	e.GET("/clear").Expect().Status(iris.StatusOK)
	e.GET("/get").Expect().Status(iris.StatusOK).Body().IsEqual("The username on the /set was: ")
}
