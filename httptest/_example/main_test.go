package main

import (
	"testing"

	"gopkg.in/kataras/iris.v6/httptest"
)

// $ cd _example
// $ go test -v
func TestNewApp(t *testing.T) {
	app := newApp()
	e := httptest.New(app, t)

	// test nauthorized
	e.GET("/hello").Expect().Status(401).Body().Equal("<h1> Unauthorized Page! </h1>")
	// test our login flash message
	name := "myname"
	e.POST("/login").WithFormField("name", name).Expect().Status(200)
	// test the /hello again with the flash (a message which deletes itself after it has been shown to the user)
	// setted on /login previously.
	expectedResponse := map[string]interface{}{
		"Message": "Hello",
		"From":    name,
	}
	e.GET("/hello").Expect().Status(200).JSON().Equal(expectedResponse)
	// test /hello nauthorized again, it should be return 401 now (flash should be removed)
	e.GET("/hello").Expect().Status(401).Body().Equal("<h1> Unauthorized Page! </h1>")
}

// for advanced test examples navigate there:
// https://github.com/gavv/httpexpect/blob/master/_examples/iris_test.go
