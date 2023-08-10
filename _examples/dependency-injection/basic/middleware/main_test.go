package main

import (
	"testing"

	"github.com/kataras/iris/v12/httptest"
)

func TestDependencyInjectionBasic_Middleware(t *testing.T) {
	app := newApp()

	e := httptest.New(t, app)
	e.POST("/42").WithJSON(testInput{Email: "my_email"}).Expect().
		Status(httptest.StatusOK).
		JSON().IsEqual(testOutput{ID: 42, Name: "my_email"})

	// it should stop the execution at the middleware and return the middleware's status code,
	// because the error is `ErrStopExecution`.
	e.POST("/42").WithJSON(testInput{Email: "invalid"}).Expect().
		Status(httptest.StatusAccepted).Body().IsEmpty()

	// it should stop the execution at the middleware and return the error's text.
	e.POST("/42").WithJSON(testInput{Email: "error"}).Expect().
		Status(httptest.StatusConflict).Body().IsEqual("my_error")
}
