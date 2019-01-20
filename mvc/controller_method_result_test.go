package mvc_test

import (
	"errors"
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/httptest"

	. "github.com/kataras/iris/mvc"
)

type testControllerMethodResult struct {
	Ctx context.Context
}

func (c *testControllerMethodResult) Get() Result {
	return Response{
		Text: "Hello World!",
	}
}

func (c *testControllerMethodResult) GetWithStatus() Response { // or Result again, no problem.
	return Response{
		Text: "This page doesn't exist",
		Code: iris.StatusNotFound,
	}
}

type testCustomStruct struct {
	Name string `json:"name" xml:"name"`
	Age  int    `json:"age" xml:"age"`
}

func (c *testControllerMethodResult) GetJson() Result {
	var err error
	if c.Ctx.URLParamExists("err") {
		err = errors.New("error here")
	}
	return Response{
		Err:    err, // if err != nil then it will fire the error's text with a BadRequest.
		Object: testCustomStruct{Name: "Iris", Age: 2},
	}
}

var things = []string{"thing 0", "thing 1", "thing 2"}

func (c *testControllerMethodResult) GetThingWithTryBy(index int) Result {
	failure := Response{
		Text: "thing does not exist",
		Code: iris.StatusNotFound,
	}

	return Try(func() Result {
		// if panic because of index exceed the slice
		// then the "failure" response will be returned instead.
		return Response{Text: things[index]}
	}, failure)
}

func (c *testControllerMethodResult) GetThingWithTryDefaultBy(index int) Result {
	return Try(func() Result {
		// if panic because of index exceed the slice
		// then the default failure response will be returned instead (400 bad request).
		return Response{Text: things[index]}
	})
}

func TestControllerMethodResult(t *testing.T) {
	app := iris.New()
	New(app).Handle(new(testControllerMethodResult))

	e := httptest.New(t, app)

	e.GET("/").Expect().Status(iris.StatusOK).
		Body().Equal("Hello World!")

	e.GET("/with/status").Expect().Status(iris.StatusNotFound).
		Body().Equal("This page doesn't exist")

	e.GET("/json").Expect().Status(iris.StatusOK).
		JSON().Equal(iris.Map{
		"name": "Iris",
		"age":  2,
	})

	e.GET("/json").WithQuery("err", true).Expect().
		Status(iris.StatusBadRequest).
		Body().Equal("error here")

	e.GET("/thing/with/try/1").Expect().
		Status(iris.StatusOK).
		Body().Equal("thing 1")
	// failure because of index exceed the slice
	e.GET("/thing/with/try/3").Expect().
		Status(iris.StatusNotFound).
		Body().Equal("thing does not exist")

	e.GET("/thing/with/try/default/3").Expect().
		Status(iris.StatusBadRequest).
		Body().Equal("Bad Request")
}

type testControllerMethodResultTypes struct {
	Ctx context.Context
}

func (c *testControllerMethodResultTypes) GetText() string {
	return "text"
}

func (c *testControllerMethodResultTypes) GetStatus() int {
	return iris.StatusBadGateway
}

func (c *testControllerMethodResultTypes) GetTextWithStatusOk() (string, int) {
	return "OK", iris.StatusOK
}

// tests should have output arguments mixed
func (c *testControllerMethodResultTypes) GetStatusWithTextNotOkBy(first string, second string) (int, string) {
	return iris.StatusForbidden, "NOT_OK_" + first + second
}

func (c *testControllerMethodResultTypes) GetTextAndContentType() (string, string) {
	return "<b>text</b>", "text/html"
}

type testControllerMethodCustomResult struct {
	HTML string
}

// The only one required function to make that a custom Response dispatcher.
func (r testControllerMethodCustomResult) Dispatch(ctx context.Context) {
	ctx.HTML(r.HTML)
}

func (c *testControllerMethodResultTypes) GetCustomResponse() testControllerMethodCustomResult {
	return testControllerMethodCustomResult{"<b>text</b>"}
}

func (c *testControllerMethodResultTypes) GetCustomResponseWithStatusOk() (testControllerMethodCustomResult, int) {
	return testControllerMethodCustomResult{"<b>OK</b>"}, iris.StatusOK
}

func (c *testControllerMethodResultTypes) GetCustomResponseWithStatusNotOk() (testControllerMethodCustomResult, int) {
	return testControllerMethodCustomResult{"<b>internal server error</b>"}, iris.StatusInternalServerError
}

func (c *testControllerMethodResultTypes) GetCustomStruct() testCustomStruct {
	return testCustomStruct{"Iris", 2}
}

func (c *testControllerMethodResultTypes) GetCustomStructWithStatusNotOk() (testCustomStruct, int) {
	return testCustomStruct{"Iris", 2}, iris.StatusInternalServerError
}

