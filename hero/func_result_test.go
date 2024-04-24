package hero_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/httptest"

	. "github.com/kataras/iris/v12/hero"
)

func GetText() string {
	return "text"
}

func GetStatus() int {
	return iris.StatusBadGateway
}

func GetTextWithStatusOk() (string, int) {
	return "OK", iris.StatusOK
}

// tests should have output arguments mixed
func GetStatusWithTextNotOkBy(first string, second string) (int, string) {
	return iris.StatusForbidden, "NOT_OK_" + first + second
}

func GetTextAndContentType() (string, string) {
	return "<b>text</b>", "text/html"
}

type testCustomResult struct {
	HTML string
}

// The only one required function to make that a custom Response dispatcher.
func (r testCustomResult) Dispatch(ctx iris.Context) {
	_, _ = ctx.HTML(r.HTML)
}

func GetCustomResponse() testCustomResult {
	return testCustomResult{"<b>text</b>"}
}

func GetCustomResponseWithStatusOk() (testCustomResult, int) {
	return testCustomResult{"<b>OK</b>"}, iris.StatusOK
}

func GetCustomResponseWithStatusNotOk() (testCustomResult, int) {
	return testCustomResult{"<b>internal server error</b>"}, iris.StatusInternalServerError
}

type testCustomStruct struct {
	Name string `json:"name" xml:"name"`
	Age  int    `json:"age" xml:"age"`
}

func GetCustomStruct() testCustomStruct {
	return testCustomStruct{"Iris", 2}
}

func GetCustomStructWithStatusNotOk() (testCustomStruct, int) {
	return testCustomStruct{"Iris", 2}, iris.StatusInternalServerError
}

func GetCustomStructWithContentType() (testCustomStruct, string) {
	return testCustomStruct{"Iris", 2}, "text/xml"
}

func GetCustomStructWithError(ctx iris.Context) (s testCustomStruct, err error) {
	s = testCustomStruct{"Iris", 2}
	if ctx.URLParamExists("err") {
		err = errors.New("omit return of testCustomStruct and fire error")
	}

	// it should send the testCustomStruct as JSON if error is nil
	// otherwise it should fire the default error(BadRequest) with the error's text.
	return
}

type err struct {
	Status  int    `json:"status_code"`
	Message string `json:"message"`
}

func (e err) Dispatch(ctx iris.Context) {
	// write the status code based on the err's StatusCode.
	ctx.StatusCode(e.Status)
	// send to the client the whole object as json
	_ = ctx.JSON(e)
}

func GetCustomErrorAsDispatcher() err {
	return err{iris.StatusBadRequest, "this is my error as json"}
}

func GetCustomTypedNilEmptyResponse() iris.Map {
	return nil
}

func GetCustomTypedPtrNilEmptyResponse() *iris.Map {
	return nil
}

func GetCustomMapNilEmptyResponse() map[string]interface{} {
	return nil
}

func GetCustomPtrStructNilEmptyResponse() *testCustomStruct {
	return nil
}

