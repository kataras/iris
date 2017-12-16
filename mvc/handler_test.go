package mvc_test

// black-box

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"

	. "github.com/kataras/iris/mvc"
)

// dynamic func
type testUserStruct struct {
	ID       int64
	Username string
}

func testBinderFunc(ctx iris.Context) testUserStruct {
	id, _ := ctx.Params().GetInt64("id")
	username := ctx.Params().Get("username")
	return testUserStruct{
		ID:       id,
		Username: username,
	}
}

// service
type (
	// these TestService and TestServiceImpl could be in lowercase, unexported
	// but the `Say` method should be exported however we have those exported
	// because of the controller handler test.
	TestService interface {
		Say(string) string
	}
	TestServiceImpl struct {
		prefix string
	}
)

func (s *TestServiceImpl) Say(message string) string {
	return s.prefix + " " + message
}

var (
	// binders, as user-defined
	testBinderFuncUserStruct = testBinderFunc
	testBinderService        = &TestServiceImpl{prefix: "say"}
	testBinderFuncParam      = func(ctx iris.Context) string {
		return ctx.Params().Get("param")
	}

	// consumers
	// a context as first input arg, which is not needed to be binded manually,
	// and a user struct which is binded to the input arg by the #1 func(ctx) any binder.
	testConsumeUserHandler = func(ctx iris.Context, user testUserStruct) {
		ctx.JSON(user)
	}

	// just one input arg, the service which is binded by the #2 service binder.
	testConsumeServiceHandler = func(service TestService) string {
		return service.Say("something")
	}
	// just one input arg, a standar string which is binded by the #3 func(ctx) any binder.
	testConsumeParamHandler = func(myParam string) string {
		return "param is: " + myParam
	}
)

func TestMakeHandler(t *testing.T) {
	var (
		h1 = MustMakeHandler(testConsumeUserHandler, reflect.ValueOf(testBinderFuncUserStruct))
		h2 = MustMakeHandler(testConsumeServiceHandler, reflect.ValueOf(testBinderService))
		h3 = MustMakeHandler(testConsumeParamHandler, reflect.ValueOf(testBinderFuncParam))
	)

	testAppWithMvcHandlers(t, h1, h2, h3)
}

func testAppWithMvcHandlers(t *testing.T, h1, h2, h3 iris.Handler) {
	app := iris.New()
	app.Get("/{id:long}/{username:string}", h1)
	app.Get("/service", h2)
	app.Get("/param/{param:string}", h3)

	expectedUser := testUserStruct{
		ID:       42,
		Username: "kataras",
	}

	e := httptest.New(t, app)
	// 1
	e.GET(fmt.Sprintf("/%d/%s", expectedUser.ID, expectedUser.Username)).Expect().Status(httptest.StatusOK).
		JSON().Equal(expectedUser)
	// 2
	e.GET("/service").Expect().Status(httptest.StatusOK).
		Body().Equal("say something")
	// 3
	e.GET("/param/the_param_value").Expect().Status(httptest.StatusOK).
		Body().Equal("param is: the_param_value")
}

// TestBindFunctionAsFunctionInputArgument tests to bind
// a whole dynamic function based on the current context
// as an input argument in the mvc-like handler's function.
func TestBindFunctionAsFunctionInputArgument(t *testing.T) {
	app := iris.New()
	postsBinder := func(ctx iris.Context) func(string) string {
		return ctx.PostValue // or FormValue, the same here.
	}

	h := MustMakeHandler(func(get func(string) string) string {
		// send the `ctx.PostValue/FormValue("username")` value
		// to the client.
		return get("username")
	},
		// bind the function binder.
		reflect.ValueOf(postsBinder))

	app.Post("/", h)

	e := httptest.New(t, app)

	expectedUsername := "kataras"
	e.POST("/").WithFormField("username", expectedUsername).
		Expect().Status(iris.StatusOK).Body().Equal(expectedUsername)
}
