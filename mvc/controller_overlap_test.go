// black-box testing
// Note: there is a test, for end-devs, of Controllers overlapping at _examples/mvc/authenticated-controller too.
package mvc_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/kataras/iris/v12/mvc"
)

func TestControllerOverlap(t *testing.T) {
	app := iris.New()
	userRouter := app.Party("/user")
	{
		userRouter.SetRegisterRule(iris.RouteOverlap)

		// Initialize a new MVC application on top of the "userRouter".
		userApp := mvc.New(userRouter)
		// Register Dependencies.
		userApp.Register(authDependency)

		// Register Controllers.
		userApp.Handle(new(AuthenticatedUserController))
		userApp.Handle(new(UnauthenticatedUserController))
	}

	e := httptest.New(t, app)
	e.GET("/user").Expect().Status(httptest.StatusUnauthorized).Body().IsEqual("unauth")
	// Test raw stop execution with a status code sent on the controller's method.
	e.GET("/user/with/status/on/method").Expect().Status(httptest.StatusBadRequest).Body().IsEqual("unauth")
	// Test stop execution with status but last code sent through the controller's method.
	e.GET("/user/with/status/on/method/too").Expect().Status(httptest.StatusInternalServerError).Body().IsEqual("unauth")
	// Test raw stop execution and no status code sent on controller's method (should be OK).
	e.GET("/user/with/no/status").Expect().Status(httptest.StatusOK).Body().IsEqual("unauth")

	// Test authenticated request.
	e.GET("/user").WithQuery("id", 42).Expect().Status(httptest.StatusOK).Body().IsEqual("auth: 42")

	// Test HandleHTTPError method accepts a not found and returns a 404
	// from a shared controller and overlapped, the url parameter matters because this method was overlapped.
	e.GET("/user/notfound").Expect().Status(httptest.StatusBadRequest).
		Body().IsEqual("error: *mvc_test.UnauthenticatedUserController: from: 404 to: 400")
	e.GET("/user/notfound").WithQuery("id", 42).Expect().Status(httptest.StatusBadRequest).
		Body().IsEqual("error: *mvc_test.AuthenticatedUserController: from: 404 to: 400")
}

type AuthenticatedTest uint64

func authDependency(ctx iris.Context) AuthenticatedTest {
	// this will be executed on not found too and that's what we expect.

	userID := ctx.URLParamUint64("id") // just for the test.
	if userID == 0 {
		if ctx.GetStatusCode() == iris.StatusNotFound || // do not send 401 on not founds, keep 404 and let controller decide.
			ctx.Path() == "/user/with/status/on/method" || ctx.Path() == "/user/with/np/status" { // leave controller method decide, raw stop execution.
			ctx.StopExecution()
		} else {
			ctx.StopWithStatus(iris.StatusUnauthorized)
		}

		return 0
	}

	return AuthenticatedTest(userID)
}

type BaseControllerTest struct{}

func (c *BaseControllerTest) HandleHTTPError(ctx iris.Context, code mvc.Code) (string, int) {
	if ctx.GetStatusCode() != int(code) {
		// should never happen.
		panic("Context current status code and given mvc code do not match!")
	}

	ctrlName := ctx.Controller().Type().String()
	newCode := 400
	return fmt.Sprintf("error: %s: from: %d to: %d", ctrlName, int(code), newCode), newCode
}

type UnauthenticatedUserController struct {
	BaseControllerTest
}

func (c *UnauthenticatedUserController) Get() string {
	return "unauth"
}

func (c *UnauthenticatedUserController) GetWithNoStatus() string {
	return "unauth"
}

func (c *UnauthenticatedUserController) GetWithStatusOnMethod() (string, int) {
	return "unauth", iris.StatusBadRequest
}

func (c *UnauthenticatedUserController) GetWithStatusOnMethodToo() (string, int) {
	return "unauth", iris.StatusInternalServerError
}

type AuthenticatedUserController struct {
	BaseControllerTest

	CurrentUserID AuthenticatedTest
}

func (c *AuthenticatedUserController) Get() string {
	return fmt.Sprintf("auth: %d", c.CurrentUserID)
}