func TestFuncResult(t *testing.T) {
	app := iris.New()
	h := New()
	// for any 'By', by is not required but we use this suffix here, like controllers
	// to make it easier for the future to resolve if any bug.
	// add the binding for path parameters.

	app.Get("/text", h.Handler(GetText))
	app.Get("/status", h.Handler(GetStatus))
	app.Get("/text/with/status/ok", h.Handler(GetTextWithStatusOk))
	app.Get("/status/with/text/not/ok/{first}/{second}", h.Handler(GetStatusWithTextNotOkBy))
	app.Get("/text/and/content/type", h.Handler(GetTextAndContentType))
	//
	app.Get("/custom/response", h.Handler(GetCustomResponse))
	app.Get("/custom/response/with/status/ok", h.Handler(GetCustomResponseWithStatusOk))
	app.Get("/custom/response/with/status/not/ok", h.Handler(GetCustomResponseWithStatusNotOk))
	//
	app.Get("/custom/struct", h.Handler(GetCustomStruct))
	app.Get("/custom/struct/with/status/not/ok", h.Handler(GetCustomStructWithStatusNotOk))
	app.Get("/custom/struct/with/content/type", h.Handler(GetCustomStructWithContentType))
	app.Get("/custom/struct/with/error", h.Handler(GetCustomStructWithError))
	app.Get("/custom/error/as/dispatcher", h.Handler(GetCustomErrorAsDispatcher))

	app.Get("/custom/nil/typed", h.Handler(GetCustomTypedNilEmptyResponse))
	app.Get("/custom/nil/typed/ptr", h.Handler(GetCustomTypedPtrNilEmptyResponse))
	app.Get("/custom/nil/map", h.Handler(GetCustomMapNilEmptyResponse))
	app.Get("/custom/nil/struct", h.Handler(GetCustomPtrStructNilEmptyResponse))

	e := httptest.New(t, app)

	e.GET("/text").Expect().Status(iris.StatusOK).
		Body().IsEqual("text")

	e.GET("/status").Expect().Status(iris.StatusBadGateway)

	e.GET("/text/with/status/ok").Expect().Status(iris.StatusOK).
		Body().IsEqual("OK")

	e.GET("/status/with/text/not/ok/first/second").Expect().Status(iris.StatusForbidden).
		Body().IsEqual("NOT_OK_firstsecond")
	// Author's note: <-- if that fails means that the last binder called for both input args,
	// see path_param_binder.go

	e.GET("/text/and/content/type").Expect().Status(iris.StatusOK).
		ContentType("text/html", "utf-8").
		Body().IsEqual("<b>text</b>")

	e.GET("/custom/response").Expect().Status(iris.StatusOK).
		ContentType("text/html", "utf-8").
		Body().IsEqual("<b>text</b>")
	e.GET("/custom/response/with/status/ok").Expect().Status(iris.StatusOK).
		ContentType("text/html", "utf-8").
		Body().IsEqual("<b>OK</b>")
	e.GET("/custom/response/with/status/not/ok").Expect().Status(iris.StatusInternalServerError).
		ContentType("text/html", "utf-8").
		Body().IsEqual("<b>internal server error</b>")

	expectedResultFromCustomStruct := map[string]interface{}{
		"name": "Iris",
		"age":  2,
	}
	e.GET("/custom/struct").Expect().Status(iris.StatusOK).
		JSON().IsEqual(expectedResultFromCustomStruct)
	e.GET("/custom/struct/with/status/not/ok").Expect().Status(iris.StatusInternalServerError).
		JSON().IsEqual(expectedResultFromCustomStruct)
	e.GET("/custom/struct/with/content/type").Expect().Status(iris.StatusOK).
		ContentType("text/xml", "utf-8")
	e.GET("/custom/struct/with/error").Expect().Status(iris.StatusOK).
		JSON().IsEqual(expectedResultFromCustomStruct)
	e.GET("/custom/struct/with/error").WithQuery("err", true).Expect().
		Status(iris.StatusBadRequest). // the default status code if error is not nil
		// the content should be not JSON it should be the status code's text
		// it will fire the error's text
		Body().IsEqual("omit return of testCustomStruct and fire error")

	e.GET("/custom/error/as/dispatcher").Expect().
		Status(iris.StatusBadRequest). // the default status code if error is not nil
		// the content should be not JSON it should be the status code's text
		// it will fire the error's text
		JSON().IsEqual(err{iris.StatusBadRequest, "this is my error as json"})

	// its result is nil should give an empty response but content-type is set correctly.
	e.GET("/custom/nil/typed").Expect().
		Status(iris.StatusOK).ContentType(context.ContentJSONHeaderValue).Body().IsEmpty()
	e.GET("/custom/nil/typed/ptr").Expect().
		Status(iris.StatusOK).ContentType(context.ContentJSONHeaderValue).Body().IsEmpty()
	e.GET("/custom/nil/map").Expect().
		Status(iris.StatusOK).ContentType(context.ContentJSONHeaderValue).Body().IsEmpty()
	e.GET("/custom/nil/struct").Expect().
		Status(iris.StatusOK).ContentType(context.ContentJSONHeaderValue).Body().IsEmpty()
}

type (
	testPreflightRequest struct {
		FailCode int `json:"fail_code"` // for the sake of the test.
	}

	testPreflightResponse struct {
		Code    int
		Message string
	}
)

func (r testPreflightResponse) Preflight(ctx iris.Context) error {
	if r.Code == httptest.StatusInternalServerError {
		return fmt.Errorf("custom error")
	}

	if r.Code > 0 {
		ctx.StatusCode(r.Code)
	}

	return nil
}

func TestPreflightResult(t *testing.T) {
	app := iris.New()
	c := New()
	handler := c.Handler(func(in testPreflightRequest) testPreflightResponse {
		return testPreflightResponse{Code: in.FailCode, Message: http.StatusText(in.FailCode)}
	})
	app.Post("/", handler)
	handler2 := c.Handler(func(in testInput) (int, testOutput) {
		return httptest.StatusAccepted, testOutput{Name: in.Name}
	})
	app.Post("/alternative", handler2)

	e := httptest.New(t, app)

	expected1 := testPreflightResponse{Code: httptest.StatusOK, Message: "OK"}
	e.POST("/").WithJSON(testPreflightRequest{FailCode: expected1.Code}).
		Expect().Status(httptest.StatusOK).JSON().IsEqual(expected1)

	expected2 := testPreflightResponse{Code: httptest.StatusBadRequest, Message: "Bad Request"}
	e.POST("/").WithJSON(testPreflightRequest{FailCode: expected2.Code}).
		Expect().Status(httptest.StatusBadRequest).JSON().IsEqual(expected2)

	// Test error returned from Preflight.
	e.POST("/").WithJSON(testPreflightRequest{FailCode: httptest.StatusInternalServerError}).
		Expect().Status(httptest.StatusBadRequest).Body().IsEqual("custom error")

	// Can be done without Preflight as the second output argument can be a status code.
	expected4 := testOutput{Name: "my_name"}
	e.POST("/alternative").WithJSON(testInput{expected4.Name}).
		Expect().Status(httptest.StatusAccepted).JSON().IsEqual(expected4)
}

func TestResponseErr(t *testing.T) {
	app := iris.New()
	var expectedErr = errors.New("response error")

	app.OnAnyErrorCode(func(ctx iris.Context) {
		err := ctx.GetErr()
		if err != expectedErr {
			t.Fatalf("expected error value does not match")
		}

		ctx.WriteString(err.Error())
	})

	app.ConfigureContainer().Get("/", func() Response {
		return Response{Code: iris.StatusBadGateway, Err: expectedErr}
	})

	e := httptest.New(t, app)
	e.GET("/").Expect().Status(iris.StatusBadGateway).Body().IsEqual("response error")
}
