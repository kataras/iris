package mvc2_test

// black-box

import (
	"fmt"
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/httptest"
	. "github.com/kataras/iris/mvc2"
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
	testService interface {
		Say(string) string
	}
	testServiceImpl struct {
		prefix string
	}
)

func (s *testServiceImpl) Say(message string) string {
	return s.prefix + " " + message
}

func TestMakeHandler(t *testing.T) {
	binders := []*InputBinder{
		// #1
		MustMakeFuncInputBinder(testBinderFunc),
		// #2
		MustMakeServiceInputBinder(&testServiceImpl{prefix: "say"}),
		// #3
		MustMakeFuncInputBinder(func(ctx iris.Context) string {
			return ctx.Params().Get("param")
		}),
	}

	var (
		// a context as first input arg, which is not needed to be binded manually,
		// and a user struct which is binded to the input arg by the #1 func(ctx) any binder.
		consumeUserHandler = func(ctx iris.Context, user testUserStruct) {
			ctx.JSON(user)
		}
		h1 = MustMakeHandler(consumeUserHandler, binders)

		// just one input arg, the service which is binded by the #2 service binder.
		consumeServiceHandler = func(service testService) string {
			return service.Say("something")
		}
		h2 = MustMakeHandler(consumeServiceHandler, binders)

		// just one input arg, a standar string which is binded by the #3 func(ctx) any binder.
		consumeParamHandler = func(myParam string) string {
			return "param is: " + myParam
		}
		h3 = MustMakeHandler(consumeParamHandler, binders)
	)

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
