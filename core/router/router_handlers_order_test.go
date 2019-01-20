// black-box testing
//
// see _examples/routing/main_test.go for the most common router tests that you may want to see,
// this is a test which makes sure that the APIBuilder's `UseGlobal`, `Use` and `Done` functions are
// working as expected.

package router_test

import (
	"testing"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"

	"github.com/kataras/iris/httptest"
)

// test registering of below handlers
// with a different order but the route's final
// response should be the same at all cases.
var (
	mainResponse = "main"
	mainHandler  = func(ctx context.Context) {
		ctx.WriteString(mainResponse)
		ctx.Next()
	}

	firstUseResponse = "use1"
	firstUseHandler  = func(ctx context.Context) {
		ctx.WriteString(firstUseResponse)
		ctx.Next()
	}

	secondUseResponse = "use2"
	secondUseHandler  = func(ctx context.Context) {
		ctx.WriteString(secondUseResponse)
		ctx.Next()
	}

	firstUseGlobalResponse = "useglobal1"
	firstUseGlobalHandler  = func(ctx context.Context) {
		ctx.WriteString(firstUseGlobalResponse)
		ctx.Next()
	}

	secondUseGlobalResponse = "useglobal2"
	secondUseGlobalHandler  = func(ctx context.Context) {
		ctx.WriteString(secondUseGlobalResponse)
		ctx.Next()
	}

	firstDoneResponse = "done1"
	firstDoneHandler  = func(ctx context.Context) {
		ctx.WriteString(firstDoneResponse)
		ctx.Next()
	}

	secondDoneResponse = "done2"
	secondDoneHandler  = func(ctx context.Context) {
		ctx.WriteString(secondDoneResponse)
	}

	finalResponse = firstUseGlobalResponse + secondUseGlobalResponse +
		firstUseResponse + secondUseResponse + mainResponse + firstDoneResponse + secondDoneResponse

	testResponse = func(t *testing.T, app *iris.Application, path string) {
		e := httptest.New(t, app)
		e.GET(path).Expect().Status(httptest.StatusOK).Body().Equal(finalResponse)
	}
)

func TestMiddlewareByRouteDef(t *testing.T) {
	app := iris.New()
	app.Get("/mypath", firstUseGlobalHandler, secondUseGlobalHandler, firstUseHandler, secondUseHandler,
		mainHandler, firstDoneHandler, secondDoneHandler)

	testResponse(t, app, "/mypath")
}
func TestMiddlewareByUseAndDoneDef(t *testing.T) {
	app := iris.New()
	app.Use(firstUseGlobalHandler, secondUseGlobalHandler, firstUseHandler, secondUseHandler)
	app.Done(firstDoneHandler, secondDoneHandler)

	app.Get("/mypath", mainHandler)

	testResponse(t, app, "/mypath")
}

func TestMiddlewareByUseUseGlobalAndDoneDef(t *testing.T) {
	app := iris.New()
	app.Use(firstUseHandler, secondUseHandler)
	// if failed then UseGlobal didnt' registered these handlers even before the
	// existing middleware.
	app.UseGlobal(firstUseGlobalHandler, secondUseGlobalHandler)
	app.Done(firstDoneHandler, secondDoneHandler)

	app.Get("/mypath", mainHandler)

	testResponse(t, app, "/mypath")
}

func TestMiddlewareByUseDoneAndUseGlobalDef(t *testing.T) {
	app := iris.New()

	app.Use(firstUseHandler, secondUseHandler)
	app.Done(firstDoneHandler, secondDoneHandler)

	app.Get("/mypath", mainHandler)

	// if failed then UseGlobal was unable to
	// prepend these handlers to the route was registered before
	// OR
	// when order failed because these should be executed in order, first the firstUseGlobalHandler,
	// because they are the same type (global begin handlers)
	app.UseGlobal(firstUseGlobalHandler)
	app.UseGlobal(secondUseGlobalHandler)

	testResponse(t, app, "/mypath")
}

func TestMiddlewareByUseGlobalUseAndDoneGlobalDef(t *testing.T) {
	app := iris.New()

	app.UseGlobal(firstUseGlobalHandler)
	app.UseGlobal(secondUseGlobalHandler)
	app.Use(firstUseHandler, secondUseHandler)

	app.Get("/mypath", mainHandler)

	app.DoneGlobal(firstDoneHandler, secondDoneHandler)

	testResponse(t, app, "/mypath")
}

func TestMiddlewareByDoneUseAndUseGlobalDef(t *testing.T) {
	app := iris.New()
	app.Done(firstDoneHandler, secondDoneHandler)

	app.Use(firstUseHandler, secondUseHandler)

	app.Get("/mypath", mainHandler)

	app.UseGlobal(firstUseGlobalHandler)
	app.UseGlobal(secondUseGlobalHandler)

	testResponse(t, app, "/mypath")
}
