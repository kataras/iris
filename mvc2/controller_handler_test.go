package mvc2_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	"github.com/kataras/iris/mvc"
	// "github.com/kataras/iris/mvc/activator/methodfunc"
	//. "github.com/kataras/iris/mvc2"
)

type testController struct {
	mvc.C
	Service *TestServiceImpl

	reqField string
}

func (c *testController) Get() string {
	return "index"
}

func (c *testController) BeginRequest(ctx iris.Context) {
	c.C.BeginRequest(ctx)
	c.reqField = ctx.URLParam("reqfield")
}

func (c *testController) OnActivate(t *mvc.TController) {
	t.Handle("GET", "/histatic", "HiStatic")
	t.Handle("GET", "/hiservice", "HiService")
	t.Handle("GET", "/hiparam/{ps:string}", "HiParamBy")
	t.Handle("GET", "/hiparamempyinput/{ps:string}", "HiParamEmptyInputBy")
}

func (c *testController) HiStatic() string {
	return c.reqField
}

func (c *testController) HiService() string {
	return c.Service.Say("hi")
}

func (c *testController) HiParamBy(v string) string {
	return v
}

func (c *testController) HiParamEmptyInputBy() string {
	return "empty in but served with ctx.Params.Get('ps')=" + c.Ctx.Params().Get("ps")
}

func TestControllerHandler(t *testing.T) {
	app := iris.New()
	app.Controller("/", new(testController), &TestServiceImpl{prefix: "service:"})
	e := httptest.New(t, app, httptest.LogLevel("debug"))

	// test the index, is not part of the current package's implementation but do it.
	e.GET("/").Expect().Status(httptest.StatusOK).Body().Equal("index")

	// the important things now.

	// this test ensures that the BeginRequest of the controller will be
	// called correctly and also the controller is binded to the first input argument
	// (which is the function's receiver, if any, in this case the *testController in go).
	expectedReqField := "this is a request field filled by this url param"
	e.GET("/histatic").WithQuery("reqfield", expectedReqField).Expect().Status(httptest.StatusOK).
		Body().Equal(expectedReqField)
	// this test makes sure that the binded values of the controller is handled correctly
	// and can be used in a user-defined, dynamic "mvc handler".
	e.GET("/hiservice").Expect().Status(httptest.StatusOK).
		Body().Equal("service: hi")

	// this worked with a temporary variadic on the resolvemethodfunc which is not
	// correct design, I should split the path and params with the rest of implementation
	// in order a simple template.Src can be given.
	e.GET("/hiparam/value").Expect().Status(httptest.StatusOK).
		Body().Equal("value")
	e.GET("/hiparamempyinput/value").Expect().Status(httptest.StatusOK).
		Body().Equal("empty in but served with ctx.Params.Get('ps')=value")

}