func (c *testControllerMethodResultTypes) GetCustomStructWithContentType() (testCustomStruct, string) {
	return testCustomStruct{"Iris", 2}, "text/xml"
}

func (c *testControllerMethodResultTypes) GetCustomStructWithError() (s testCustomStruct, err error) {
	s = testCustomStruct{"Iris", 2}
	if c.Ctx.URLParamExists("err") {
		err = errors.New("omit return of testCustomStruct and fire error")
	}

	// it should send the testCustomStruct as JSON if error is nil
	// otherwise it should fire the default error(BadRequest) with the error's text.
	return
}

func TestControllerMethodResultTypes(t *testing.T) {
	app := iris.New()
	New(app).Handle(new(testControllerMethodResultTypes))

	e := httptest.New(t, app)

	e.GET("/text").Expect().Status(iris.StatusOK).
		Body().Equal("text")

	e.GET("/status").Expect().Status(iris.StatusBadGateway)

	e.GET("/text/with/status/ok").Expect().Status(iris.StatusOK).
		Body().Equal("OK")

	e.GET("/status/with/text/not/ok/first/second").Expect().Status(iris.StatusForbidden).
		Body().Equal("NOT_OK_firstsecond")
	// Author's note: <-- if that fails means that the last binder called for both input args,
	// see path_param_binder.go

	e.GET("/text/and/content/type").Expect().Status(iris.StatusOK).
		ContentType("text/html", "utf-8").
		Body().Equal("<b>text</b>")

	e.GET("/custom/response").Expect().Status(iris.StatusOK).
		ContentType("text/html", "utf-8").
		Body().Equal("<b>text</b>")
	e.GET("/custom/response/with/status/ok").Expect().Status(iris.StatusOK).
		ContentType("text/html", "utf-8").
		Body().Equal("<b>OK</b>")
	e.GET("/custom/response/with/status/not/ok").Expect().Status(iris.StatusInternalServerError).
		ContentType("text/html", "utf-8").
		Body().Equal("<b>internal server error</b>")

	expectedResultFromCustomStruct := map[string]interface{}{
		"name": "Iris",
		"age":  2,
	}
	e.GET("/custom/struct").Expect().Status(iris.StatusOK).
		JSON().Equal(expectedResultFromCustomStruct)
	e.GET("/custom/struct/with/status/not/ok").Expect().Status(iris.StatusInternalServerError).
		JSON().Equal(expectedResultFromCustomStruct)
	e.GET("/custom/struct/with/content/type").Expect().Status(iris.StatusOK).
		ContentType("text/xml", "utf-8")
	e.GET("/custom/struct/with/error").Expect().Status(iris.StatusOK).
		JSON().Equal(expectedResultFromCustomStruct)
	e.GET("/custom/struct/with/error").WithQuery("err", true).Expect().
		Status(iris.StatusBadRequest). // the default status code if error is not nil
		// the content should be not JSON it should be the status code's text
		// it will fire the error's text
		Body().Equal("omit return of testCustomStruct and fire error")
}

type testControllerViewResultRespectCtxViewData struct {
	T *testing.T
}

func (t *testControllerViewResultRespectCtxViewData) BeginRequest(ctx context.Context) {
	ctx.ViewData("name_begin", "iris_begin")
}

func (t *testControllerViewResultRespectCtxViewData) EndRequest(ctx context.Context) {
	// check if data is not overridden by return View {Data: context.Map...}

	dataWritten := ctx.GetViewData()
	if dataWritten == nil {
		t.T.Fatalf("view data is nil, both BeginRequest and Get failed to write the data")
		return
	}

	if dataWritten["name_begin"] == nil {
		t.T.Fatalf(`view data[name_begin] is nil,
			BeginRequest's ctx.ViewData call have been overridden  by Get's return View {Data: }.
			Total view data: %v`, dataWritten)
	}

	if dataWritten["name"] == nil {
		t.T.Fatalf("view data[name] is nil, Get's return View {Data: } didn't work. Total view data: %v", dataWritten)
	}
}

func (t *testControllerViewResultRespectCtxViewData) Get() Result {
	return View{
		Name: "doesnt_exists.html",
		Data: context.Map{"name": "iris"}, // we care about this only.
		Code: iris.StatusInternalServerError,
	}
}

func TestControllerViewResultRespectCtxViewData(t *testing.T) {
	app := iris.New()
	m := New(app.Party("/"))
	m.Register(t)
	m.Handle(new(testControllerViewResultRespectCtxViewData))

	e := httptest.New(t, app)

	e.GET("/").Expect().Status(iris.StatusInternalServerError)
}
