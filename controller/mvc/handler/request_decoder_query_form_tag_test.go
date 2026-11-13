package handler_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/httptest"
)

type (
	// testQueryFormTag carries one binding tag per field: `form`.
	testQueryFormTag struct {
		ID   int    `form:"hosp_id" json:"hosp_id"`
		Name string `form:"hosp_name" json:"hosp_name"`
	}

	testQueryFormTagPerson struct {
		Name string `form:"name" json:"name"`
		Age  int    `form:"age" json:"age"`
	}

	// testQueryFormTagValidated validates itself: it has a Validate() error
	// method, which the form decoder calls after it decoded the value.
	testQueryFormTagValidated struct {
		Name string `form:"name" json:"name"`
		Age  int    `form:"age" json:"age"`
	}
)

func (v *testQueryFormTagValidated) Validate() error {
	if v.Name == "" {
		return errors.New("name is required")
	}

	return nil
}

func testEcho[T any](_ context.Context, req T) (T, error) { return req, nil }

func newQueryFormTagApp() *iris.Application {
	app := iris.New()

	app.Any("/typed", handler.Handler(testEcho[testQueryFormTag]))
	app.Any("/typed/person", handler.Handler(testEcho[testQueryFormTagPerson]))
	app.Any("/typed/validated", handler.Handler(testEcho[testQueryFormTagValidated]))

	return app
}

// A request type that has `form` tags only is bound from the URL query on a
// GET and from the form on a POST. The default request decoder hands such a
// GET to ctx.ReadBody, which reads the query with ctx.ReadQuery, and
// ctx.ReadQuery binds a field that has no `url` tag by its `form` tag.
// Issue #1725.
func TestDefaultRequestDecoderQueryFormTag(t *testing.T) {
	e := httptest.New(t, newQueryFormTagApp())

	// The GET of the issue.
	e.GET("/typed").WithQuery("hosp_id", 100).WithQuery("hosp_name", "get test name").
		Expect().Status(httptest.StatusOK).JSON().
		IsEqual(testQueryFormTag{ID: 100, Name: "get test name"})

	// The POST of the issue: the form is read, the query is not.
	e.POST("/typed").WithQuery("hosp_id", 100).WithQuery("hosp_name", "get test name").
		WithFormField("hosp_name", "post test name").
		Expect().Status(httptest.StatusOK).JSON().
		IsEqual(testQueryFormTag{Name: "post test name"})

	// Another method with a query and no body goes through ctx.ReadBody too.
	e.DELETE("/typed").WithQuery("hosp_id", 100).WithQuery("hosp_name", "get test name").
		Expect().Status(httptest.StatusOK).JSON().
		IsEqual(testQueryFormTag{ID: 100, Name: "get test name"})

	// An empty query value on a numeric field is not an error.
	e.GET("/typed/person").WithQuery("age", "").WithQuery("name", "x").
		Expect().Status(httptest.StatusOK).JSON().
		IsEqual(testQueryFormTagPerson{Name: "x"})
}

// The query is not read ahead of the body. A request type that has `form`
// tags and a Validate method is validated after its form was read, and a
// query on a POST does not reach it.
func TestDefaultRequestDecoderQueryFormTagValidate(t *testing.T) {
	e := httptest.New(t, newQueryFormTagApp())

	want := testQueryFormTagValidated{Name: "x"}

	e.POST("/typed/validated").WithFormField("name", "x").
		Expect().Status(httptest.StatusOK).JSON().IsEqual(want)

	e.POST("/typed/validated").WithJSON(iris.Map{"name": "x"}).
		Expect().Status(httptest.StatusOK).JSON().IsEqual(want)

	e.POST("/typed/validated").WithQuery("age", 3).WithFormField("name", "x").
		Expect().Status(httptest.StatusOK).JSON().IsEqual(want)

	e.POST("/typed/validated").WithQuery("age", 3).WithJSON(iris.Map{"name": "x"}).
		Expect().Status(httptest.StatusOK).JSON().IsEqual(want)

	// A value that is not a number does not reach a POST either.
	e.POST("/typed/person").WithQuery("age", "").WithFormField("age", 5).
		Expect().Status(httptest.StatusOK).JSON().
		IsEqual(testQueryFormTagPerson{Age: 5})
}
