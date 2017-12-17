package mvc_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/httptest"

	. "github.com/kataras/iris/mvc"
)

type testControllerHandle struct {
	Ctx     context.Context
	Service TestService

	reqField string
}

func (c *testControllerHandle) BeforeActivation(b BeforeActivation) { // BeforeActivation(t *mvc.TController) {
	b.Handle("GET", "/histatic", "HiStatic")
	b.Handle("GET", "/hiservice", "HiService")
	b.Handle("GET", "/hiparam/{ps:string}", "HiParamBy")
	b.Handle("GET", "/hiparamempyinput/{ps:string}", "HiParamEmptyInputBy")
}

func (c *testControllerHandle) BeginRequest(ctx iris.Context) {
	c.reqField = ctx.URLParam("reqfield")
}

func (c *testControllerHandle) EndRequest(ctx iris.Context) {}

func (c *testControllerHandle) Get() string {
	return "index"
}

func (c *testControllerHandle) HiStatic() string {
	return c.reqField
}

func (c *testControllerHandle) HiService() string {
	return c.Service.Say("hi")
}

func (c *testControllerHandle) HiParamBy(v string) string {
	return v
}

func (c *testControllerHandle) HiParamEmptyInputBy() string {
	return "empty in but served with ctx.Params.Get('ps')=" + c.Ctx.Params().Get("ps")
}

func TestControllerHandle(t *testing.T) {
	app := iris.New()

	m := NewEngine()
	m.Dependencies.Add(&TestServiceImpl{prefix: "service:"})
	m.Controller(app, new(testControllerHandle))

	e := httptest.New(t, app)

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
