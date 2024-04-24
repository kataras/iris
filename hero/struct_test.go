package hero_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kataras/iris/v12"
	. "github.com/kataras/iris/v12/hero"
	"github.com/kataras/iris/v12/httptest"
)

type testStruct struct {
	Ctx iris.Context
}

func (c *testStruct) MyHandler(name string) testOutput {
	return fn(42, testInput{Name: name})
}

func (c *testStruct) MyHandler2(id int, in testInput) testOutput {
	return fn(id, in)
}

func (c *testStruct) MyHandler3(in testInput) testOutput {
	return fn(42, in)
}

func (c *testStruct) MyHandler4() {
	c.Ctx.WriteString("MyHandler4")
}

func TestStruct(t *testing.T) {
	app := iris.New()

	b := New()
	s := b.Struct(&testStruct{}, 0)

	postHandler := s.MethodHandler("MyHandler", 0) // fallbacks such as {path} and {string} should registered first when same path.
	app.Post("/{name:string}", postHandler)
	postHandler2 := s.MethodHandler("MyHandler2", 0)
	app.Post("/{id:int}", postHandler2)
	postHandler3 := s.MethodHandler("MyHandler3", 0)
	app.Post("/myHandler3", postHandler3)
	getHandler := s.MethodHandler("MyHandler4", 0)
	app.Get("/myHandler4", getHandler)

	e := httptest.New(t, app)
	e.POST("/" + input.Name).Expect().Status(httptest.StatusOK).JSON().IsEqual(expectedOutput)
	path := fmt.Sprintf("/%d", expectedOutput.ID)
	e.POST(path).WithJSON(input).Expect().Status(httptest.StatusOK).JSON().IsEqual(expectedOutput)
	e.POST("/myHandler3").WithJSON(input).Expect().Status(httptest.StatusOK).JSON().IsEqual(expectedOutput)
	e.GET("/myHandler4").Expect().Status(httptest.StatusOK).Body().IsEqual("MyHandler4")
}

type testStructErrorHandler struct{}

func (s *testStructErrorHandler) HandleError(ctx iris.Context, err error) {
	ctx.StopWithError(httptest.StatusConflict, err)
}

func (s *testStructErrorHandler) Handle(errText string) error {
	return errors.New(errText)
}

func TestStructErrorHandler(t *testing.T) {
	b := New()
	s := b.Struct(&testStructErrorHandler{}, 0)

	app := iris.New()
	app.Get("/{errText:string}", s.MethodHandler("Handle", 0))

	expectedErrText := "an error"
	e := httptest.New(t, app)
	e.GET("/" + expectedErrText).Expect().Status(httptest.StatusConflict).Body().IsEqual(expectedErrText)
}

type (
	testServiceInterface1 interface {
		Parse() string
	}

	testServiceImpl1 struct {
		inner string
	}

	testServiceInterface2 interface {
	}

	testServiceImpl2 struct {
		tf int
	}

	testControllerDependenciesSorter struct {
		Service2 testServiceInterface2
		Service1 testServiceInterface1
	}
)

func (s *testServiceImpl1) Parse() string {
	return s.inner
}

func (c *testControllerDependenciesSorter) Index() string {
	return fmt.Sprintf("%#+v | %#+v", c.Service1, c.Service2)
}

func TestStructFieldsSorter(t *testing.T) { // see https://github.com/kataras/iris/issues/1343
	b := New()
	b.Register(&testServiceImpl1{"parser"})
	b.Register(&testServiceImpl2{24})
	s := b.Struct(&testControllerDependenciesSorter{}, 0)

	app := iris.New()
	app.Get("/", s.MethodHandler("Index", 0))

	e := httptest.New(t, app)

	expectedBody := `&hero_test.testServiceImpl1{inner:"parser"} | &hero_test.testServiceImpl2{tf:24}`
	e.GET("/").Expect().Status(httptest.StatusOK).Body().IsEqual(expectedBody)
}
